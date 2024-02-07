DEPSGOBIN:=$(shell pwd)/.bin
export PATH:=$(DEPSGOBIN):$(PATH)
export GOBIN:=$(DEPSGOBIN)

.PHONY: clean
clean: clean-helm
	rm -rf vendor*
	rm -rf $(DEPSGOBIN)

.PHONY: generate
generate: go-generate

.PHONY: vendor
vendor:
	go mod tidy
	go mod download
	go mod vendor

# use the version from go.mod
GINKGO_VERSION ?= $(shell cat go.mod | grep ginkgo | cut -d" " -f2 | sed 's/^v0\.\([[:digit:]]\{1,\}\)\.[[:digit:]]\{1,\}$$/1.\1.x/')

.PHONY: install-tools
install-tools: install-go-tools install-protoc install-esbuild update-ui-deps install-go-test-coverage

# Go dependencies download
# Retry is mainly for pipelines, on big installs we can sometimes get a connect error
.PHONY: mod-download
mod-download:
	go mod download

# Go tools installation
.PHONY: install-go-tools
install-go-tools: mod-download
	mkdir -p $(DEPSGOBIN)
	go install go.uber.org/mock/mockgen@latest
	go install github.com/onsi/ginkgo/v2/ginkgo@$(GINKGO_VERSION)

# Run go-generate on all sub-packages. This generates mocks and primitives used in the portal API implementation.
.PHONY: go-generate
go-generate:
	go generate -v ./api/... ./internal/cognito/...

RELEASE := "true"
ifeq ($(TAGGED_VERSION),)
	TAGGED_VERSION := $(shell git describe --tags --dirty)
	RELEASE := "false"
endif

export VERSION ?= $(shell echo $(TAGGED_VERSION) | sed -e "s/^refs\/tags\///" | cut -c 2-)

CHART_DIR := helm
HELM_SYNC_DIR ?= _helm_sync_dir
PACKAGED_CHART_DIR ?= $(HELM_SYNC_DIR)/charts

# package helm release
.PHONY: package-helm
package-helm: helm-install set-version
	mkdir -p $(PACKAGED_CHART_DIR)
	helm package --destination $(PACKAGED_CHART_DIR) $(CHART_DIR)

# install helm
.PHONY: helm-install
helm-install:
	which helm || curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

.PHONY: publish-chart
publish-chart: package-helm
ifeq ($(RELEASE),"true")
	gsutil -m rsync -r gs://gloo-mesh-enterprise/gloo-portal-idp-connect $(HELM_SYNC_DIR)/
	helm repo index $(HELM_SYNC_DIR) --merge $(HELM_SYNC_DIR)/index.yaml
	gsutil -m rsync -r -d $(HELM_SYNC_DIR) gs://gloo-mesh-enterprise/gloo-portal-idp-connect
else
	@echo "Not a release, skipping publishing Helm chart."
endif

.PHONY: clean-helm
clean-helm:
	rm -rf $(HELM_SYNC_DIR)

SALT=bpk2CI0R944e
VERSION_MINOR=$(shell echo "$(VERSION)" | cut -d. -f1-2)
HUB ?= us-docker.pkg.dev
REPO_DIR=gloo-portal-idp-connect
REPOSITORY ?= gloo-mesh/$(REPO_DIR)/gloo-portal-idp-connect
DOCKER_IMAGE := $(HUB)/$(REPOSITORY):$(VERSION)

.PHONY: docker-build
docker-build:
	docker build . -t $(DOCKER_IMAGE)

.PHONY: docker-release
docker-release:
ifeq ($(RELEASE),"true")
	VERSION_MINOR=${VERSION_MINOR} REPO_DIR=${REPO_DIR} scripts/release-docker.sh
	docker buildx create --use --name multi-builder --platform linux/amd64,linux/arm64
	docker buildx use multi-builder
	docker buildx build --platform=linux/amd64,linux/arm64 --push . -t $(DOCKER_IMAGE)
else
	@echo "Not a release, skipping publishing Docker image."
endif

.PHONY: set-version
set-version:
	sed -e 's/%version%/'$(VERSION)'/' $(CHART_DIR)/Chart-template.yaml > $(CHART_DIR)/Chart.yaml
	sed -e 's/%version%/'$(VERSION)'/' $(CHART_DIR)/values-template.yaml > $(CHART_DIR)/values.yaml
	# .bak for Linux/Mac portability
	sed -i.bak 's/%repo-dir%/'$(REPO_DIR)'/' $(CHART_DIR)/values.yaml
	rm -rf $(CHART_DIR)/values.yaml.bak

CLUSTER ?= kind

.PHONY: load-docker
kind-load: docker-build
	kind load docker-image --name $(CLUSTER) $(DOCKER_IMAGE)

.PHONY: run-e2e-tests
run-e2e-tests: setup-test-clusters
	ginkgo run ./test/e2e

.PHONY: setup-test-clusters
setup-test-clusters:
	./env/setup/test-clusters.sh setup

.PHONY: cleanup-test-clusters
cleanup-test-clusters:
	./env/setup/test-clusters.sh cleanup
