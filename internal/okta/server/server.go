package server

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	middleware "github.com/oapi-codegen/echo-middleware"
	"github.com/okta/okta-sdk-golang/v5/okta"
	"github.com/rotisserie/eris"
	"github.com/spf13/pflag"

	portalv1 "github.com/solo-io/gloo-portal-idp-connect/pkg/api/v1"
)

type Options struct {
	Port       string
	OktaDomain string
	APIToken   string
}

func (o *Options) AddToFlags(flag *pflag.FlagSet) {
	flag.StringVar(&o.Port, "port", "8080", "Port for HTTP server")
	flag.StringVar(&o.OktaDomain, "okta-domain", "", "Okta domain (e.g. https://dev-123456.okta.com)")
	flag.StringVar(&o.APIToken, "api-token", "", "Okta API token for application management")
}

func (o *Options) Validate() error {
	if o.OktaDomain == "" {
		return eris.New("Okta domain is required")
	}

	// Try to get API token from environment variable if not provided via flag
	if o.APIToken == "" {
		if envToken := os.Getenv("OKTA_API_TOKEN"); envToken != "" {
			o.APIToken = envToken
		} else {
			return eris.New("Okta API token is required (via --api-token flag or OKTA_API_TOKEN environment variable)")
		}
	}

	return nil
}

// oktaClientWrapper wraps the Okta SDK client to implement our OktaClient interface
type oktaClientWrapper struct {
	apiClient *okta.APIClient
}

func (w *oktaClientWrapper) GetApplicationAPI() ApplicationAPI {
	return &applicationAPIWrapper{api: w.apiClient.ApplicationAPI}
}

// applicationAPIWrapper wraps the SDK ApplicationAPI to match our interface
type applicationAPIWrapper struct {
	api okta.ApplicationAPI
}

func (w *applicationAPIWrapper) CreateApplication(ctx context.Context) ApiCreateApplicationRequest {
	return &createApplicationRequestWrapper{req: w.api.CreateApplication(ctx)}
}

func (w *applicationAPIWrapper) ListApplications(ctx context.Context) ApiListApplicationsRequest {
	return &listApplicationsRequestWrapper{req: w.api.ListApplications(ctx)}
}

func (w *applicationAPIWrapper) DeactivateApplication(ctx context.Context, appId string) ApiDeactivateApplicationRequest {
	return &deactivateApplicationRequestWrapper{req: w.api.DeactivateApplication(ctx, appId)}
}

func (w *applicationAPIWrapper) DeleteApplication(ctx context.Context, appId string) ApiDeleteApplicationRequest {
	return &deleteApplicationRequestWrapper{req: w.api.DeleteApplication(ctx, appId)}
}

// Request wrappers
type createApplicationRequestWrapper struct {
	req okta.ApiCreateApplicationRequest
}

func (w *createApplicationRequestWrapper) Application(application okta.ListApplications200ResponseInner) ApiCreateApplicationRequest {
	w.req = w.req.Application(application)
	return w
}

func (w *createApplicationRequestWrapper) Execute() (*okta.ListApplications200ResponseInner, *okta.APIResponse, error) {
	return w.req.Execute()
}

type listApplicationsRequestWrapper struct {
	req okta.ApiListApplicationsRequest
}

func (w *listApplicationsRequestWrapper) Execute() ([]okta.ListApplications200ResponseInner, *okta.APIResponse, error) {
	return w.req.Execute()
}

type deactivateApplicationRequestWrapper struct {
	req okta.ApiDeactivateApplicationRequest
}

func (w *deactivateApplicationRequestWrapper) Execute() (*okta.APIResponse, error) {
	return w.req.Execute()
}

type deleteApplicationRequestWrapper struct {
	req okta.ApiDeleteApplicationRequest
}

func (w *deleteApplicationRequestWrapper) Execute() (*okta.APIResponse, error) {
	return w.req.Execute()
}

func ListenAndServe(ctx context.Context, opts *Options) error {
	if err := opts.Validate(); err != nil {
		return err
	}

	swagger, err := portalv1.GetSwagger()
	if err != nil {
		return eris.Wrap(err, "could not load swagger spec")
	}

	// Clear out the servers array in the swagger spec, that skips validating
	// that server names match. We don't know how this thing will be run.
	swagger.Servers = nil

	// Initialize Okta SDK client
	config, err := okta.NewConfiguration(
		okta.WithOrgUrl(opts.OktaDomain),
		okta.WithToken(opts.APIToken),
	)
	if err != nil {
		return eris.Wrap(err, "failed to create Okta configuration")
	}

	oktaAPIClient := okta.NewAPIClient(config)
	oktaClient := &oktaClientWrapper{apiClient: oktaAPIClient}

	// Create an instance of our handler which satisfies the generated interface
	oktaHandler := NewStrictServerHandler(oktaClient)
	portalHandler := portalv1.NewStrictHandler(oktaHandler, nil)

	e := echo.New()

	// Use our validation middleware to check all requests against the
	// OpenAPI schema.
	e.Use(echomiddleware.Logger())

	// Use our validation middleware to check all requests against the
	// OpenAPI schema.
	e.Use(middleware.OapiRequestValidatorWithOptions(swagger, &middleware.Options{
		Options: openapi3filter.Options{
			AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
		}},
	))

	// We now register our portal handler above as the handler for the interface
	portalv1.RegisterHandlers(e, portalHandler)

	s := &http.Server{
		Handler: e,
		Addr:    net.JoinHostPort("0.0.0.0", opts.Port),
	}

	log.Printf("Starting server on port %v\n", opts.Port)
	return s.ListenAndServe()
}
