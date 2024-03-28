package server

import (
	"context"
	"errors"

	"github.com/aws/smithy-go/transport/http"
	portalv1 "github.com/solo-io/gloo-portal-idp-connect/pkg/api/v1"
)

type StrictServerHandler struct {
	issuer         string
	resourceServer string
}

func NewStrictServerHandler(opts *Options) *StrictServerHandler {
	return &StrictServerHandler{
		issuer:         opts.Issuer,
		resourceServer: opts.ResourceServer,
	}
}

// CreateOAuthApplication creates a client in Keycloak
func (s *StrictServerHandler) CreateOAuthApplication(
	ctx context.Context,
	request portalv1.CreateOAuthApplicationRequestObject,
) (portalv1.CreateOAuthApplicationResponseObject, error) {
	if request.Body == nil || len(request.Body.Name) == 0 {
		return portalv1.CreateOAuthApplication400JSONResponse(newPortal400Error("client name is required")), nil
	}

	// DO IT
	err := errors.New("unimplemented")

	if err != nil {
		return portalv1.CreateOAuthApplication500JSONResponse(unwrapError(err)), nil
	}

	clientId := "client-id"
	clientSecret := "client-secret"
	clientName := "client-name"

	return portalv1.CreateOAuthApplication201JSONResponse{
		ClientId:     &clientId,
		ClientSecret: &clientSecret,
		ClientName:   &clientName,
	}, nil
}

// DeleteApplication deletes an application by ID.
func (s *StrictServerHandler) DeleteApplication(
	ctx context.Context,
	request portalv1.DeleteApplicationRequestObject,
) (portalv1.DeleteApplicationResponseObject, error) {

	// DO IT
	err := errors.New("unimplemented")

	if err != nil {
		switch keycloakErr := unwrapError(err); keycloakErr.Code {
		case 404:
			return portalv1.DeleteApplication404JSONResponse(keycloakErr), nil
		default:
			return portalv1.DeleteApplication500JSONResponse(keycloakErr), nil
		}
	}

	return portalv1.DeleteApplication204Response{}, nil
}

// UpdateAppAPIProducts updates scopes for a client in Keycloak.
func (s *StrictServerHandler) UpdateAppAPIProducts(
	ctx context.Context,
	request portalv1.UpdateAppAPIProductsRequestObject,
) (portalv1.UpdateAppAPIProductsResponseObject, error) {
	if request.Body == nil {
		return portalv1.UpdateAppAPIProducts400JSONResponse(newPortal400Error("request body is required")), nil
	}

	// DO IT
	err := errors.New("unimplemented")

	if err != nil {
		switch keycloakErr := unwrapError(err); keycloakErr.Code {
		case 404:
			return portalv1.UpdateAppAPIProducts404JSONResponse(keycloakErr), nil
		default:
			return portalv1.UpdateAppAPIProducts500JSONResponse(keycloakErr), nil
		}
	}

	return portalv1.UpdateAppAPIProducts204Response{}, nil
}

// CreateAPIProduct creates scopes in Keycloak
func (s *StrictServerHandler) CreateAPIProduct(
	ctx context.Context,
	request portalv1.CreateAPIProductRequestObject,
) (portalv1.CreateAPIProductResponseObject, error) {
	if request.Body == nil {
		return portalv1.CreateAPIProduct400JSONResponse(newPortal400Error("request body is required")), nil
	}

	// DO IT
	err := errors.New("unimplemented")

	if err != nil {
		return portalv1.CreateAPIProduct500JSONResponse(unwrapError(err)), nil
	}

	return portalv1.CreateAPIProduct201Response{}, nil
}

// DeleteAPIProduct deletes scopes in Keycloak
func (s *StrictServerHandler) DeleteAPIProduct(
	ctx context.Context,
	request portalv1.DeleteAPIProductRequestObject,
) (portalv1.DeleteAPIProductResponseObject, error) {

	// DO IT
	err := errors.New("unimplemented")

	if err != nil {
		return portalv1.DeleteAPIProduct500JSONResponse(unwrapError(err)), nil
	}

	return portalv1.DeleteAPIProduct204Response{}, nil
}

func unwrapError(err error) portalv1.Error {
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
