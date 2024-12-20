// Package v1 provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen/v2 version v2.1.0 DO NOT EDIT.
package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/oapi-codegen/runtime"
	strictecho "github.com/oapi-codegen/runtime/strictmiddleware/echo"
)

// ServerInterface represents all server handlers.
type ServerInterface interface {
	// Creates an application of type oauth2.
	// (POST /applications)
	CreateOAuthApplication(ctx echo.Context, params CreateOAuthApplicationParams) error
	// Deletes an application in the OpenID Connect Provider.
	// (DELETE /applications/{id})
	DeleteOAuthApplication(ctx echo.Context, id string, params DeleteOAuthApplicationParams) error
}

// ServerInterfaceWrapper converts echo contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler ServerInterface
}

// CreateOAuthApplication converts echo context to params.
func (w *ServerInterfaceWrapper) CreateOAuthApplication(ctx echo.Context) error {
	var err error

	// Parameter object where we will unmarshal all parameters from the context
	var params CreateOAuthApplicationParams

	headers := ctx.Request().Header
	// ------------- Optional header parameter "token" -------------
	if valueList, found := headers[http.CanonicalHeaderKey("token")]; found {
		var Token string
		n := len(valueList)
		if n != 1 {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Expected one value for token, got %d", n))
		}

		err = runtime.BindStyledParameterWithOptions("simple", "token", valueList[0], &Token, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationHeader, Explode: false, Required: false})
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter token: %s", err))
		}

		params.Token = &Token
	}

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.CreateOAuthApplication(ctx, params)
	return err
}

// DeleteOAuthApplication converts echo context to params.
func (w *ServerInterfaceWrapper) DeleteOAuthApplication(ctx echo.Context) error {
	var err error
	// ------------- Path parameter "id" -------------
	var id string

	err = runtime.BindStyledParameterWithOptions("simple", "id", ctx.Param("id"), &id, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter id: %s", err))
	}

	// Parameter object where we will unmarshal all parameters from the context
	var params DeleteOAuthApplicationParams

	headers := ctx.Request().Header
	// ------------- Optional header parameter "token" -------------
	if valueList, found := headers[http.CanonicalHeaderKey("token")]; found {
		var Token string
		n := len(valueList)
		if n != 1 {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Expected one value for token, got %d", n))
		}

		err = runtime.BindStyledParameterWithOptions("simple", "token", valueList[0], &Token, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationHeader, Explode: false, Required: false})
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter token: %s", err))
		}

		params.Token = &Token
	}

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.DeleteOAuthApplication(ctx, id, params)
	return err
}

// This is a simple interface which specifies echo.Route addition functions which
// are present on both echo.Echo and echo.Group, since we want to allow using
// either of them for path registration
type EchoRouter interface {
	CONNECT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	HEAD(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	OPTIONS(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	TRACE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
}

// RegisterHandlers adds each server route to the EchoRouter.
func RegisterHandlers(router EchoRouter, si ServerInterface) {
	RegisterHandlersWithBaseURL(router, si, "")
}

// Registers handlers, and prepends BaseURL to the paths, so that the paths
// can be served under a prefix.
func RegisterHandlersWithBaseURL(router EchoRouter, si ServerInterface, baseURL string) {

	wrapper := ServerInterfaceWrapper{
		Handler: si,
	}

	router.POST(baseURL+"/applications", wrapper.CreateOAuthApplication)
	router.DELETE(baseURL+"/applications/:id", wrapper.DeleteOAuthApplication)

}

type CreateOAuthApplicationRequestObject struct {
	Params CreateOAuthApplicationParams
	Body   *CreateOAuthApplicationJSONRequestBody
}

type CreateOAuthApplicationResponseObject interface {
	VisitCreateOAuthApplicationResponse(w http.ResponseWriter) error
}

type CreateOAuthApplication201JSONResponse OAuthApplication

func (response CreateOAuthApplication201JSONResponse) VisitCreateOAuthApplicationResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)

	return json.NewEncoder(w).Encode(response)
}

type CreateOAuthApplication400JSONResponse Error

func (response CreateOAuthApplication400JSONResponse) VisitCreateOAuthApplicationResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(400)

	return json.NewEncoder(w).Encode(response)
}

