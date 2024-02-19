# Overview

This directory contains the API for Gloo Platform Portal IDP Connect.

## Generation

This directory contains the code generation tooling for creating the GO server following the provided APIs. The tool is based on [oapi-codegen](https://github.com/deepmap/oapi-codegen), which is a tool that ingests an OpenAPI schema and generates the corresponding server and client Go code. We currently use this tool to generate only the server code, not client.
Code generation is done by running the `make generate` command.

## Configuration

Code generation is configured by the `*-config.yaml` file under each portal version directory. We currently configure `oapi-codegen`
to generate primitives used in our portal API implementation which uses [echo](https://github.com/labstack/echo) as
our server. The `Models` field generates the types defined in the `Components` section of our openAPI schema, and
`embedded-spec` generates the OpenAPI specification as a gzipped blob. See [here](https://github.com/deepmap/oapi-codegen/blob/f4cf8f9a570380c24c6ba03ae04b9393cf120692/pkg/codegen/configuration.go#L14) 
for full configuration options.

The `openapi.yaml` file contains the openAPI schema for the portal API, and is used by `oapi-codegen` to generate
the go code.

## Generated Code

The generated go code lives in `server.go`, `types.go` and `spec.go` files under each version directory in `pkg/api`.