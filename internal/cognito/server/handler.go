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

// DeleteOAuthApplication deletes an application by ID.
func (s *StrictServerHandler) DeleteOAuthApplication(
	ctx context.Context,
	request portalv1.DeleteOAuthApplicationRequestObject,
) (portalv1.DeleteOAuthApplicationResponseObject, error) {
	_, err := s.cognitoClient.DeleteUserPoolClient(ctx, &cognito.DeleteUserPoolClientInput{
		UserPoolId: &s.userPool,
		ClientId:   aws.String(request.Id),
	})

	if err != nil {
		switch cognitoErr := unwrapCognitoError(err); cognitoErr.Code {
		case 404:
			return portalv1.DeleteOAuthApplication404JSONResponse(cognitoErr), nil
		default:
			return portalv1.DeleteOAuthApplication500JSONResponse(cognitoErr), nil
		}
	}

	return portalv1.DeleteOAuthApplication204Response{}, nil
}

// CreateOAuthApplication creates a client in Cognito
func (s *StrictServerHandler) CreateOAuthApplication(
	ctx context.Context,
	request portalv1.CreateOAuthApplicationRequestObject,
) (portalv1.CreateOAuthApplicationResponseObject, error) {
	if request.Body == nil || len(request.Body.Id) == 0 {
		return portalv1.CreateOAuthApplication400JSONResponse(newPortal400Error("unique id is required")), nil
	}

	out, err := s.cognitoClient.CreateUserPoolClient(ctx, &cognito.CreateUserPoolClientInput{
		UserPoolId:     &s.userPool,
		ClientName:     aws.String(request.Body.Id),
		GenerateSecret: true,
	})

	if err != nil {
		return portalv1.CreateOAuthApplication500JSONResponse(unwrapCognitoError(err)), nil
	}

	return portalv1.CreateOAuthApplication201JSONResponse{
		ClientId:     *out.UserPoolClient.ClientId,
		ClientSecret: *out.UserPoolClient.ClientSecret,
		ClientName:   aws.String(request.Body.Id),
	}, nil
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
