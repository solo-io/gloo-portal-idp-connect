package server

import (
	"context"
	"errors"

	resty "github.com/go-resty/resty/v2"
	portalv1 "github.com/solo-io/gloo-portal-idp-connect/pkg/api/v1"
)

type StrictServerHandler struct {
	client               resty.Client
	registrationEndpoint string
	bearerToken          string
	resourceServer       string
}

type CreatedClient struct {
	Id     string `json:"client_id"`
	Name   string `json:"client_name"`
	Secret string `json:"client_secret"`
}

type KeycloakError struct {
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

func NewStrictServerHandler(opts *Options, restyClient *resty.Client, registrationEndpoint string) *StrictServerHandler {
	return &StrictServerHandler{
		client:               *restyClient,
		registrationEndpoint: registrationEndpoint,
		bearerToken:          opts.BearerToken,
		resourceServer:       opts.ResourceServer,
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

	var createdClient CreatedClient

	resp, err := s.client.R().
		SetAuthToken(s.bearerToken).
		SetBody(map[string]interface{}{
			"client_name": request.Body.Name,
		}).
		SetResult(&createdClient).
		SetError(&KeycloakError{}).
		Post(s.registrationEndpoint)

	if err != nil || resp.IsError() {
		return portalv1.CreateOAuthApplication500JSONResponse(unwrapError(resp, err)), nil
	}

	return portalv1.CreateOAuthApplication201JSONResponse{
		ClientId:     &createdClient.Id,
		ClientSecret: &createdClient.Secret,
		ClientName:   &createdClient.Name,
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
		switch portalErr := unwrapError(nil, err); portalErr.Code {
		case 404:
			return portalv1.DeleteApplication404JSONResponse(portalErr), nil
		default:
			return portalv1.DeleteApplication500JSONResponse(portalErr), nil
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
		switch portalErr := unwrapError(nil, err); portalErr.Code {
		case 404:
			return portalv1.UpdateAppAPIProducts404JSONResponse(portalErr), nil
		default:
			return portalv1.UpdateAppAPIProducts500JSONResponse(portalErr), nil
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
		return portalv1.CreateAPIProduct500JSONResponse(unwrapError(nil, err)), nil
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
		return portalv1.DeleteAPIProduct500JSONResponse(unwrapError(nil, err)), nil
	}

	return portalv1.DeleteAPIProduct204Response{}, nil
}

func unwrapError(resp *resty.Response, err error) portalv1.Error {
	if err == nil {
		error := resp.Error().(*KeycloakError)
		return portalv1.Error{
			Code:    resp.StatusCode(),
			Message: error.Error,
			Reason:  error.Description,
		}
	}

	var respErr *resty.ResponseError
	if ok := errors.As(err, &respErr); ok {
		return portalv1.Error{
			Code:    respErr.Response.StatusCode(),
			Message: respErr.Response.Status(),
			Reason:  respErr.Error(),
		}
	}

	return newPortal500Error(err.Error())
}
