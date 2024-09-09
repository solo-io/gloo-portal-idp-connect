package server

import (
	"context"
	"log"
	"net"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3filter"
	resty "github.com/go-resty/resty/v2"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	middleware "github.com/oapi-codegen/echo-middleware"
	"github.com/rotisserie/eris"
	"github.com/spf13/pflag"

	portalv1 "github.com/solo-io/gloo-portal-idp-connect/pkg/api/v1"
)

const wellKnownUmaConfigPath = "/.well-known/uma2-configuration"

type Options struct {
	Port             string
	Issuer           string
	MgmtClientId     string
	MgmtClientSecret string
}

type DiscoveredEndpoints struct {
	Policy               string
	ResourceRegistration string
	Tokens               string
}

func (o *Options) AddToFlags(flag *pflag.FlagSet) {
	flag.StringVar(&o.Port, "port", "8080", "Port for HTTP server")
	flag.StringVar(&o.Issuer, "issuer", "", "Keycloak issuer URL (e.g. https://keycloak.example.com/realms/my-org)")
	flag.StringVar(&o.MgmtClientId, "client-id", "", "ID of the Keycloak client that is authorised to manage app clients")
	flag.StringVar(&o.MgmtClientSecret, "client-secret", "", "Secret of the Keycloak client that is authorised to manage app clients")
}

func (o *Options) Validate() error {
	if o.Issuer == "" {
		return eris.New("Issuer is required")
	}
	return nil
}

func ListenAndServe(ctx context.Context, opts *Options) error {
	type UmaConfiguration struct {
		PolicyEndpoint               string `json:"policy_endpoint"`
		ResourceRegistrationEndpoint string `json:"resource_registration_endpoint"`
		TokenEndpoint                string `json:"token_endpoint"`
	}

	if err := opts.Validate(); err != nil {
		return err
	}

	client := resty.New()

	umaConfiguration, err := client.R().
		SetResult(UmaConfiguration{}).
		Get(opts.Issuer + wellKnownUmaConfigPath)
	if err != nil {
		return eris.Wrap(err, "UMA configuration could not be discovered")
	}

	policyEndpoint := umaConfiguration.Result().(*UmaConfiguration).PolicyEndpoint
	if len(policyEndpoint) == 0 {
		return eris.New("Policy endpoint was not provided by the issuer")
	}

	resourceRegistrationEndpoint := umaConfiguration.Result().(*UmaConfiguration).ResourceRegistrationEndpoint
	if len(resourceRegistrationEndpoint) == 0 {
		return eris.New("Resource registration endpoint was not provided by the issuer")
	}

	tokenEndpoint := umaConfiguration.Result().(*UmaConfiguration).TokenEndpoint
	if len(tokenEndpoint) == 0 {
		return eris.New("Token endpoint was not provided by the issuer")
	}

	discoveredEndpoints := DiscoveredEndpoints{
		Policy:               policyEndpoint,
		ResourceRegistration: resourceRegistrationEndpoint,
		Tokens:               tokenEndpoint,
	}

	swagger, err := portalv1.GetSwagger()
	if err != nil {
		return eris.Wrap(err, "could not load swagger spec")
	}

	// Clear out the servers array in the swagger spec, that skips validating
	// that server names match. We don't know how this thing will be run.
	swagger.Servers = nil

	// Create an instance of our handler which satisfies the generated interface
	keycloakHandler := NewStrictServerHandler(opts, client, discoveredEndpoints)
	portalHandler := portalv1.NewStrictHandler(keycloakHandler, nil)

	e := echo.New()
	// Log all requests
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

	// And we serve HTTP until the world ends.
	s := &http.Server{
		Handler: e,
		Addr:    net.JoinHostPort("0.0.0.0", opts.Port),
	}

	log.Printf("Starting server on port %v\n", opts.Port)
	return s.ListenAndServe()
}
