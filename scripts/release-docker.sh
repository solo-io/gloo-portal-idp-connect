#!/bin/sh

if [ "${VERSION_MINOR}" = "" ]; then
  echo "Need to pass in a VERSION_MINOR (e.g. 0.1)"
  exit 1;
fi

if [ "${REPO_DIR}" = "" ]; then
  echo "Need to pass in a REPO_DIR (e.g. gloo-portal-idp-connect)"
  exit 1;
fi

PROJECT=${PROJECT:-gloo-mesh}

gcloud components install beta --quiet
gcloud auth configure-docker us-docker.pkg.dev --quiet

if ! gcloud beta artifacts repositories describe --project gloo-mesh --location us "${REPO_DIR}"; then
  gcloud beta artifacts repositories create "${REPO_DIR}" \
    --project gloo-mesh --repository-format docker --location us \
    --labels gloo-portal-idp-connect-build-pipeline=solo-io \
    --labels version="$(echo "${VERSION_MINOR}" | sed 's/\./-/g')"

  gcloud artifacts repositories add-iam-policy-binding "${REPO_DIR}" \
    --location=us --member=allUsers --role=roles/artifactregistry.reader
fi
