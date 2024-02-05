// Package v1 provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen/v2 version v2.1.0 DO NOT EDIT.
package v1

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"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",
	"Ly4AAA==",
}

// GetSwagger returns the content of the embedded swagger specification file
// or error if failed to decode
func decodeSpec() ([]byte, error) {
	zipped, err := base64.StdEncoding.DecodeString(strings.Join(swaggerSpec, ""))
	if err != nil {
		return nil, fmt.Errorf("error base64 decoding spec: %w", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(zipped))
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}

	return buf.Bytes(), nil
}

var rawSpec = decodeSpecCached()

// a naive cached of a decoded swagger spec
func decodeSpecCached() func() ([]byte, error) {
	data, err := decodeSpec()
	return func() ([]byte, error) {
		return data, err
	}
}

// Constructs a synthetic filesystem for resolving external references when loading openapi specifications.
func PathToRawSpec(pathToFile string) map[string]func() ([]byte, error) {
	res := make(map[string]func() ([]byte, error))
	if len(pathToFile) > 0 {
		res[pathToFile] = rawSpec
	}

	return res
}

// GetSwagger returns the Swagger specification corresponding to the generated code
// in this file. The external references of Swagger specification are resolved.
// The logic of resolving external references is tightly connected to "import-mapping" feature.
// Externally referenced files must be embedded in the corresponding golang packages.
// Urls can be supported but this task was out of the scope.
func GetSwagger() (swagger *openapi3.T, err error) {
	resolvePath := PathToRawSpec("")

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, url *url.URL) ([]byte, error) {
		pathToFile := url.String()
		pathToFile = path.Clean(pathToFile)
		getSpec, ok := resolvePath[pathToFile]
		if !ok {
			err1 := fmt.Errorf("path not found: %s", pathToFile)
			return nil, err1
		}
		return getSpec()
	}
	var specData []byte
	specData, err = rawSpec()
	if err != nil {
		return
	}
	swagger, err = loader.LoadFromData(specData)
	if err != nil {
		return
	}
	return
}
