#!/bin/bash
set -e

KIND_VERSION=${KIND_VERSION:-v0.19.0}
echo "Installing kind version $KIND_VERSION"
curl -Lo ./kind https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-linux-amd64

chmod +x ./kind
sudo mv ./kind /usr/local/bin/