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
