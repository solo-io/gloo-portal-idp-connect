#!/bin/bash

###############################################################################################################
#
# This script is used to set up (or tear down) KinD clusters in which to run our integration tests.
#
###############################################################################################################

USAGE_MSG="Usage: test-clusters.sh [sub-command] [options] [flags]

Sub-commands:
  setup     Setup the cluster.
  cleanup   Delete the cluster.

Options:
  -c  --connector                 The connector to use for the installation. Current options are: 'cognito'. Default is 'cognito'.
      --cognito-user-pool         The user pool id to use for the installation. Default is 'us-west-2_CngONp9kI'.
      --idp-connect-version       The IDP Connect release version for the installation.
      --skip-docker-build         Do not build the IDP Connect locally.
Flags:
      --help     Show command usage.
"

# Declare all the variables that represent the inputs to this script.
# Some can also be provided as environment variables, but command line arguments take precedence.
OPERATION_TYPE=""
CONNECTOR="cognito"
USER_POOL_ID=${USER_POOL_ID:-"us-west-2_CngONp9kI"}
CLUSTER_NAME="kind"
TAGGED_VERSION=""
SKIP_DOCKER_BUILD="true"

# Parse first program argument.
case "$1" in
setup)
  OPERATION_TYPE="setup"
  shift
  ;;
cleanup)
  OPERATION_TYPE="cleanup"
  shift
  ;;
--help)
  printf "%s\n" "$USAGE_MSG"
  exit 0
  ;;
*)
  printf "Unknown operation type: %s \n\n%s\n" "$1" "$USAGE_MSG"
  exit 1
  ;;
esac

# Parse program options and flags.
# Options key and value must be space separated (e.g. --option argument)
while [ $# -gt 0 ]; do
  key="$1"

  case $key in
  -c | --connector)
    CONNECTOR="$2"
    shift
    shift
    ;;
  --cognito-user-pool)
    USER_POOL_ID="$2"
    shift
    shift
    ;;
  --cluster)
    CLUSTER_NAME="$2"
    shift
    shift
    ;;
  --idp-connect-version)
    TAGGED_VERSION="$2"
    shift
    shift
    ;;
  --skip-docker-build)
    SKIP_DOCKER_BUILD="$2"
    shift
    shift
    ;;
  *)
    printf "Unknown option: %s \n\n%s\n" "$key" "$USAGE_MSG"
    exit 1
    ;;
  esac
done

cur_dir="$(dirname "${BASH_SOURCE[0]}")"
source "$cur_dir/setup-funcs.sh"

if [ "$OPERATION_TYPE" == "setup" ]; then
  create_cluster "${CLUSTER_NAME}"

  if [ "${SKIP_DOCKER_BUILD}" != "true" ]; then
    retry make kind-load
  fi

  install_idp_connect "${CONNECTOR}" "${USER_POOL_ID}" "${TAGGED_VERSION}"

  # Apply apps
  kubectl apply -f "$cur_dir/apps"
elif [ "$OPERATION_TYPE" == "cleanup" ]; then
  delete_cluster "${CLUSTER_NAME}"
fi
