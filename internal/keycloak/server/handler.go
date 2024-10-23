package server

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	resty "github.com/go-resty/resty/v2"
	portalv1 "github.com/solo-io/gloo-portal-idp-connect/pkg/api/v1"
)

type StrictServerHandler struct {
	restClient          resty.Client
	issuer              string
	discoveredEndpoints DiscoveredEndpoints
	adminRoot           string
	mgmtClientId        string
	mgmtClientSecret    string
}

type KeycloakToken struct {
	AccessToken string `json:"access_token"`
}

type KeycloakClient struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Secret string `json:"secret"`
}

type KeycloakError struct {
	Error       string `json:"error"`
	Description string `json:"error_description"`
}

func NewStrictServerHandler(opts *Options, restyClient *resty.Client, discoveredEndpoints DiscoveredEndpoints) *StrictServerHandler {
	r := regexp.MustCompile("^(https?:.*?)/realms/(.[^/]*)/?$")
	adminRoot := r.ReplaceAllString(opts.Issuer, "$1/admin/realms/$2")

	var token *KeycloakToken
	var tokenRefreshed time.Time

	restyClient.OnBeforeRequest(func(c *resty.Client, r *resty.Request) error {
		// If we already have user info, assume this is a request to fetch a token
		if r.UserInfo != nil {
			return nil
		}

		// Reuse the last token if we got it less than a minute ago
		if token == nil || time.Since(tokenRefreshed).Seconds() > 60 {
			tokenResponse, err := c.R().
				SetBasicAuth(opts.MgmtClientId, opts.MgmtClientSecret).
				SetFormData(map[string]string{
					"grant_type": "urn:ietf:params:oauth:grant-type:uma-ticket",
					"audience":   opts.MgmtClientId,
				}).
				SetResult(&token).
				SetError(&KeycloakError{}).
				Post(discoveredEndpoints.Tokens)

			tokenRefreshed = time.Now()

			if err != nil {
				return err
			}

			if tokenResponse.IsError() {
				error := tokenResponse.Error().(*KeycloakError)
				return fmt.Errorf("could not obtain token for client %s: [%s] %s", opts.MgmtClientId, error.Error, error.Description)
			}
		}

		r.SetAuthToken(token.AccessToken)
		r.SetError(&KeycloakError{})

		return nil
	})

	return &StrictServerHandler{
		restClient:          *restyClient,
		issuer:              opts.Issuer,
		discoveredEndpoints: discoveredEndpoints,
		adminRoot:           adminRoot,
		mgmtClientId:        opts.MgmtClientId,
		mgmtClientSecret:    opts.MgmtClientSecret,
	}
}

// CreateOAuthApplication creates a client in Keycloak
func (s *StrictServerHandler) CreateOAuthApplication(
	_ context.Context,
	request portalv1.CreateOAuthApplicationRequestObject,
) (portalv1.CreateOAuthApplicationResponseObject, error) {
	if request.Body == nil || len(request.Body.Id) == 0 {
		return portalv1.CreateOAuthApplication400JSONResponse(newPortal400Error("unique id is required")), nil
	}

	var createdClient KeycloakClient

	resp, err := s.restClient.R().
		SetBody(map[string]interface{}{
			"clientId":               request.Body.Id,
			"name":                   request.Body.Id,
			"serviceAccountsEnabled": true,
		}).
		SetResult(&createdClient).
		Post(s.issuer + "/clients-registrations/default")

	if err != nil || resp.IsError() {
		return portalv1.CreateOAuthApplication500JSONResponse(unwrapError(resp, err)), nil
	}

	return portalv1.CreateOAuthApplication201JSONResponse{
		ClientId:     createdClient.Name,
		ClientName:   &createdClient.Name,
		ClientSecret: createdClient.Secret,
	}, nil
}

// DeleteOAuthApplication deletes a client in Keycloak by ID.
func (s *StrictServerHandler) DeleteOAuthApplication(
	_ context.Context,
	request portalv1.DeleteOAuthApplicationRequestObject,
) (portalv1.DeleteOAuthApplicationResponseObject, error) {
	if len(request.Id) == 0 {
		return portalv1.DeleteOAuthApplication404JSONResponse(newPortal400Error("client ID is required")), nil
	}

	// Get the Keycloak internal ID of the client
	var clients []KeycloakClient
	getId, err := s.restClient.R().
		SetQueryParams(map[string]string{
			"clientId": request.Id,
		}).
		SetResult(&clients).
		Get(s.adminRoot + "/clients")

	if err != nil || getId.IsError() {
		return portalv1.DeleteOAuthApplication500JSONResponse(unwrapError(getId, err)), nil
	}

	if len(clients) == 0 {
		return portalv1.DeleteOAuthApplication404JSONResponse(newPortal400Error("no client matches name [" + request.Id + "]")), nil
	}

	if len(clients) > 1 {
		// If we get this then we're not looking up the ID properly
		return portalv1.DeleteOAuthApplication500JSONResponse(newPortal500Error("more than one matching client found for [" + request.Id + "]")), nil
	}

	// Delete the client with the single ID we located
	resp, err := s.restClient.R().
		Delete(s.adminRoot + "/clients/" + clients[0].Id)

	if err != nil || resp.IsError() {
		switch portalErr := unwrapError(resp, err); portalErr.Code {
		case 404:
			return portalv1.DeleteOAuthApplication404JSONResponse(portalErr), nil
		default:
			return portalv1.DeleteOAuthApplication500JSONResponse(portalErr), nil
		}
	}

	return portalv1.DeleteOAuthApplication204Response{}, nil
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
