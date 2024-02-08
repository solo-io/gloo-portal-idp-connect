package server

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	cognito "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	"github.com/aws/smithy-go/transport/http"

	portalv1 "github.com/solo-io/gloo-portal-idp-connect/pkg/api/v1"
)

//go:generate mockgen -destination=mock/cognito_client.go . CognitoClient

type CognitoClient interface {
	DeleteUserPoolClient(
		ctx context.Context,
		params *cognito.DeleteUserPoolClientInput,
		optFns ...func(*cognito.Options),
	) (*cognito.DeleteUserPoolClientOutput, error)

	CreateUserPoolClient(
		ctx context.Context,
		params *cognito.CreateUserPoolClientInput,
		optFns ...func(*cognito.Options),
	) (*cognito.CreateUserPoolClientOutput, error)

	UpdateUserPoolClient(
		ctx context.Context,
		params *cognito.UpdateUserPoolClientInput,
		optFns ...func(*cognito.Options),
	) (*cognito.UpdateUserPoolClientOutput, error)

	CreateResourceServer(
		ctx context.Context,
		params *cognito.CreateResourceServerInput,
		optFns ...func(*cognito.Options),
	) (*cognito.CreateResourceServerOutput, error)

	DescribeResourceServer(
		ctx context.Context,
		params *cognito.DescribeResourceServerInput,
		optFns ...func(*cognito.Options),
	) (*cognito.DescribeResourceServerOutput, error)

	UpdateResourceServer(
		ctx context.Context,
		params *cognito.UpdateResourceServerInput,
		optFns ...func(*cognito.Options),
	) (*cognito.UpdateResourceServerOutput, error)
}

type StrictServerHandler struct {
	userPool string

	cognitoClient  CognitoClient
	resourceServer string
}

func NewStrictServerHandler(opts *Options, cognitoClient CognitoClient) *StrictServerHandler {
	return &StrictServerHandler{
		userPool:       opts.CognitoUserPool,
		cognitoClient:  cognitoClient,
		resourceServer: opts.ResourceServer,
	}
}

// DeleteClient deletes a client in the OpenId Connect Provider
func (s *StrictServerHandler) DeleteClient(
	ctx context.Context,
	request portalv1.DeleteClientRequestObject,
) (portalv1.DeleteClientResponseObject, error) {
	_, err := s.cognitoClient.DeleteUserPoolClient(ctx, &cognito.DeleteUserPoolClientInput{
		UserPoolId: &s.userPool,
		ClientId:   aws.String(request.Id),
	})

	if err != nil {
		if err != nil {
			cognitoErr := unwrapCognitoError(err)
			return errResponseFactory[portalv1.DeleteClientResponseObject](
				cognitoErr,
				map[int]portalv1.DeleteClientResponseObject{
					404: portalv1.DeleteClient404JSONResponse(cognitoErr),
				},
				portalv1.DeleteClient500JSONResponse(cognitoErr),
			), nil
		}
	}

	return portalv1.DeleteClient204Response{}, nil
}

// CreateClient creates a client in the OpenId Connect Provider
func (s *StrictServerHandler) CreateClient(
	ctx context.Context,
	request portalv1.CreateClientRequestObject,
) (portalv1.CreateClientResponseObject, error) {
	if request.Body == nil {
		return portalv1.CreateClient500JSONResponse(newPortal500Error("request body is required")), nil
	}

	out, err := s.cognitoClient.CreateUserPoolClient(ctx, &cognito.CreateUserPoolClientInput{
		UserPoolId:     &s.userPool,
		ClientName:     aws.String(request.Body.ClientName),
		GenerateSecret: true,
	})

	if err != nil {
		return portalv1.CreateClient500JSONResponse(unwrapCognitoError(err)), nil
	}

	return portalv1.CreateClient201JSONResponse{
		ClientId:     out.UserPoolClient.ClientId,
		ClientSecret: out.UserPoolClient.ClientSecret,
		ClientName:   aws.String(request.Body.ClientName),
	}, nil
}

// AddClientScope adds scope to a client in the OpenId Connect Provider
func (s *StrictServerHandler) UpdateClientScopes(
	ctx context.Context,
	request portalv1.UpdateClientScopesRequestObject,
) (portalv1.UpdateClientScopesResponseObject, error) {
	if request.Body == nil {
		return portalv1.UpdateClientScopes500JSONResponse(newPortal500Error("request body is required")), nil
	}

	var cognitoScopes []string
	for _, scope := range request.Body.Scopes {
		cognitoScopes = append(cognitoScopes, fmt.Sprintf("%s/%s", s.resourceServer, scope))
	}

	_, err := s.cognitoClient.UpdateUserPoolClient(ctx, &cognito.UpdateUserPoolClientInput{
		UserPoolId:         &s.userPool,
		ClientId:           &request.Id,
		AllowedOAuthScopes: cognitoScopes,
	})

	if err != nil {
		cognitoErr := unwrapCognitoError(err)
		return errResponseFactory[portalv1.UpdateClientScopesResponseObject](
			cognitoErr,
			map[int]portalv1.UpdateClientScopesResponseObject{
				404: portalv1.UpdateClientScopes404JSONResponse(cognitoErr),
			},
			portalv1.UpdateClientScopes500JSONResponse(cognitoErr),
		), nil
	}

	return portalv1.UpdateClientScopes204Response{}, nil
}

