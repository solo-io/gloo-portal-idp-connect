// This is an example of implementing the Pet Store from the OpenAPI documentation
// found at:
// https://github.com/OAI/OpenAPI-Specification/blob/master/examples/v3.0/petstore.yaml

package server

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/config"
	cognito "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	middleware "github.com/oapi-codegen/echo-middleware"
	"github.com/rotisserie/eris"

	portalv1 "github.com/solo-io/gloo-portal-idp-connect/pkg/api/v1"
)

type Options struct {
	port            string
	cognitoUserPool string
}

func (o *Options) AddToFlags(flag *flag.FlagSet) {
	flag.StringVar(&o.port, "port", "8080", "Port for HTTP server")
	flag.StringVar(&o.cognitoUserPool, "cognito-user-pool", "", "Cognito user pool")
}

func (o *Options) Validate() error {
	if o.cognitoUserPool == "" {
		return eris.New("cognito user pool is required")
	}
	return nil
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

	// Unless performance is a concern, always use LoadDefaultConfig because it will search the environment
	// for valid configuration; this allows users maximum flexibility and provides break-glass provider
	// configuration options
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return eris.Wrap(err, "failed to locate aws configuration using full provider chain")
	}

	cognitoClient := cognito.NewFromConfig(cfg)
	// Create an instance of our handler which satisfies the generated interface
	congitoHandler := newStrictServerHandler(opts, cognitoClient)
	portalHandler := portalv1.NewStrictHandler(congitoHandler, nil)

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

	// We now register our petStore above as the handler for the interface
	portalv1.RegisterHandlers(e, portalHandler)

	s := &http.Server{
		Handler: e,
		Addr:    net.JoinHostPort("0.0.0.0", opts.port),
	}

	log.Printf("Starting server on port %v\n", opts.port)
	return s.ListenAndServe()
}
