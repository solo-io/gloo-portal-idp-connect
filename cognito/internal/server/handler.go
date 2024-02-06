package server

import (
	"context"
	"errors"

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

	DescribeUserPoolClient(
		ctx context.Context,
		params *cognito.DescribeUserPoolClientInput,
		optFns ...func(*cognito.Options),
	) (*cognito.DescribeUserPoolClientOutput, error)

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

	cognitoClient CognitoClient
}

const AccessResourceServerName = "access"

func NewStrictServerHandler(opts *Options, cognitoClient CognitoClient) *StrictServerHandler {
	return &StrictServerHandler{
		userPool:      opts.CognitoUserPool,
		cognitoClient: cognitoClient,
	}
}

// DeleteClient deletes a client in the OpenId Connect Provider
func (s *StrictServerHandler) DeleteClient(
	ctx context.Context,
	request portalv1.DeleteClientRequestObject,
) (portalv1.DeleteClientResponseObject, error) {
	_, err := s.cognitoClient.DeleteUserPoolClient(ctx, &cognito.DeleteUserPoolClientInput{
		UserPoolId: &s.userPool,
		ClientId:   aws.String(request.Params.Id),
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

// GetClient gets a client from the OpenId Connect Provider
func (s *StrictServerHandler) GetClient(
	ctx context.Context,
	request portalv1.GetClientRequestObject,
) (portalv1.GetClientResponseObject, error) {
	out, err := s.cognitoClient.DescribeUserPoolClient(ctx, &cognito.DescribeUserPoolClientInput{
		UserPoolId: &s.userPool,
		ClientId:   aws.String(request.Params.Id),
	})

	if err != nil {
		cognitoErr := unwrapCognitoError(err)
		return errResponseFactory[portalv1.GetClientResponseObject](
			cognitoErr,
			map[int]portalv1.GetClientResponseObject{
				404: portalv1.GetClient404JSONResponse(cognitoErr),
			},
			portalv1.GetClient500JSONResponse(cognitoErr),
		), nil
	}

	return portalv1.GetClient200JSONResponse{
		ClientName: out.UserPoolClient.ClientName,
		ClientId:   &request.Params.Id,
	}, nil
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

	return portalv1.CreateClient200JSONResponse{
		ClientId:     out.UserPoolClient.ClientId,
		ClientSecret: out.UserPoolClient.ClientSecret,
		ClientName:   aws.String(request.Body.ClientName),
	}, nil
}

// DeleteClientScope deletes a client in the OpenId Connect Provider
func (s *StrictServerHandler) DeleteClientScope(
	ctx context.Context,
	request portalv1.DeleteClientScopeRequestObject,
) (portalv1.DeleteClientScopeResponseObject, error) {
	out, err := s.cognitoClient.DescribeUserPoolClient(ctx, &cognito.DescribeUserPoolClientInput{
		UserPoolId: &s.userPool,
		ClientId:   aws.String(request.Params.Id),
	})

	if err != nil {
		cognitoErr := unwrapCognitoError(err)
		return errResponseFactory[portalv1.DeleteClientScopeResponseObject](
			cognitoErr,
			map[int]portalv1.DeleteClientScopeResponseObject{
				404: portalv1.DeleteClientScope404JSONResponse(cognitoErr),
			},
			portalv1.DeleteClientScope500JSONResponse(cognitoErr),
		), nil
	}

	scopeExists := false
	var updatedScopes []string
	for _, scope := range out.UserPoolClient.AllowedOAuthScopes {
		if scope == request.Params.Scope {
			scopeExists = true
			continue
		}
		updatedScopes = append(updatedScopes, scope)
	}

	if !scopeExists {
		return portalv1.DeleteClientScope404JSONResponse(newPortalError(404, "Resource Not Found", "Scope not present in client")), nil
	}

	_, err = s.cognitoClient.UpdateUserPoolClient(ctx, &cognito.UpdateUserPoolClientInput{
		UserPoolId:         &s.userPool,
		ClientId:           aws.String(request.Params.Id),
		AllowedOAuthScopes: updatedScopes,
	})

	if err != nil {
		return portalv1.DeleteClientScope500JSONResponse(unwrapCognitoError(err)), nil
	}

	return portalv1.DeleteClientScope204Response{}, nil
}

// GetClientScopes gets scopes for a client from the OpenId Connect Provider
func (s *StrictServerHandler) GetClientScopes(
	ctx context.Context,
	request portalv1.GetClientScopesRequestObject,
) (portalv1.GetClientScopesResponseObject, error) {
	out, err := s.cognitoClient.DescribeUserPoolClient(ctx, &cognito.DescribeUserPoolClientInput{
		UserPoolId: &s.userPool,
		ClientId:   aws.String(request.Params.Id),
	})

	if err != nil {
		cognitoErr := unwrapCognitoError(err)
		return errResponseFactory[portalv1.GetClientScopesResponseObject](
			cognitoErr,
			map[int]portalv1.GetClientScopesResponseObject{
				404: portalv1.GetClientScopes404JSONResponse(cognitoErr),
			},
			portalv1.GetClientScopes500JSONResponse(cognitoErr),
		), nil
	}

	return portalv1.GetClientScopes200JSONResponse{
		Scopes: out.UserPoolClient.AllowedOAuthScopes,
	}, nil
}

// AddClientScope adds scope to a client in the OpenId Connect Provider
func (s *StrictServerHandler) AddClientScope(
	ctx context.Context,
	request portalv1.AddClientScopeRequestObject,
) (portalv1.AddClientScopeResponseObject, error) {
	out, err := s.cognitoClient.DescribeUserPoolClient(ctx, &cognito.DescribeUserPoolClientInput{
		UserPoolId: &s.userPool,
		ClientId:   aws.String(request.Body.Id),
	})

	if err != nil {
		cognitoErr := unwrapCognitoError(err)
		return errResponseFactory[portalv1.AddClientScopeResponseObject](
			cognitoErr,
			map[int]portalv1.AddClientScopeResponseObject{
				404: portalv1.AddClientScope404JSONResponse(cognitoErr),
			},
			portalv1.AddClientScope500JSONResponse(cognitoErr),
		), nil
	}

	inScope := request.Body.Scope

	userScopes := out.UserPoolClient.AllowedOAuthScopes
	for _, scope := range userScopes {
		if scope == inScope {
			return portalv1.AddClientScope409JSONResponse(newPortalError(409, "Resource Exists", "scope already exists")), nil
		}
	}

	userScopes = append(userScopes, inScope)

	_, err = s.cognitoClient.UpdateUserPoolClient(ctx, &cognito.UpdateUserPoolClientInput{
		UserPoolId:         &s.userPool,
		ClientId:           aws.String(request.Body.Id),
		AllowedOAuthScopes: userScopes,
	})

	if err != nil {
		return portalv1.AddClientScope500JSONResponse(unwrapCognitoError(err)), nil
	}

	return portalv1.AddClientScope204Response{}, nil
}

// DeleteScope deletes scopes in the OpenId Connect Provider
func (s *StrictServerHandler) DeleteScope(
	ctx context.Context,
	request portalv1.DeleteScopeRequestObject,
) (portalv1.DeleteScopeResponseObject, error) {
	out, err := s.cognitoClient.DescribeResourceServer(ctx, &cognito.DescribeResourceServerInput{
		UserPoolId: &s.userPool,
		Identifier: aws.String(AccessResourceServerName),
	})

	scopeExists := false
	var updatedScopes []types.ResourceServerScopeType
	for _, scope := range out.ResourceServer.Scopes {
		if scope.ScopeName == nil {
			continue
		}

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
		Identifier: aws.String(AccessResourceServerName),
		Name:       aws.String(AccessResourceServerName),
		Scopes:     updatedScopes,
	})

	if err != nil {
		return portalv1.DeleteScope500JSONResponse(unwrapCognitoError(err)), nil
	}

	return portalv1.DeleteScope204Response{}, nil
}

// GetScopes creates scopes in the OpenId Connect Provider
func (s *StrictServerHandler) GetScopes(
	ctx context.Context,
	_ portalv1.GetScopesRequestObject,
) (portalv1.GetScopesResponseObject, error) {
	out, err := s.cognitoClient.DescribeResourceServer(ctx, &cognito.DescribeResourceServerInput{
		UserPoolId: &s.userPool,
		Identifier: aws.String(AccessResourceServerName),
	})

	if err != nil {
		var notFoundErr *types.ResourceNotFoundException
		if ok := errors.As(err, &notFoundErr); ok {
			if err = createResourceServer(ctx, s); err != nil {
				return portalv1.GetScopes500JSONResponse(newPortal500Error(err.Error())), nil
			}
			return portalv1.GetScopes200JSONResponse{}, nil
		}

		return portalv1.GetScopes500JSONResponse(unwrapCognitoError(err)), nil
	}

	scopes := cognitoScopesToAPIScopesType(out.ResourceServer.Scopes...)

	return portalv1.GetScopes200JSONResponse{
		Scopes: scopes,
	}, nil
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
		Identifier: aws.String(AccessResourceServerName),
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
		Identifier: aws.String(AccessResourceServerName),
		Name:       aws.String(AccessResourceServerName),
		Scopes:     cognitoScopes,
	})

	return portalv1.CreateScope204Response{}, nil
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
		Identifier: aws.String(AccessResourceServerName),
		Name:       aws.String(AccessResourceServerName),
	})

	return err
}

func errResponseFactory[respIface interface{}](err portalv1.Error, respMap map[int]respIface, def respIface) respIface {
	if valErr, ok := respMap[err.Code]; ok {
		return valErr
	}
	return def
}
