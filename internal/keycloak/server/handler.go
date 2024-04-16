package server

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	resty "github.com/go-resty/resty/v2"
	portalv1 "github.com/solo-io/gloo-portal-idp-connect/pkg/api/v1"
)

type StrictServerHandler struct {
	restClient       resty.Client
	issuer           string
	tokenEndpoint    string
	adminRoot        string
	mgmtClientId     string
	mgmtClientSecret string
	resourceServer   string
}

type KeycloakToken struct {
	AccessToken string `json:"access_token"`
}

type CreatedClient struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Secret string `json:"secret"`
}

type KeycloakError struct {
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

func NewStrictServerHandler(opts *Options, restyClient *resty.Client, tokenEndpoint string) *StrictServerHandler {
	r := regexp.MustCompile("^(https?:.*?)/realms/(.[^/]*)/?$")
	adminRoot := r.ReplaceAllString(opts.Issuer, "$1/admin/realms/$2")

	restyClient.OnBeforeRequest(func(c *resty.Client, r *resty.Request) error {
		// If we already have user info, assume this is a request to fetch a token
		if r.UserInfo != nil {
			return nil
		}

		var token KeycloakToken

		tokenResponse, err := c.R().
			SetBasicAuth(opts.MgmtClientId, opts.MgmtClientSecret).
			SetFormData(map[string]string{
				"grant_type": "urn:ietf:params:oauth:grant-type:uma-ticket",
				"audience":   opts.MgmtClientId,
			}).
			SetResult(&token).
			SetError(&KeycloakError{}).
			Post(tokenEndpoint)

		if err != nil {
			return err
		}

		if tokenResponse.IsError() {
			error := tokenResponse.Error().(*KeycloakError)
			return fmt.Errorf("could not obtain token for client %s: [%s] %s", opts.MgmtClientId, error.Error, error.Description)
		}

		r.SetAuthToken(token.AccessToken)
		r.SetError(&KeycloakError{})

		return nil
	})

	return &StrictServerHandler{
		restClient:       *restyClient,
		issuer:           opts.Issuer,
		tokenEndpoint:    tokenEndpoint,
		adminRoot:        adminRoot,
		mgmtClientId:     opts.MgmtClientId,
		mgmtClientSecret: opts.MgmtClientSecret,
		resourceServer:   opts.ResourceServer,
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

	resp, err := s.restClient.R().
		SetBody(map[string]interface{}{
			"clientId": request.Body.Name,
			"name":     request.Body.Name,
			"authorizationServicesEnabled": true,
		}).
		SetResult(&createdClient).
		Post(s.issuer + "/clients-registrations/default")

	if err != nil || resp.IsError() {
		return portalv1.CreateOAuthApplication500JSONResponse(unwrapError(resp, err)), nil
	}

	return portalv1.CreateOAuthApplication201JSONResponse{
		ClientId:     &createdClient.Id,
		ClientSecret: &createdClient.Secret,
		ClientName:   &createdClient.Name,
	}, nil
}

// DeleteApplication deletes a client by ID.
func (s *StrictServerHandler) DeleteApplication(
	ctx context.Context,
	request portalv1.DeleteApplicationRequestObject,
) (portalv1.DeleteApplicationResponseObject, error) {
	resp, err := s.restClient.R().
		Delete(s.adminRoot + "/clients/" + request.Id)

	if err != nil || resp.IsError() {
		switch portalErr := unwrapError(resp, err); portalErr.Code {
		case 404:
			return portalv1.DeleteApplication404JSONResponse(portalErr), nil
		default:
			return portalv1.DeleteApplication500JSONResponse(portalErr), nil
		}
	}

	return portalv1.DeleteApplication204Response{}, nil
}

// UpdateAppAPIProducts updates resources for a client in Keycloak.
func (s *StrictServerHandler) UpdateAppAPIProducts(
	ctx context.Context,
	request portalv1.UpdateAppAPIProductsRequestObject,
) (portalv1.UpdateAppAPIProductsResponseObject, error) {
	if request.Body == nil {
		return portalv1.UpdateAppAPIProducts400JSONResponse(newPortal400Error("request body is required")), nil
	}

	// TODO: implement updating API products
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

// CreateAPIProduct creates resources in Keycloak
func (s *StrictServerHandler) CreateAPIProduct(
	ctx context.Context,
	request portalv1.CreateAPIProductRequestObject,
) (portalv1.CreateAPIProductResponseObject, error) {
	if request.Body == nil {
		return portalv1.CreateAPIProduct400JSONResponse(newPortal400Error("request body is required")), nil
	}

	// TODO: implement creating API products
	err := errors.New("unimplemented")

	if err != nil {
		return portalv1.CreateAPIProduct500JSONResponse(unwrapError(nil, err)), nil
	}

	return portalv1.CreateAPIProduct201Response{}, nil
}

// DeleteAPIProduct deletes resources in Keycloak
func (s *StrictServerHandler) DeleteAPIProduct(
	ctx context.Context,
	request portalv1.DeleteAPIProductRequestObject,
) (portalv1.DeleteAPIProductResponseObject, error) {

	// TODO: implement deleting API products
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
