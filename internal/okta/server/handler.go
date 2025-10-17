package server

import (
	"context"
	"errors"
	"fmt"
	"strings"

	resty "github.com/go-resty/resty/v2"
	portalv1 "github.com/solo-io/gloo-portal-idp-connect/pkg/api/v1"
)

type StrictServerHandler struct {
	restClient *resty.Client
	oktaDomain string
	apiToken   string
}

type OktaApplication struct {
	Id          string           `json:"id,omitempty"`
	Name        string           `json:"name"`
	Label       string           `json:"label"`
	Status      string           `json:"status,omitempty"`
	SignOnMode  string           `json:"signOnMode"`
	Credentials *OktaCredentials `json:"credentials,omitempty"`
	Settings    *OktaSettings    `json:"settings,omitempty"`
}

type OktaCredentials struct {
	OAuthClient *OktaOAuthClient `json:"oauthClient,omitempty"`
}

type OktaOAuthClient struct {
	ClientId                string `json:"client_id,omitempty"`
	ClientSecret            string `json:"client_secret,omitempty"`
	TokenEndpointAuthMethod string `json:"token_endpoint_auth_method,omitempty"`
	ClientUri               string `json:"client_uri,omitempty"`
}

type OktaSettings struct {
	OAuthClient *OktaOAuthClientSettings `json:"oauthClient,omitempty"`
}

type OktaOAuthClientSettings struct {
	ClientUri       string   `json:"client_uri,omitempty"`
	LogoUri         string   `json:"logo_uri,omitempty"`
	RedirectUris    []string `json:"redirect_uris,omitempty"`
	ResponseTypes   []string `json:"response_types,omitempty"`
	GrantTypes      []string `json:"grant_types,omitempty"`
	ApplicationType string   `json:"application_type,omitempty"`
	ConsentMethod   string   `json:"consent_method,omitempty"`
	IssuerMode      string   `json:"issuer_mode,omitempty"`
}

type OktaError struct {
	ErrorCode    string `json:"errorCode"`
	ErrorSummary string `json:"errorSummary"`
	ErrorLink    string `json:"errorLink,omitempty"`
	ErrorId      string `json:"errorId,omitempty"`
}

func NewStrictServerHandler(opts *Options, restyClient *resty.Client) *StrictServerHandler {
	// Set up authentication for all requests
	restyClient.OnBeforeRequest(func(c *resty.Client, r *resty.Request) error {
		r.SetHeader("Authorization", "SSWS "+opts.APIToken)
		r.SetHeader("Accept", "application/json")
		r.SetHeader("Content-Type", "application/json")
		r.SetError(&OktaError{})
		return nil
	})

	return &StrictServerHandler{
		restClient: restyClient,
		oktaDomain: opts.OktaDomain,
		apiToken:   opts.APIToken,
	}
}

// CreateOAuthApplication creates a client in Okta
func (s *StrictServerHandler) CreateOAuthApplication(
	_ context.Context,
	request portalv1.CreateOAuthApplicationRequestObject,
) (portalv1.CreateOAuthApplicationResponseObject, error) {
	if request.Body == nil || len(request.Body.Id) == 0 {
		return portalv1.CreateOAuthApplication400JSONResponse(newPortal400Error("unique id is required")), nil
	}

	// Create OAuth 2.0 Service Application in Okta
	app := OktaApplication{
		Name:       "oidc_client", // Use correct Okta application name
		Label:      request.Body.Id,
		SignOnMode: "OPENID_CONNECT",
		Credentials: &OktaCredentials{
			OAuthClient: &OktaOAuthClient{
				TokenEndpointAuthMethod: "client_secret_basic",
			},
		},
		Settings: &OktaSettings{
			OAuthClient: &OktaOAuthClientSettings{
				GrantTypes:      []string{"client_credentials"},
				ApplicationType: "service",
				ConsentMethod:   "TRUSTED",
				IssuerMode:      "ORG_URL",
			},
		},
	}

	var createdApp OktaApplication

	// Construct the full URL - ensure no double slashes
	apiURL := s.oktaDomain
	if !strings.HasSuffix(apiURL, "/") {
		apiURL += "/"
	}
	apiURL += "api/v1/apps"

	resp, err := s.restClient.R().
		SetBody(app).
		SetResult(&createdApp).
		Post(apiURL)

	if err != nil {
		return portalv1.CreateOAuthApplication500JSONResponse(unwrapError(resp, err)), nil
	}

	if resp.IsError() {
		// Log the response for debugging
		oktaErr := resp.Error().(*OktaError)
		return portalv1.CreateOAuthApplication500JSONResponse(portalv1.Error{
			Code:    resp.StatusCode(),
			Message: fmt.Sprintf("Okta API Error: %s", oktaErr.ErrorSummary),
			Reason:  fmt.Sprintf("[%s] %s - Status: %d, Body: %s", oktaErr.ErrorCode, oktaErr.ErrorSummary, resp.StatusCode(), string(resp.Body())),
		}), nil
	}

	clientId := ""
	clientSecret := ""
	if createdApp.Credentials != nil && createdApp.Credentials.OAuthClient != nil {
		clientId = createdApp.Credentials.OAuthClient.ClientId
		clientSecret = createdApp.Credentials.OAuthClient.ClientSecret
	}

	return portalv1.CreateOAuthApplication201JSONResponse{
		ClientId:     clientId,
		ClientSecret: clientSecret,
		ClientName:   &createdApp.Label,
	}, nil
}

