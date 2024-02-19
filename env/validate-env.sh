#!/usr/bin/env bash

# Instructions for adding new/updating existing command checks:
#
# 1. Add a "check_<command>" to the check_developer_environment function
#    - Please maintain alphabetical order.
# 2. Create a the "check_<command>" function.
#    - You must set:
#        CMD_NAME - the command name WITHOUT the path
#        CMD_VERSION_OPT - the command line option of the above command that
#            outputs the version
#        CMD_EXPLAIN - explain what this command is used for; it should always
#            begin with "Required." for tools that every dev must have and
#            "Optional." for the rest.
#    - You should set:
#        CMD_INSTALL_LINK - the URL with information on installation
#    - You may set:
#        CMD_INSTALL_BREW - the brew package that will install the command
#        CMD_INSTALL_CMD - a command to run to install the command
#        CMD_INSTALL_MULTI_LINK - a link with additional instructions on how
#            to install the program in multiple versions simultaneously
#        CMD_INSTALL_SPECIAL - a custom installation note
#        CMD_LABEL - if you want a pretty name different than just the
#            capitalized $CMD_NAME
#        CMD_REQUIRED_VERSION - the minimum semantic version that must be
#            installed
#        CMD_REQUIRED_VERSION_FUNC - a bash function (that you will have to
#            write) to extract the version from the version sub-command if
#            the built-in one is inadequate
#        CMD_REQUIRED_VERSION_OP - set to "=" if the version requirement is
#            exact, otherwise it will default to ">=" for greater than or equal
#        CMD_VERSION_FUNC - Normally, the version number can be automatically
#            extracted by running "$CMD_NAME $CMD_VERSION_OPT" but sometimes,
#            it's more complicated. In that case, you can provide a custom
#            function that will be given the output from the version command
#            on stdin and should extract it out and write it to stdout.
# 3. Cargo culting...
# 4. Profit!
#
# Instructions for adding new environment checks:
#
# 1. Add a "check_env <environment-variable>" line to the check_environment
#    function.
#    - Please maintain alphabetical order.
# 2. Cargo culting...
# 3. Profit!
#
# Instructions for excluding/redacting environment variables in the long
# report.
#
# The long form of this command outputs the full environment (mostly). However,
# some environment variables are skipped and some are shown to be set, but the
# actual value is replaced with "<-- REDACTED -->" because the value itself
# should not shared. If you discover another variable that needs special
# treatment, you can make the following changes:
#
# 1. Add a variable that does not matter at all or should never be shared by
#    anyone to ./dev-tools/env/excluded-env-vars.txt
# 2. Add a variable that you are too ashamed to share, but would like see when
#    others set it to ~/.excluded-env-vars.txt
# 3. Add a variable that is important to have set, but whose value should not be
#    shared by anyone to ./dev-tools/env/redacted-env-vars.txt
# 4. Add any variable values that embarrass you personally and you don't want to
#    share to ~/.redacted-env-vars.txt
#
# Comments starting with '#' are permitted and ignored in these files.
#

# please keep these in alphabetic order to make diffs better
check_developer_environment() {
  print_info Ran as: "$0 $ORIG_ARGS"
  export PATH="$PWD/.bin:$PATH"
  print_info "Added repo's .bin to PATH"

  # ALPHABETICAL ORDER!
  check_docker
  check_ginkgo
  check_git
  check_go
  check_helm
  check_kind
  check_kubectl

  if [[ "$LONG_FORM" == "Y" ]]; then
    check_os
    check_path
    check_full_environment
  fi
}




check_docker() {
  CMD_NAME=docker
  CMD_VERSION_OPT="--version"
  CMD_REQUIRED_VERSION="23.0.0"
  CMD_INSTALL_LINK="https://docs.docker.com/desktop/mac/install/"
  CMD_EXPLAIN="Required. Docker is required for kind and other tooling to create test clusters and run tests locally."
  run_checks
}

check_ginkgo() {
  CMD_NAME=ginkgo
  CMD_VERSION_OPT="version"
  CMD_REQUIRED_VERSION="2.0.0"
  CMD_INSTALL_CMD="make install-go-tools"
  CMD_INSTALL_SPECIAL="For a permanent install of ginkgo that survives 'make clean', see https://onsi.github.io/ginkgo/#installing-ginkgo"
  CMD_EXPLAIN="Required. We use ginkgo as part of the test harness which runs tests."
  run_checks
}

