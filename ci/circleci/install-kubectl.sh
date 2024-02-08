#!/bin/bash
set -e

if [[ -z "$KUBECTL_VERSION" ]]; then
  echo "KUBECTL_VERSION is not set"
  exit 1
fi

curl -LO https://storage.googleapis.com/kubernetes-release/release/v"${KUBECTL_VERSION}"/bin/linux/amd64/kubectl
chmod +x ./kubectl
sudo mv ./kubectl /usr/local/bin/kubectl
kubectl version --client=true

echo "export KUBECTL_VERSION=${KUBECTL_VERSION}" >> .env
