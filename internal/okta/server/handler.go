package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/okta/okta-sdk-golang/v5/okta"
	portalv1 "github.com/solo-io/gloo-portal-idp-connect/pkg/api/v1"
)

//go:generate mockgen -destination=mock/okta_client.go . OktaClient,ApplicationAPI,ApiCreateApplicationRequest,ApiListApplicationsRequest,ApiDeactivateApplicationRequest,ApiDeleteApplicationRequest

type OktaClient interface {
	GetApplicationAPI() ApplicationAPI
}

type ApplicationAPI interface {
	CreateApplication(ctx context.Context) ApiCreateApplicationRequest
	ListApplications(ctx context.Context) ApiListApplicationsRequest
	DeactivateApplication(ctx context.Context, appId string) ApiDeactivateApplicationRequest
	DeleteApplication(ctx context.Context, appId string) ApiDeleteApplicationRequest
}

type ApiCreateApplicationRequest interface {
	Application(application okta.ListApplications200ResponseInner) ApiCreateApplicationRequest
	Execute() (*okta.ListApplications200ResponseInner, *okta.APIResponse, error)
}

type ApiListApplicationsRequest interface {
	Execute() ([]okta.ListApplications200ResponseInner, *okta.APIResponse, error)
}

type ApiDeactivateApplicationRequest interface {
	Execute() (*okta.APIResponse, error)
}

type ApiDeleteApplicationRequest interface {
	Execute() (*okta.APIResponse, error)
}

type StrictServerHandler struct {
	oktaClient OktaClient
}

func NewStrictServerHandler(oktaClient OktaClient) *StrictServerHandler {
	return &StrictServerHandler{
		oktaClient: oktaClient,
	}
}

// CreateOAuthApplication creates a client in Okta
func (s *StrictServerHandler) CreateOAuthApplication(
	ctx context.Context,
	request portalv1.CreateOAuthApplicationRequestObject,
) (portalv1.CreateOAuthApplicationResponseObject, error) {
	if request.Body == nil || len(request.Body.Id) == 0 {
		return portalv1.CreateOAuthApplication400JSONResponse(newPortal400Error("unique id is required")), nil
	}

	// Create OAuth 2.0 Service Application in Okta using SDK
	// Set credentials
	credentials := okta.NewOAuthApplicationCredentials()
	oauthClient := okta.NewApplicationCredentialsOAuthClient()
	oauthClient.SetTokenEndpointAuthMethod("client_secret_basic")
	credentials.SetOauthClient(*oauthClient)

	// Set settings
	settings := okta.NewOpenIdConnectApplicationSettings()
	oauthClientSettings := okta.NewOpenIdConnectApplicationSettingsClient()
	oauthClientSettings.SetGrantTypes([]string{"client_credentials"})
	oauthClientSettings.SetResponseTypes([]string{"token"})
	oauthClientSettings.SetApplicationType("service")
	oauthClientSettings.SetConsentMethod("TRUSTED")
	oauthClientSettings.SetIssuerMode("ORG_URL")
	settings.SetOauthClient(*oauthClientSettings)

	app := okta.NewOpenIdConnectApplication(*credentials, "oidc_client", *settings, "OPENID_CONNECT", request.Body.Id)

	// Create the application - wrap in union type
	appUnion := okta.OpenIdConnectApplicationAsListApplications200ResponseInner(app)

	createdAppUnion, resp, err := s.oktaClient.GetApplicationAPI().
		CreateApplication(ctx).
		Application(appUnion).
		Execute()

	if err != nil {
		return portalv1.CreateOAuthApplication500JSONResponse(unwrapSDKError(resp.Response, err)), nil
	}

	// Extract the OpenIdConnectApplication from the union type
	if createdAppUnion == nil || createdAppUnion.OpenIdConnectApplication == nil {
		return portalv1.CreateOAuthApplication500JSONResponse(newPortal500Error("unexpected application type returned")), nil
	}

	oidcApp := createdAppUnion.OpenIdConnectApplication

	clientId := ""
	clientSecret := ""
	clientName := request.Body.Id

	creds := oidcApp.GetCredentials()
	if oauthCreds, ok := creds.GetOauthClientOk(); ok && oauthCreds != nil {
		if id, ok := oauthCreds.GetClientIdOk(); ok && id != nil {
			clientId = *id
		}
		if secret, ok := oauthCreds.GetClientSecretOk(); ok && secret != nil {
			clientSecret = *secret
		}
	}

	return portalv1.CreateOAuthApplication201JSONResponse{
		ClientId:     clientId,
		ClientSecret: clientSecret,
		ClientName:   &clientName,
	}, nil
}

