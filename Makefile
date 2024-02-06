.PHONY: generate
generate: go-generate

install-tools:
	go install go.uber.org/mock/mockgen@latest

# Run go-generate on all sub-packages. This generates mocks and primitives used in the portal API implementation.
.PHONY: go-generate
go-generate:
	go generate -v ./api/... ./cognito/...

package-docker: