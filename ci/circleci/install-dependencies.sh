#!/bin/bash
set -e

# NOTE: this script is intended to be called from circleci workflows for ubuntu machine
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

git config --global url."https://${GITHUB_TOKEN}@github.com/solo-io/".insteadOf "https://github.com/solo-io/"

$SCRIPT_DIR/install-go.sh
$SCRIPT_DIR/install-kubectl.sh
$SCRIPT_DIR/install-helm.sh
$SCRIPT_DIR/install-kind.sh
