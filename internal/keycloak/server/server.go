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

const (
	wellKnownOpenIdConfigPath = "/.well-known/openid-configuration"
)

type Options struct {
	Port             string
	Issuer           string
	MgmtClientId     string
	MgmtClientSecret string
	ResourceServer   string
}

type OpenidConfiguration struct {
	TokenEndpoint string `json:"token_endpoint"`
}

func (o *Options) AddToFlags(flag *pflag.FlagSet) {
	flag.StringVar(&o.Port, "port", "8080", "Port for HTTP server")
	flag.StringVar(&o.Issuer, "issuer", "", "Keycloak issuer URL (e.g. https://keycloak.example.com/realms/my-org)")
	flag.StringVar(&o.MgmtClientId, "client-id", "", "ID of the Keycloak client that is authorised to manage clients")
	flag.StringVar(&o.MgmtClientSecret, "client-secret", "", "Secret of the Keycloak client that is authorised to manage clients")
	flag.StringVar(&o.ResourceServer, "resource-server", "", "Resource server to configure API Product scopes")
}

func (o *Options) Validate() error {
	if o.Issuer == "" {
		return eris.New("Issuer is required")
	}
	if o.ResourceServer == "" {
		return eris.New("Resource server is required")
	}
	return nil
}

func ListenAndServe(ctx context.Context, opts *Options) error {
	if err := opts.Validate(); err != nil {
		return err
	}

	client := resty.New()

	resp, err := client.R().
		SetResult(OpenidConfiguration{}).
		Get(opts.Issuer + wellKnownOpenIdConfigPath)
	if err != nil {
		return eris.Wrap(err, "Issuer configuration could not be discovered")
	}

	tokenEndpoint := resp.Result().(*OpenidConfiguration).TokenEndpoint
	if len(tokenEndpoint) == 0 {
		return eris.New("Token endpoint was not provided by the issuer")
	}

	swagger, err := portalv1.GetSwagger()
	if err != nil {
		return eris.Wrap(err, "could not load swagger spec")
	}

	// Clear out the servers array in the swagger spec, that skips validating
	// that server names match. We don't know how this thing will be run.
	swagger.Servers = nil

	// Create an instance of our handler which satisfies the generated interface
	keycloakHandler := NewStrictServerHandler(opts, client, tokenEndpoint)
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