check_git() {
  CMD_NAME=git
  CMD_VERSION_OPT="--version"
  CMD_INSTALL_LINK="https://git-scm.com/download/mac"
  CMD_EXPLAIN="Required. All code is in git."
  run_checks
}

check_go() {
  CMD_NAME=go
  CMD_VERSION_OPT="version"
  CMD_REQUIRED_VERSION="1.21.1"
  CMD_INSTALL_BREW="go"
  CMD_INSTALL_MULTI_LINK="https://github.com/moovweb/gvm"
  CMD_INSTALL_LINK="https://go.dev/doc/install"
  CMD_EXPLAIN="Required. Almost all code is written in Golang."
  run_checks
}

get_required_go_version() {
  # This would be easier with a YAML reader :)
  # In the 'pull_request.yaml' file, grep for the 'e2e-rbac' header and 'go-version' entries,
  # then get and clean up the first 'go-version' entry after the 'e2e-rbac' header.
  grep -E '(go-version):' | \
      grep -A 1 'go-version' | \
      tail -1 | \
      awk '{ print $2 }' | \
      sed -E 's/[^0-9.]//g'
}

check_helm() {
  CMD_NAME=helm
  CMD_VERSION_OPT="version"
  CMD_REQUIRED_VERSION="3.12.3"
  CMD_REQUIRED_VERSION_OP="="
  CMD_INSTALL_LINK="https://helm.sh/"
  CMD_INSTALL_BREW="helm"
  CMD_EXPLAIN="Required. Helm is used to install Gloo Mesh."
  run_checks
}

check_kind() {
  CMD_NAME=kind
  CMD_VERSION_OPT="version"
  CMD_REQUIRED_VERSION="0.17.0"
  CMD_INSTALL_LINK="https://kind.sigs.k8s.io/#installation-and-usage"
  CMD_INSTALL_BREW="kind"
  CMD_EXPLAIN="Required. We use kind to build clusters to put into a local test mesh to run tests."
  run_checks
}

check_kubectl() {
  CMD_NAME=kubectl
  CMD_VERSION_OPT=version
  CMD_VERSION_FUNC=get_kubectl_version
  CMD_REQUIRED_VERSION="1.21" # https://docs.solo.io/gloo-mesh-enterprise/main/reference/version/versions/
  CMD_INSTALL_LINK="https://kubernetes.io/docs/tasks/tools/#kubectl"
  CMD_EXPLAIN="Required. The kubectl command allows you to interact with the Kubernetes API server on each cluster."
  run_checks
}

get_kubectl_version() {
  local KUBECTL_VERSION_STRING
  local KUBECTL_VERSION
  local KUBECTL_COMMIT

  KUBECTL_VERSION_STRING="$(cat | grep '^Client Version:')"
  KUBECTL_VERSION="$(echo "$KUBECTL_VERSION_STRING" | sed -E 's/^.*GitVersion:"v([^"]+)".*$/\1/g')"
  KUBECTL_COMMIT="$(echo "$KUBECTL_VERSION_STRING" | sed -E 's/^.*GitCommit:"([^"]+)".*$/\1/g')"
  echo "${KUBECTL_VERSION} (${KUBECTL_COMMIT})"
}

#### MAIN ##############################################################

SCRIPT_DIR="$(cd "$(dirname "$0")" ; pwd)"

source "$SCRIPT_DIR"/check-funcs.sh

EXPLAIN_MORE=N
LONG_FORM=N
QUIET_MODE=N
DOUBLE_SECRET_TEST_MODE=

if [[ "$HAS_TTY" == "N" ]]; then
  QUIET_MODE=Y
fi

print_usage() {
  cat << EOF
$0: Checks that the tools needed for Gloo Mesh development are installed.

Usage: validate.sh [ -lqh ]
  -i    show an explanation for why you might want each thing
  -l    long form lists OS and full environment
  -q    status report only, leave off installation instructions
  -h    display this message

When this command is piped to another command, wide characters and colors
will be replaced with boring ASCII and non-control-cahracter values
automatically.
EOF
}

ORIG_ARGS="$@"
while getopts 'hilqt:' _FLAG ; do
  case "${_FLAG}" in
    i) EXPLAIN_MORE=Y ;;
    l) LONG_FORM=Y ;;
    q) QUIET_MODE=Y ;;
    h) print_usage
       exit 0 ;;
    t) DOUBLE_SECRET_TEST_MODE="$OPTARG" ;;
    *) print_usage
       exit 1 ;;
  esac
done

check_developer_environment
