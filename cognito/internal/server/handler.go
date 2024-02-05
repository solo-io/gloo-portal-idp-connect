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

type strictServerHandler struct {
	userPool string

	cognitoClient *cognito.Client
}

const accessResourceServerName = "access"

func newStrictServerHandler(opts *Options, cognitoClient *cognito.Client) *strictServerHandler {
	return &strictServerHandler{
		userPool:      opts.cognitoUserPool,
		cognitoClient: cognitoClient,
	}
}

// DeleteClient deletes a client in the OpenId Connect Provider
func (s *strictServerHandler) DeleteClient(
	ctx context.Context,
	request portalv1.DeleteClientRequestObject,
) (portalv1.DeleteClientResponseObject, error) {
	_, err := s.cognitoClient.DeleteUserPoolClient(ctx, &cognito.DeleteUserPoolClientInput{
		UserPoolId: &s.userPool,
		ClientId:   aws.String(request.Params.Id),
	})

	if err != nil {
		return portalv1.DeleteClient500JSONResponse(unwrapCognitoError(err)), nil
	}

	return portalv1.DeleteClient204Response{}, nil
}

// GetClient gets a client from the OpenId Connect Provider
func (s *strictServerHandler) GetClient(
	ctx context.Context,
	request portalv1.GetClientRequestObject,
) (portalv1.GetClientResponseObject, error) {
	out, err := s.cognitoClient.DescribeUserPoolClient(ctx, &cognito.DescribeUserPoolClientInput{
		UserPoolId: &s.userPool,
		ClientId:   aws.String(request.Params.Id),
	})

	if err != nil {
		return portalv1.GetClient500JSONResponse(unwrapCognitoError(err)), nil
	}

	return portalv1.GetClient200JSONResponse{
		"ClientName": *out.UserPoolClient.ClientName,
		"ClientId":   *out.UserPoolClient.ClientId,
	}, nil
}

// CreateClient creates a client in the OpenId Connect Provider
func (s *strictServerHandler) CreateClient(
	ctx context.Context,
	request portalv1.CreateClientRequestObject,
) (portalv1.CreateClientResponseObject, error) {
	if request.Body == nil {
		return portalv1.CreateClient500JSONResponse(newPortal500Error("request body is required")), nil
	}

	bodyParams := request.Body
	if bodyParams.GlooPortalUserId == nil && bodyParams.GlooPortalTeamId == nil {
		return portalv1.CreateClient500JSONResponse(newPortal500Error("either glooPortalUserId or glooPortalTeamId is required")), nil
	}

	var clientName string
	if request.Body.Passthrough != nil && (*request.Body.Passthrough)["ClientName"] != nil {
		clientName = (*request.Body.Passthrough)["ClientName"].(string)
	} else {
		if bodyParams.GlooPortalUserId != nil && bodyParams.GlooPortalTeamId != nil {
			clientName = *bodyParams.GlooPortalUserId + "-" + *bodyParams.GlooPortalTeamId
		} else if bodyParams.GlooPortalUserId != nil {
			clientName = *bodyParams.GlooPortalUserId
		} else {
			clientName = *bodyParams.GlooPortalTeamId
		}
	}

	clientName = shortenName(clientName)

	out, err := s.cognitoClient.CreateUserPoolClient(ctx, &cognito.CreateUserPoolClientInput{
		UserPoolId:     &s.userPool,
		ClientName:     aws.String(clientName),
		GenerateSecret: true,
	})

	if err != nil {
		return portalv1.CreateClient500JSONResponse(unwrapCognitoError(err)), nil
	}

	return portalv1.CreateClient200JSONResponse{
		ClientId:     out.UserPoolClient.ClientId,
		ClientSecret: out.UserPoolClient.ClientSecret,
		ClientName:   aws.String(clientName),
	}, nil
}

// DeleteClientScope deletes a client in the OpenId Connect Provider
func (s *strictServerHandler) DeleteClientScope(
	ctx context.Context,
	request portalv1.DeleteClientScopeRequestObject,
) (portalv1.DeleteClientScopeResponseObject, error) {
	// TODO: implement DeleteClientScope
	return portalv1.DeleteClientScope204Response{}, nil
}

// GetClientScopes gets scopes for a client from the OpenId Connect Provider
func (s *strictServerHandler) GetClientScopes(
	ctx context.Context,
	request portalv1.GetClientScopesRequestObject,
) (portalv1.GetClientScopesResponseObject, error) {
	// TODO: implement GetClientScopes
	return portalv1.GetClientScopes200JSONResponse{}, nil
}

