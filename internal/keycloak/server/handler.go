package server

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
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

type Permission struct {
	Id      string   `json:"id"`
	Clients []string `json:"clients"`
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
			"clientId": request.Body.Id,
			"name":     request.Body.Id,
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

// DeleteApplication deletes a client by ID.
func (s *StrictServerHandler) DeleteApplication(
	_ context.Context,
	request portalv1.DeleteApplicationRequestObject,
) (portalv1.DeleteApplicationResponseObject, error) {
	if len(request.Id) == 0 {
		return portalv1.DeleteApplication404JSONResponse(newPortal400Error("client ID is required")), nil
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
		return portalv1.DeleteApplication500JSONResponse(unwrapError(getId, err)), nil
	}

	if len(clients) == 0 {
		return portalv1.DeleteApplication404JSONResponse(newPortal400Error("no client matches name [" + request.Id + "]")), nil
	}

	if len(clients) > 1 {
		// If we get this then we're not looking up the ID properly
		return portalv1.DeleteApplication500JSONResponse(newPortal500Error("more than one matching client found for [" + request.Id + "]")), nil
	}

	// Delete the client with the single ID we located
	resp, err := s.restClient.R().
		Delete(s.adminRoot + "/clients/" + clients[0].Id)

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
	if len(request.Id) == 0 {
		return portalv1.UpdateAppAPIProducts400JSONResponse(newPortal400Error("client ID is required")), nil
	}
	if request.Body == nil {
		return portalv1.UpdateAppAPIProducts400JSONResponse(newPortal400Error("request body is required")), nil
	}

	// Make sure the client exists
	var clients []KeycloakClient
	getClient, err := s.restClient.R().
		SetQueryParams(map[string]string{
			"clientId": request.Id,
		}).
		SetResult(&clients).
		Get(s.adminRoot + "/clients")

	if err != nil || getClient.IsError() {
		return portalv1.UpdateAppAPIProducts500JSONResponse(unwrapError(getClient, err)), nil
	}

	if len(clients) == 0 {
		return portalv1.UpdateAppAPIProducts404JSONResponse(newPortal400Error("no client matches name [" + request.Id + "]")), nil
	}

	if len(clients) > 1 {
		// If we get this then we're not looking up the ID properly
		return portalv1.UpdateAppAPIProducts500JSONResponse(newPortal500Error("more than one matching client found for [" + request.Id + "]")), nil
	}

	// We need the internal IDs of the API resources before we can associate them with permissions
	var resourceIds = make(map[string]string)

	for _, api := range request.Body.ApiProducts {
		var matchingResourceIds []string

		getId, err := s.restClient.R().
			SetQueryParams(map[string]string{
				"name":      api,
				"exactName": "true",
			}).
			SetResult(&matchingResourceIds).
			Get(s.discoveredEndpoints.ResourceRegistration)

		if err != nil || getId.IsError() {
			return portalv1.UpdateAppAPIProducts500JSONResponse(unwrapError(getId, err)), nil
		}

		if len(matchingResourceIds) == 0 {
			return portalv1.UpdateAppAPIProducts400JSONResponse(newPortal400Error("no resource matches name [" + api + "]")), nil
		}

		if len(matchingResourceIds) > 1 {
			// Keycloak enforces unique names, so if we get this then we're not looking up the ID properly
			return portalv1.UpdateAppAPIProducts500JSONResponse(newPortal500Error("more than one matching resource found for [" + api + "]")), nil
		}

		resourceIds[api] = matchingResourceIds[0]
	}

	// Get all the existing permissions, so we can filter by those that are just for the given client
	var allPermissions []Permission
	getPermissions, err := s.restClient.R().
		SetResult(&allPermissions).
		Get(s.discoveredEndpoints.Policy)

	if err != nil || getPermissions.IsError() {
		return portalv1.UpdateAppAPIProducts500JSONResponse(unwrapError(getPermissions, err)), nil
	}

	// Delete all existing permissions for this client
	for _, permission := range allPermissions {
		if slices.Contains(permission.Clients, request.Id) {
			deletePermission, err := s.restClient.R().
				Delete(s.discoveredEndpoints.Policy + "/" + permission.Id)

			if err != nil || deletePermission.IsError() {
				return portalv1.UpdateAppAPIProducts500JSONResponse(unwrapError(deletePermission, err)), nil
			}
		}
	}

	// Create new permissions for all API products in the request
	for resourceName, resourceId := range resourceIds {
		newPermission, err := s.restClient.R().
			SetBody(map[string]interface{}{
				"name":        request.Id + "/" + resourceName,
				"description": resourceName + " access for client " + request.Id,
				"clients":     [1]string{request.Id},
			}).
			Post(s.discoveredEndpoints.Policy + "/" + resourceId)

		if err != nil || newPermission.IsError() {
			return portalv1.UpdateAppAPIProducts500JSONResponse(unwrapError(newPermission, err)), nil
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

	resp, err := s.restClient.R().
		SetBody(map[string]interface{}{
			"name":               request.Body.ApiProduct.Name,
			"displayName":        request.Body.ApiProduct.Description,
			"ownerManagedAccess": "true",
		}).
		Post(s.discoveredEndpoints.ResourceRegistration)

	if err != nil || resp.IsError() {
		switch portalErr := unwrapError(resp, err); portalErr.Code {
		case 409:
			return portalv1.CreateAPIProduct409JSONResponse(portalErr), nil
		default:
			return portalv1.CreateAPIProduct500JSONResponse(portalErr), nil
		}
	}

	return portalv1.CreateAPIProduct201Response{}, nil
}

// DeleteAPIProduct deletes resources in Keycloak
func (s *StrictServerHandler) DeleteAPIProduct(
	ctx context.Context,
	request portalv1.DeleteAPIProductRequestObject,
) (portalv1.DeleteAPIProductResponseObject, error) {
	if len(request.Name) == 0 {
		return portalv1.DeleteAPIProduct404JSONResponse(newPortal400Error("name is required")), nil
	}

	// We need the internal ID of the resource before we can delete it
	var resourceIds []string
	getId, err := s.restClient.R().
		SetQueryParams(map[string]string{
			"name":      request.Name,
			"exactName": "true",
		}).
		SetResult(&resourceIds).
		Get(s.discoveredEndpoints.ResourceRegistration)

	if err != nil || getId.IsError() {
		return portalv1.DeleteAPIProduct500JSONResponse(unwrapError(getId, err)), nil
	}

	if len(resourceIds) == 0 {
		return portalv1.DeleteAPIProduct404JSONResponse(newPortal400Error("no resource matches this name")), nil
	}

	if len(resourceIds) > 1 {
		// Keycloak enforces unique names, so if we get this then we're not looking up the ID properly
		return portalv1.DeleteAPIProduct500JSONResponse(newPortal500Error("more than one matching resource found")), nil
	}

	resp, err := s.restClient.R().
		Delete(s.discoveredEndpoints.ResourceRegistration + "/" + resourceIds[0])

	if err != nil || resp.IsError() {
		return portalv1.DeleteAPIProduct500JSONResponse(unwrapError(resp, err)), nil
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
