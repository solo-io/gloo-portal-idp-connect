.PHONY: generate
generate: go-generate

# Run go-generate on all sub-packages. This generates mocks and primitives used in the portal API implementation.
.PHONY: go-generate
go-generate:
	go generate -v ./api/...

package-docker: