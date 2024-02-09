#!/bin/bash

create_cluster() {
  local cluster_name=$1

  retry kind create cluster --name "${cluster_name}"
}

delete_cluster() {
  local cluster_name=$1

  retry kind delete cluster --name "${cluster_name}"
}

install_idp_connect() {
  local connector=$1
  local cognito_user_pool=$2
  local release_version=$3

  # Check connector equals 'cognito'
  if [ "$connector" != "cognito" ]; then
    echo "ERROR: Valid connectors are: 'cognito'"
    exit 1
  fi

  if [ -z ${AWS_REGION} ]; then
    echo "WARNING: AWS_REGION not set. Defaulting to us-west-2"
  fi

  AWS_REGION=${AWS_REGION:-"us-west-2"}

  if [ -z ${AWS_ACCESS_KEY_ID} ] || [ -z ${AWS_SECRET_ACCESS_KEY} ]; then
    echo "ERROR: AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY must be set"
    exit 1
  fi

values_file=$(mktemp)

cat <<EOF > "$values_file"
image:
  pullPolicy: IfNotPresent
connector: ${connector}
cognito:
  userPoolId: ${cognito_user_pool}
  aws:
    region: ${AWS_REGION}
    accessKeyId: ${AWS_ACCESS_KEY_ID}
    secretAccessKey: ${AWS_SECRET_ACCESS_KEY}
    sessionToken: ${AWS_SESSION_TOKEN}
EOF

  local helm_installation="./helm"
  if [ -n "${release_version}" ]; then
    helm repo add idp-connect https://storage.googleapis.com/gloo-mesh-enterprise/gloo-portal-idp-connect
    helm_installation="idp-connect/gloo-portal-idp-connect --version ${release_version}"
  fi

  # Install the IDP connector
  helm upgrade --install idp-connect ${helm_installation} \
    --values "$values_file" \
    --wait --timeout 30s

  rm -f "$values_file"
}

retry() {
  local n=1
  local max=5
  local delay=1
  while true; do
    "$@" && break || {
      if [[ $n -lt $max ]]; then
        ((n++))
        echo "Command failed. Attempt $n/$max:"
        sleep $delay;
      else
        fail "The command has failed after $n attempts."
      fi
    }
  done
}