// DeleteScope deletes scopes in the OpenId Connect Provider
func (s *StrictServerHandler) DeleteScope(
	ctx context.Context,
	request portalv1.DeleteScopeRequestObject,
) (portalv1.DeleteScopeResponseObject, error) {
	out, err := s.cognitoClient.DescribeResourceServer(ctx, &cognito.DescribeResourceServerInput{
		UserPoolId: &s.userPool,
		Identifier: aws.String(s.resourceServer),
	})
	if err != nil {
		cognitoErr := unwrapCognitoError(err)
		return errResponseFactory[portalv1.DeleteScopeResponseObject](
			cognitoErr,
			map[int]portalv1.DeleteScopeResponseObject{
				404: portalv1.DeleteScope404JSONResponse(cognitoErr),
			},
			portalv1.DeleteScope500JSONResponse(cognitoErr),
		), nil
	}

	log.Printf("Params scope: %v", request.Params.Scope)

	scopeExists := false
	var updatedScopes []types.ResourceServerScopeType
	for _, scope := range out.ResourceServer.Scopes {
		if scope.ScopeName == nil {
			continue
		}
		log.Printf("Existing scope: %v", *scope.ScopeName)

		if *scope.ScopeName == request.Params.Scope {
			scopeExists = true
			continue
		}

		updatedScopes = append(updatedScopes, scope)
	}

	if !scopeExists {
		// Return early as if scope was deleted even if it doesn't exist, since resultant state is the same.
		return portalv1.DeleteScope404JSONResponse{}, nil
	}

	_, err = s.cognitoClient.UpdateResourceServer(ctx, &cognito.UpdateResourceServerInput{
		UserPoolId: &s.userPool,
		Identifier: aws.String(s.resourceServer),
		Name:       aws.String(s.resourceServer),
		Scopes:     updatedScopes,
	})
	if err != nil {
		return portalv1.DeleteScope500JSONResponse(unwrapCognitoError(err)), nil
	}

	return portalv1.DeleteScope204Response{}, nil
}

// CreateScope creates scopes in the OpenId Connect Provider
func (s *StrictServerHandler) CreateScope(
	ctx context.Context,
	request portalv1.CreateScopeRequestObject,
) (portalv1.CreateScopeResponseObject, error) {
	if request.Body == nil {
		return portalv1.CreateScope500JSONResponse(newPortal500Error("request body is required")), nil
	}

	cognitoScope := apiScopesToCognitoScopeType(request.Body.Scope)

	out, err := s.cognitoClient.DescribeResourceServer(ctx, &cognito.DescribeResourceServerInput{
		UserPoolId: &s.userPool,
		Identifier: aws.String(s.resourceServer),
	})

	var cognitoScopes []types.ResourceServerScopeType
	if err != nil {
		var notFoundErr *types.ResourceNotFoundException
		if ok := errors.As(err, &notFoundErr); ok {
			if err = createResourceServer(ctx, s); err != nil {
				return portalv1.CreateScope500JSONResponse(newPortal500Error(err.Error())), nil
			}
		} else {
			return portalv1.CreateScope500JSONResponse(unwrapCognitoError(err)), nil
		}
	} else {
		cognitoScopes = out.ResourceServer.Scopes
	}

	for _, scope := range cognitoScopes {
		if *scope.ScopeName == *cognitoScope.ScopeName {
			return portalv1.CreateScope409JSONResponse(newPortalError(409, "Resource Exists", "scope already exists")), nil
		}
	}

	cognitoScopes = append(cognitoScopes, cognitoScope)

	_, err = s.cognitoClient.UpdateResourceServer(ctx, &cognito.UpdateResourceServerInput{
		UserPoolId: &s.userPool,
		Identifier: aws.String(s.resourceServer),
		Name:       aws.String(s.resourceServer),
		Scopes:     cognitoScopes,
	})
	if err != nil {
		return portalv1.CreateScope500JSONResponse(unwrapCognitoError(err)), nil
	}

	return portalv1.CreateScope201Response{}, nil
}

func unwrapCognitoError(err error) portalv1.Error {
	var notFoundErr *types.ResourceNotFoundException
	if ok := errors.As(err, &notFoundErr); ok {
		return portalv1.Error{
			Code:    404,
			Message: "Resource Not Found",
			Reason:  notFoundErr.Error(),
		}
	}

	var respErr *http.ResponseError
	if ok := errors.As(err, &respErr); ok {
		return portalv1.Error{
			Code:    respErr.HTTPStatusCode(),
			Message: respErr.HTTPResponse().Status,
			Reason:  respErr.Err.Error(),
		}
	}

	return newPortal500Error(err.Error())
}

func createResourceServer(ctx context.Context, s *StrictServerHandler) error {
	_, err := s.cognitoClient.CreateResourceServer(ctx, &cognito.CreateResourceServerInput{
		UserPoolId: &s.userPool,
		Identifier: aws.String(s.resourceServer),
		Name:       aws.String(s.resourceServer),
	})

	return err
}

func errResponseFactory[respIface interface{}](err portalv1.Error, respMap map[int]respIface, def respIface) respIface {
	if valErr, ok := respMap[err.Code]; ok {
		return valErr
	}
	return def
}