// DeleteOAuthApplication deletes a client in Okta by ID.
func (s *StrictServerHandler) DeleteOAuthApplication(
	ctx context.Context,
	request portalv1.DeleteOAuthApplicationRequestObject,
) (portalv1.DeleteOAuthApplicationResponseObject, error) {
	if len(request.Id) == 0 {
		return portalv1.DeleteOAuthApplication500JSONResponse(newPortal500Error("client ID is required")), nil
	}

	// First, find the application by searching for apps with matching label
	apps, resp, err := s.oktaClient.GetApplicationAPI().
		ListApplications(ctx).
		Execute()

	if err != nil {
		return portalv1.DeleteOAuthApplication500JSONResponse(unwrapSDKError(resp.Response, err)), nil
	}

	// Find the app with matching label, name, ID, or client ID
	var targetAppId string

	for _, appUnion := range apps {
		var appId string
		var appLabel string
		var appName string
		var clientId string

		// Try OpenIdConnectApplication
		if appUnion.OpenIdConnectApplication != nil {
			app := appUnion.OpenIdConnectApplication
			if id, ok := app.GetIdOk(); ok && id != nil {
				appId = *id
			}
			if label, ok := app.GetLabelOk(); ok && label != nil {
				appLabel = *label
			}
			if name, ok := app.GetNameOk(); ok && name != nil {
				appName = *name
			}
			creds := app.GetCredentials()
			if oauthCreds, ok := creds.GetOauthClientOk(); ok && oauthCreds != nil {
				if id, ok := oauthCreds.GetClientIdOk(); ok && id != nil {
					clientId = *id
				}
			}
		}

		// Try matching by label, name, internal ID, or OAuth client ID
		if appLabel == request.Id ||
			appName == request.Id ||
			appId == request.Id ||
			clientId == request.Id {
			targetAppId = appId
			break
		}
	}

	if targetAppId == "" {
		reason := fmt.Sprintf("Application '%s' not found. Found %d applications",
			request.Id, len(apps))
		return portalv1.DeleteOAuthApplication404JSONResponse(portalv1.Error{
			Code:    404,
			Message: "Not Found",
			Reason:  reason,
		}), nil
	}

	// Step 1: Deactivate the application first (Okta requires this before deletion)
	_, err = s.oktaClient.GetApplicationAPI().
		DeactivateApplication(ctx, targetAppId).
		Execute()

	if err != nil {
		return portalv1.DeleteOAuthApplication500JSONResponse(portalv1.Error{
			Code:    500,
			Message: "Failed to deactivate application",
			Reason:  fmt.Sprintf("Error deactivating app before delete: %v", err),
		}), nil
	}

	// Step 2: Now delete the deactivated application
	deleteResp, err := s.oktaClient.GetApplicationAPI().
		DeleteApplication(ctx, targetAppId).
		Execute()

	if err != nil {
		return portalv1.DeleteOAuthApplication500JSONResponse(unwrapSDKError(deleteResp.Response, err)), nil
	}

	return portalv1.DeleteOAuthApplication204Response{}, nil
}

func unwrapSDKError(resp *http.Response, err error) portalv1.Error {
	if err != nil {
		if resp != nil {
			return portalv1.Error{
				Code:    resp.StatusCode,
				Message: resp.Status,
				Reason:  err.Error(),
			}
		}
		return newPortal500Error(err.Error())
	}

	return newPortal500Error("unknown error occurred")
}
