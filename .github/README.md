# CI/CD Workflows for Gloo Portal IDP Connect

This repository contains several GitHub Actions workflows to manage the continuous integration, development releases, and production releases of the Gloo Portal IDP Connect project.

## Workflows

### 1. `gloo-portal-idp-connect CI`

File: `ci-pr.yml`

This workflow is triggered on any push or pull request to the `main` branch, excluding changes to `.ci/` and markdown files.

**Jobs:**
- **style-check**: Runs Go style checks using `golangci-lint`.
- **go-unit-test**: Executes unit tests for the Go codebase after the style check passes.

This workflow ensures that code quality and correctness are maintained in the `main` branch by enforcing linting and running unit tests.

### 2. `Release`

File: `ci-release.yml`

This workflow is triggered when a release is published on GitHub.

**Jobs:**
- **style-check**: Runs Go style checks using `golangci-lint`.
- **docker-release**: Publishes a Docker image tagged with the release version.
- **release-helm**: Publishes the `gloo-portal-idp-connect` Helm chart using the release version.

This workflow automates the production release process, ensuring that the Docker image and Helm chart are built and published whenever a release is created.

### 3. `Dev Release`

File: `ci-release-dev.yaml`

This workflow is manually triggered through the GitHub Actions UI (`workflow_dispatch`).

**Jobs:**
- **set-version**: Generates a version based on the current branch and commit hash. Naming convention: `dev-$BRANCH-$HASH`.
- **docker-release**: Publishes a Docker image tagged with the generated dev version.
- **release-helm**: Publishes the `gloo-portal-idp-connect` Helm chart tagged with the dev version.

This workflow helps during development, by allowing us to build and publish dev images as-needed, instead of needing to do a release or manual creation.
The only caveat is that it is published alongside the release images, and don't automatically get cleaned up.

## Future Improvements

- **Automated Cleanup**: We should add scheduled workflows or something to automatically clean up old dev released images and Helm charts to manage storage and maintain a clean registry.
  - This could be as easy as having a weekly workflow that removes anything with the `dev-` prefix that is older than a certain date for images and charts.
- **Updated repositories**: Currently the repositories where we publish reference `gloo-mesh`, but should be updated to `gloo-ee` as this is part of the Gloo Gateway product.