type CreateOAuthApplication500JSONResponse Error

func (response CreateOAuthApplication500JSONResponse) VisitCreateOAuthApplicationResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(500)

	return json.NewEncoder(w).Encode(response)
}

type DeleteOAuthApplicationRequestObject struct {
	Id     string `json:"id"`
	Params DeleteOAuthApplicationParams
}

type DeleteOAuthApplicationResponseObject interface {
	VisitDeleteOAuthApplicationResponse(w http.ResponseWriter) error
}

type DeleteOAuthApplication204Response struct {
}

func (response DeleteOAuthApplication204Response) VisitDeleteOAuthApplicationResponse(w http.ResponseWriter) error {
	w.WriteHeader(204)
	return nil
}

type DeleteOAuthApplication404JSONResponse Error

func (response DeleteOAuthApplication404JSONResponse) VisitDeleteOAuthApplicationResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(404)

	return json.NewEncoder(w).Encode(response)
}

type DeleteOAuthApplication500JSONResponse Error

func (response DeleteOAuthApplication500JSONResponse) VisitDeleteOAuthApplicationResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(500)

	return json.NewEncoder(w).Encode(response)
}

// StrictServerInterface represents all server handlers.
type StrictServerInterface interface {
	// Creates an application of type oauth2.
	// (POST /applications)
	CreateOAuthApplication(ctx context.Context, request CreateOAuthApplicationRequestObject) (CreateOAuthApplicationResponseObject, error)
	// Deletes an application in the OpenID Connect Provider.
	// (DELETE /applications/{id})
	DeleteOAuthApplication(ctx context.Context, request DeleteOAuthApplicationRequestObject) (DeleteOAuthApplicationResponseObject, error)
}

type StrictHandlerFunc = strictecho.StrictEchoHandlerFunc
type StrictMiddlewareFunc = strictecho.StrictEchoMiddlewareFunc

func NewStrictHandler(ssi StrictServerInterface, middlewares []StrictMiddlewareFunc) ServerInterface {
	return &strictHandler{ssi: ssi, middlewares: middlewares}
}

type strictHandler struct {
	ssi         StrictServerInterface
	middlewares []StrictMiddlewareFunc
}

// CreateOAuthApplication operation middleware
func (sh *strictHandler) CreateOAuthApplication(ctx echo.Context, params CreateOAuthApplicationParams) error {
	var request CreateOAuthApplicationRequestObject

	request.Params = params

	var body CreateOAuthApplicationJSONRequestBody
	if err := ctx.Bind(&body); err != nil {
		return err
	}
	request.Body = &body

	handler := func(ctx echo.Context, request interface{}) (interface{}, error) {
		return sh.ssi.CreateOAuthApplication(ctx.Request().Context(), request.(CreateOAuthApplicationRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "CreateOAuthApplication")
	}

	response, err := handler(ctx, request)

	if err != nil {
		return err
	} else if validResponse, ok := response.(CreateOAuthApplicationResponseObject); ok {
		return validResponse.VisitCreateOAuthApplicationResponse(ctx.Response())
	} else if response != nil {
		return fmt.Errorf("unexpected response type: %T", response)
	}
	return nil
}

// DeleteOAuthApplication operation middleware
func (sh *strictHandler) DeleteOAuthApplication(ctx echo.Context, id string, params DeleteOAuthApplicationParams) error {
	var request DeleteOAuthApplicationRequestObject

	request.Id = id
	request.Params = params

	handler := func(ctx echo.Context, request interface{}) (interface{}, error) {
		return sh.ssi.DeleteOAuthApplication(ctx.Request().Context(), request.(DeleteOAuthApplicationRequestObject))
	}
	for _, middleware := range sh.middlewares {
		handler = middleware(handler, "DeleteOAuthApplication")
	}

	response, err := handler(ctx, request)

	if err != nil {
		return err
	} else if validResponse, ok := response.(DeleteOAuthApplicationResponseObject); ok {
		return validResponse.VisitDeleteOAuthApplicationResponse(ctx.Response())
	} else if response != nil {
		return fmt.Errorf("unexpected response type: %T", response)
	}
	return nil
}