// AddClientScope adds scope to a client in the OpenId Connect Provider
func (s *strictServerHandler) AddClientScope(
	ctx context.Context,
	request portalv1.AddClientScopeRequestObject,
) (portalv1.AddClientScopeResponseObject, error) {
	out, err := s.cognitoClient.DescribeUserPoolClient(ctx, &cognito.DescribeUserPoolClientInput{
		UserPoolId: &s.userPool,
		ClientId:   aws.String(request.Body.Id),
	})

	if err != nil {
		return portalv1.AddClientScope500JSONResponse(unwrapCognitoError(err)), nil
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
func (s *strictServerHandler) DeleteScope(
	ctx context.Context,
	request portalv1.DeleteScopeRequestObject,
) (portalv1.DeleteScopeResponseObject, error) {
	out, err := s.cognitoClient.DescribeResourceServer(ctx, &cognito.DescribeResourceServerInput{
		UserPoolId: &s.userPool,
		Identifier: aws.String(accessResourceServerName),
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

	// TODO: test that we modify state correctly here
	if !scopeExists {
		// Return early as if scope was deleted even if it doesn't exist, since resultant state is the same.
		return portalv1.DeleteScope204Response{}, nil
	}

	_, err = s.cognitoClient.UpdateResourceServer(ctx, &cognito.UpdateResourceServerInput{
		UserPoolId: &s.userPool,
		Identifier: aws.String(accessResourceServerName),
		Name:       aws.String(accessResourceServerName),
		Scopes:     updatedScopes,
	})

	if err != nil {
		return portalv1.DeleteScope500JSONResponse(unwrapCognitoError(err)), nil
	}

	return portalv1.DeleteScope204Response{}, nil
}

// GetScopes creates scopes in the OpenId Connect Provider
func (s *strictServerHandler) GetScopes(
	ctx context.Context,
	_ portalv1.GetScopesRequestObject,
) (portalv1.GetScopesResponseObject, error) {
	out, err := s.cognitoClient.DescribeResourceServer(ctx, &cognito.DescribeResourceServerInput{
		UserPoolId: &s.userPool,
		Identifier: aws.String(accessResourceServerName),
	})

	if err != nil {
		return portalv1.GetScopes500JSONResponse(unwrapCognitoError(err)), nil
	}

	scopes := cognitoScopesToAPIScopesType(out.ResourceServer.Scopes...)

	return portalv1.GetScopes200JSONResponse{
		Scopes: scopes,
	}, nil
}

// CreateScope creates scopes in the OpenId Connect Provider
func (s *strictServerHandler) CreateScope(
	ctx context.Context,
	request portalv1.CreateScopeRequestObject,
) (portalv1.CreateScopeResponseObject, error) {
	if request.Body == nil {
		return portalv1.CreateScope500JSONResponse(newPortal500Error("request body is required")), nil
	}

	cognitoScope := apiScopesToCognitoScopeType(request.Body.Scope)

	out, err := s.cognitoClient.DescribeResourceServer(ctx, &cognito.DescribeResourceServerInput{
		UserPoolId: &s.userPool,
		Identifier: aws.String(accessResourceServerName),
	})

	if err != nil {
		// TODO: create if it doesn't exist
		// _, err := s.cognitoClient.CreateResourceServer(ctx, &cognito.CreateResourceServerInput{
		// 	UserPoolId: &s.userPool,
		// 	Identifier: aws.String(accessResourceServerName),
		// 	Name:       aws.String(accessResourceServerName),
		// })
		return portalv1.CreateScope500JSONResponse(unwrapCognitoError(err)), nil
	}

	cognitoScopes := out.ResourceServer.Scopes
	for _, scope := range cognitoScopes {
		if *scope.ScopeName == *cognitoScope.ScopeName {
			return portalv1.CreateScope409JSONResponse(newPortalError(409, "Resource Exists", "scope already exists")), nil
		}
	}

	cognitoScopes = append(cognitoScopes, cognitoScope)

	_, err = s.cognitoClient.UpdateResourceServer(ctx, &cognito.UpdateResourceServerInput{
		UserPoolId: &s.userPool,
		Identifier: aws.String(accessResourceServerName),
		Name:       aws.String(accessResourceServerName),
		Scopes:     cognitoScopes,
	})

	return portalv1.CreateScope204Response{}, nil
}

func unwrapCognitoError(err error) portalv1.Error {
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