// DeleteOAuthApplication deletes a client in Okta by ID.
func (s *StrictServerHandler) DeleteOAuthApplication(
	_ context.Context,
	request portalv1.DeleteOAuthApplicationRequestObject,
) (portalv1.DeleteOAuthApplicationResponseObject, error) {
	if len(request.Id) == 0 {
		return portalv1.DeleteOAuthApplication500JSONResponse(newPortal500Error("client ID is required")), nil
	}

	// First, find the application by searching for apps with matching label
	var apps []OktaApplication

	// Construct search URL
	searchURL := s.oktaDomain
	if !strings.HasSuffix(searchURL, "/") {
		searchURL += "/"
	}
	searchURL += "api/v1/apps"

	// Try to find the application - Okta search can be tricky, so try multiple approaches
	searchResp, err := s.restClient.R().
		SetResult(&apps).
		Get(searchURL)

	// For debugging, let's also try with query param in case that works better
	if err != nil {
		apps = []OktaApplication{} // Reset
		searchResp, err = s.restClient.R().
			SetQueryParam("q", request.Id).
			SetResult(&apps).
			Get(searchURL)
	}

	if err != nil {
		return portalv1.DeleteOAuthApplication500JSONResponse(unwrapError(searchResp, err)), nil
	}

	if searchResp.IsError() {
		return portalv1.DeleteOAuthApplication500JSONResponse(unwrapError(searchResp, nil)), nil
	}

	// Find the app with matching label, name, ID, or client ID
	var targetApp *OktaApplication
	var foundApps []string

	for _, app := range apps {
		clientId := ""
		if app.Credentials != nil && app.Credentials.OAuthClient != nil {
			clientId = app.Credentials.OAuthClient.ClientId
		}

		foundApps = append(foundApps, fmt.Sprintf("ID: %s, Name: %s, Label: %s, ClientId: %s",
			app.Id, app.Name, app.Label, clientId))

		// Try matching by label, name, internal ID, or OAuth client ID
		if app.Label == request.Id ||
			app.Name == request.Id ||
			app.Id == request.Id ||
			clientId == request.Id {
			targetApp = &app
			break
		}
	}

	if targetApp == nil {
		// Return detailed error with found applications for debugging
		reason := fmt.Sprintf("Application '%s' not found. Found %d applications: [%s]",
			request.Id, len(apps), strings.Join(foundApps, "; "))
		return portalv1.DeleteOAuthApplication404JSONResponse(portalv1.Error{
			Code:    404,
			Message: "Not Found",
			Reason:  reason,
		}), nil
	}

	// Step 1: Deactivate the application first (Okta requires this before deletion)
	deactivateURL := s.oktaDomain
	if !strings.HasSuffix(deactivateURL, "/") {
		deactivateURL += "/"
	}
	deactivateURL += "api/v1/apps/" + targetApp.Id + "/lifecycle/deactivate"

	deactivateResp, err := s.restClient.R().
		Post(deactivateURL)

	if err != nil {
		return portalv1.DeleteOAuthApplication500JSONResponse(portalv1.Error{
			Code:    500,
			Message: "Failed to deactivate application",
			Reason:  fmt.Sprintf("Error deactivating app before delete: %v", err),
		}), nil
	}

	if deactivateResp.IsError() {
		return portalv1.DeleteOAuthApplication500JSONResponse(portalv1.Error{
			Code:    deactivateResp.StatusCode(),
			Message: "Failed to deactivate application",
			Reason:  fmt.Sprintf("Okta deactivate failed - Status: %d, Body: %s", deactivateResp.StatusCode(), string(deactivateResp.Body())),
		}), nil
	}

	// Step 2: Now delete the deactivated application
	deleteURL := s.oktaDomain
	if !strings.HasSuffix(deleteURL, "/") {
		deleteURL += "/"
	}
	deleteURL += "api/v1/apps/" + targetApp.Id

	resp, err := s.restClient.R().
		Delete(deleteURL)

	if err != nil {
		return portalv1.DeleteOAuthApplication500JSONResponse(unwrapError(resp, err)), nil
	}

	if resp.IsError() {
		switch resp.StatusCode() {
		case 404:
			return portalv1.DeleteOAuthApplication404JSONResponse(unwrapError(resp, nil)), nil
		default:
			return portalv1.DeleteOAuthApplication500JSONResponse(unwrapError(resp, nil)), nil
		}
	}

	return portalv1.DeleteOAuthApplication204Response{}, nil
}

func unwrapError(resp *resty.Response, err error) portalv1.Error {
	if err != nil {
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

	if resp != nil && resp.Error() != nil {
		oktaErr := resp.Error().(*OktaError)
		return portalv1.Error{
			Code:    resp.StatusCode(),
			Message: oktaErr.ErrorSummary,
			Reason:  fmt.Sprintf("[%s] %s", oktaErr.ErrorCode, oktaErr.ErrorSummary),
		}
	}

	return newPortal500Error("unknown error occurred")
}
