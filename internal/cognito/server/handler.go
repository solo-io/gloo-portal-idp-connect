package server

import (
	"context"
	"errors"
	"fmt"

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

// DeleteClient deletes a client in Cognito
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
			switch cognitoErr := unwrapCognitoError(err); cognitoErr.Code {
			case 404:
				return portalv1.DeleteClient404JSONResponse(cognitoErr), nil
			default:
				return portalv1.DeleteClient500JSONResponse(cognitoErr), nil
			}
		}
	}

	return portalv1.DeleteClient204Response{}, nil
}

// CreateClient creates a client in Cognito
func (s *StrictServerHandler) CreateClient(
	ctx context.Context,
	request portalv1.CreateClientRequestObject,
) (portalv1.CreateClientResponseObject, error) {
	if request.Body == nil {
		return portalv1.CreateClient400JSONResponse(newPortal400Error("request body is required")), nil
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

// UpdateClientAPIProducts updates scopes for a client in Cognito.
func (s *StrictServerHandler) UpdateClientAPIProducts(
	ctx context.Context,
	request portalv1.UpdateClientAPIProductsRequestObject,
) (portalv1.UpdateClientAPIProductsResponseObject, error) {
	if request.Body == nil {
		return portalv1.UpdateClientAPIProducts400JSONResponse(newPortal400Error("request body is required")), nil
	}

	var cognitoScopes []string
	for _, apiProduct := range request.Body.ApiProducts {
		cognitoScopes = append(cognitoScopes, fmt.Sprintf("%s/%s", s.resourceServer, apiProduct))
	}

	clientInput := &cognito.UpdateUserPoolClientInput{
		UserPoolId:         &s.userPool,
		ClientId:           &request.Id,
		AllowedOAuthScopes: cognitoScopes,
	}
	if len(cognitoScopes) != 0 {
		clientInput.AllowedOAuthFlowsUserPoolClient = true
		clientInput.AllowedOAuthFlows = []types.OAuthFlowType{
			types.OAuthFlowTypeClientCredentials,
		}
	}

	_, err := s.cognitoClient.UpdateUserPoolClient(ctx, clientInput)

	if err != nil {
		switch cognitoErr := unwrapCognitoError(err); cognitoErr.Code {
		case 404:
			return portalv1.UpdateClientAPIProducts404JSONResponse(cognitoErr), nil
		default:
			return portalv1.UpdateClientAPIProducts500JSONResponse(cognitoErr), nil
		}
	}

	return portalv1.UpdateClientAPIProducts204Response{}, nil
}

// DeleteAPIProduct deletes scopes in Cognito
func (s *StrictServerHandler) DeleteAPIProduct(
	ctx context.Context,
	request portalv1.DeleteAPIProductRequestObject,
) (portalv1.DeleteAPIProductResponseObject, error) {
	out, err := s.cognitoClient.DescribeResourceServer(ctx, &cognito.DescribeResourceServerInput{
		UserPoolId: &s.userPool,
		Identifier: aws.String(s.resourceServer),
	})
	if err != nil {
		switch cognitoErr := unwrapCognitoError(err); cognitoErr.Code {
		case 404:
			return portalv1.DeleteAPIProduct404JSONResponse(cognitoErr), nil
		default:
			return portalv1.DeleteAPIProduct500JSONResponse(cognitoErr), nil
		}
	}

	scopeExists := false
	var updatedScopes []types.ResourceServerScopeType
	for _, scope := range out.ResourceServer.Scopes {
		if scope.ScopeName == nil {
			continue
		}

		if *scope.ScopeName == request.ApiProduct {
			scopeExists = true
			continue
		}

		updatedScopes = append(updatedScopes, scope)
	}

	if !scopeExists {
		// Return early as if scope was deleted even if it doesn't exist, since resultant state is the same.
		return portalv1.DeleteAPIProduct404JSONResponse{}, nil
	}

	_, err = s.cognitoClient.UpdateResourceServer(ctx, &cognito.UpdateResourceServerInput{
		UserPoolId: &s.userPool,
		Identifier: aws.String(s.resourceServer),
		Name:       aws.String(s.resourceServer),
		Scopes:     updatedScopes,
	})
	if err != nil {
		return portalv1.DeleteAPIProduct500JSONResponse(unwrapCognitoError(err)), nil
	}

	return portalv1.DeleteAPIProduct204Response{}, nil
}

// CreateAPIProduct creates scopes in Cognito
func (s *StrictServerHandler) CreateAPIProduct(
	ctx context.Context,
	request portalv1.CreateAPIProductRequestObject,
) (portalv1.CreateAPIProductResponseObject, error) {
	if request.Body == nil {
		return portalv1.CreateAPIProduct400JSONResponse(newPortal400Error("request body is required")), nil
	}

	if request.Body.ApiProduct.Description == nil {
		request.Body.ApiProduct.Description = aws.String(request.Body.ApiProduct.Name)
	}

	out, err := s.cognitoClient.DescribeResourceServer(ctx, &cognito.DescribeResourceServerInput{
		UserPoolId: &s.userPool,
		Identifier: aws.String(s.resourceServer),
	})

	if err != nil {
		var notFoundErr *types.ResourceNotFoundException
		if !errors.As(err, &notFoundErr) {
			return portalv1.CreateAPIProduct500JSONResponse(unwrapCognitoError(err)), nil
		}

		// If Resource Server does not exist, create it.
		if err = createResourceServer(ctx, s); err != nil {
			return portalv1.CreateAPIProduct500JSONResponse(newPortal500Error(err.Error())), nil
		}
	}

	var cognitoScopes []types.ResourceServerScopeType
	if out != nil {
		cognitoScopes = out.ResourceServer.Scopes
	}

	inScope := apiProductToCognitoScopeType(request.Body.ApiProduct)
	for _, scope := range cognitoScopes {
		if *scope.ScopeName == *inScope.ScopeName {
			return portalv1.CreateAPIProduct409JSONResponse(newPortalError(409, "Resource Exists", "scope already exists")), nil
		}
	}

	cognitoScopes = append(cognitoScopes, inScope)

	_, err = s.cognitoClient.UpdateResourceServer(ctx, &cognito.UpdateResourceServerInput{
		UserPoolId: &s.userPool,
		Identifier: aws.String(s.resourceServer),
		Name:       aws.String(s.resourceServer),
		Scopes:     cognitoScopes,
	})
	if err != nil {
		return portalv1.CreateAPIProduct500JSONResponse(unwrapCognitoError(err)), nil
	}

	return portalv1.CreateAPIProduct201Response{}, nil
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
