#!/bin/bash

if [ -t 1 ]; then
  HAS_TTY=Y
else
  HAS_TTY=N
fi

if [[ "$HAS_TTY" == Y ]]; then
  ANSI_BLUE='\033[0;34m'
  ANSI_GREEN='\033[0;32m'
  ANSI_RED='\033[0;31m'
  ANSI_YELLOW='\033[0;33m'
  ANSI_WHITE='\033[0;97m'
  ANSI_RESET='\033[0m'

  ANSI_CURSOR_TO_START_OF_PREV_LINE='\033[0F'

  ANSI_ERASE_LINE_TO_HERE='\033[1K'

  # spaces here indicate the symbol is NOT double-width
  # no space indicates the symbol IS double-width
  PRETTY_CHECKMARK="\xe2\x9c\x94 "
  HORRIFYING_EX="\xe2\x9d\x8c"
  CUTE_INFO_ICON="\xe2\x93\x98 "
  FLASHY_TRIANGLE="\xe2\x9a\xa0 "
  ZIPPY_BULLET="\xe2\x86\xb3"
else
  # if we're piped to a file, switch to ASCII
  PRETTY_CHECKMARK="[X]"
  HORRIFYING_EX="[ ]"
  CUTE_INFO_ICON="(i)"
  FLASHY_TRIANGLE="/!\\"
  ZIPPY_BULLET="*"
fi

# panic exits with an error
panic() {
  if [[ $# -gt 0 ]]; then
    echo -ne "$ANSI_RED" 1>&2
    echo -n 'ERROR:' "$@" 1>&2
    echo -e "$ANSI_RESET" 1>&2
  fi
  exit 1
}

# get_require_version reads a file in and then sends the file in
# CMD_REQUIRED_VERSION_FILE and sends it to the CMD_REQUIRED_VERSION_FUNC to grep
# out and then cleanup the version information from the file. It echos the
# version to the output.
get_required_version() {
  extract_from_file "$CMD_LABEL version" "$CMD_REQUIRED_VERSION_FILE" "$CMD_REQUIRED_VERSION_FUNC"
}

# is_required reads the first word from CMD_EXPLAIN. If it's "Optional." then
# it is not required. Otherwise, it is required.
is_required() {
  FIRST_WORD=$(echo -n $CMD_EXPLAIN | head -n1 | awk '{print $1}' | tr '[:upper:]' '[:lower:]')
  if [[ "$FIRST_WORD" = optional* ]]; then
    return 1
  else
    return 0
  fi
}

# extract_from_file extracts a piece of data from a file when given a file name
# and a function that can pull the data wanted out and echo it. It then echos
# the data found itself.
#
# The arguments are:
# * what: Describe what is being extracted for the error message
# * file: The file to pull from (local to the project directory)
# * filter: The function used to extract the desired data
PROJECT_DIR="$(cd "$SCRIPT_DIR"/../.. ; pwd)"
extract_from_file() {
  local what="$1"
  local file="$2"
  local filter="$3"

  local SRC_FILE="$PROJECT_DIR"/"$file"
  REQUIRED_VERSION=$(cat $SRC_FILE | $filter)

  if [[ -z "$REQUIRED_VERSION" ]] ; then
    panic "Unable to obtain $what from '$SRC_FILE'"
  fi

  echo $REQUIRED_VERSION
}

# labelize capitalizes the first character of the given string and echoes it.
labelize() {
  local name="$1"
  echo "$(tr '[:lower:]' '[:upper:]' <<< ${name:0:1})${name:1}"
}

# run_checks performs a series of checks on a command. It's input is all based
# on environment variables starting with CMD_*. These are documented in
# validate.sh
run_checks() {
  init_checks
  CHECK_STR="Checking $CMD_LABEL ..."
  progress_label "$CHECK_STR"

  locate_command_path
  find_command_version
  verify_command_version

  progress_complete

  engage_test_mode

  report_check_progress

  deinit_checks
}

# check_env checks to see if the given environment variable is set.
check_env() {
  REQUIRED="$1"
  ENV_VAR="$2"
  DESCRIPTION="$3"
  local pass=N

  if [[ -n "${!ENV_VAR}" ]]; then
    ENV_MISSING=N
  else
    ENV_MISSING=Y
  fi

  engage_test_mode

  if [[ "$REQUIRED" != Y && "$REQUIRED" != N ]]; then
    REQUIRED=$("$REQUIRED")
  fi

  if [[ "$ENV_MISSING" == Y ]]; then
    if [[ "$REQUIRED" == Y ]]; then
      print_nix "$ENV_VAR"
      [[ "$QUIET_MODE" == Y ]] || print_info "$ENV_VAR $DESCRIPTION"
    else
      print_warn "$ENV_VAR"
      [[ "$QUIET_MODE" == Y ]] || print_info "$ENV_VAR $DESCRIPTION"
    fi
  else
    print_check "$ENV_VAR"
    if [[ "$EXPLAIN_MORE" == Y ]]; then
      [[ "$QUIET_MODE" == Y ]] || print_info "$ENV_VAR $DESCRIPTION"
    fi
  fi
}

# protoc_special_check is used to make MAX_CONCURRENT_PROTOCS and ulimit required when neither are set.
protoc_special_check() {
  N_ULIMIT=$(ulimit -n)
  if [[ "$N_ULIMIT" -lt 1000 && -n "${!ENV_VAR}" ]]; then
    echo -n Y
  else
    echo -n N
  fi
}

# check_os outputs operating system status.
check_os() {
  echo -e "\nOperating System:"
  echo "  $(uname -smr)"
}

check_path() {
  echo -e "\nPath:"
  echo $PATH | tr ':' '\n' | sed -E 's/^/  /'
}

read_var_file() {
  [[ -f "$1" ]] && cat "$1" | sed 's/[[:space:]]*#.*$//g'
}

REDACTED_ENV_VARS=(
  $(read_var_file "$SCRIPT_DIR/redacted-env-vars.txt")
  $(read_var_file "$HOME/.redacted-env-vars.txt")
)

EXCLUDED_ENV_VARS=(
  $(read_var_file "$SCRIPT_DIR/excluded-env-vars.txt")
  $(read_var_file "$HOME/.excluded-env-vars.txt")
)

in_set() {
  local v="$1"
  shift
  for g in "$@"; do
    [[ "$v" == "$g" ]] && return 0
  done
  return 1
}

# check_full_environment outputs all environment variables with redactions.
check_full_environment() {
  echo -e "\nFull Environment:"
  echo

  for v in $(env | sed -E 's/=.*$//g' | sort); do
    if in_set "$v" "${EXCLUDED_ENV_VARS[@]}"; then
      continue
    elif in_set "$v" "${REDACTED_ENV_VARS[@]}"; then
      echo "  $v=<--REDACTED-->"
    else
      echo "  $v=${!v}"
    fi
  done

  [[ "$QUIET_MODE" == Y ]] && return 0

  local b="$ZIPPY_BULLET"
  echo
  print_warn "Double-check that all variables listed above are shareable. If not:"
  print_warn "$b Add names to show, but redact to $(dirname $0)/redacted-env-vars.txt OR ~/.redacted-env-vars.txt"
  print_warn "$b Add names to skip completely to $(dirname $0)/excluded-env-vars.txt OR ~/.excluded-env-vars.txt"
  print_warn "$b Re-run this script."
}

# engage_test_mode causes everything to fail or everything to succeed so you can
# try out the error messages.
engage_test_mode() {
  [[ -z "$DOUBLE_SECRET_TEST_MODE" ]] && return 0

  if [[ "$DOUBLE_SECRET_TEST_MODE" == "fail" ]]; then
    force_pass=N
  elif [[ "$DOUBLE_SECRET_TEST_MODE" == "pass" ]]; then
    force_pass=Y
  elif [[ "$DOUBLE_SECRET_TEST_MODE" == "rand" ]]; then
    case $(($RANDOM%2)) in
      0) force_pass=N ;;
      1) force_pass=Y ;;
    esac
  fi

  if [[ "$force_pass" == Y ]]; then
    record_error "This is a test error"
    CMD_MISSING=Y
    CMD_WRONG_VERSION=Y
    ENV_MISSING=Y
  elif [[ "$force_pass" == N ]]; then
    CMD_ERRORS=()
    CMD_PATH="/usr/bin/env"
    CMD_VERSION="0.0.0"
    CMD_MISSING=N
    CMD_WRONG_VERSION=N
    ENV_MISSING=N
  fi
}

# Progress label outputs a string describing the current check.
progress_label() {
  if [[ "$HAS_TTY" == Y ]]; then
    echo -n "$1 "
  fi
}

# Progress clears the progress string.
progress_complete() {
  if [[ "$HAS_TTY" == Y ]]; then
    echo -e "$ANSI_ERASE_LINE_TO_HERE$ANSI_CURSOR_TO_START_OF_PREV_LINE"
  fi
}

# print_check prints a string with a pretty checkmark.
print_check() {
  echo -e " $ANSI_GREEN$PRETTY_CHECKMARK$ANSI_RESET $@"
}

# print_nix prints a string with a horrifying ex in front.
print_nix() {
  echo -e " $ANSI_RED$HORRIFYING_EX $@$ANSI_RESET"
}

# print_baddish prints either a print_warn or print_nix depending on whether the current check is_required.
print_baddish() {
  if is_required; then
    print_nix "$@"
  else
    print_warn "$@"
  fi
}

# print_info prints a string with a cute little info icon.
print_info() {
  echo -e " $ANSI_BLUE$CUTE_INFO_ICON$ANSI_RESET $@"
}

# print_warn prints a string with a flashy yellow triangle.
print_warn() {
  echo -e " $ANSI_YELLOW$FLASHY_TRIANGLE$ANSI_WHITE $@$ANSI_RESET" > /dev/stderr
}

# report_check_progress outputs a report for the current check.
report_check_progress() {
  echo "$CMD_LABEL:"

  if [[ "$CMD_MISSING" == N ]]; then
    print_check "Path: $CMD_PATH"
  else
    print_baddish "Path: not found"
  fi

  if [[ "$CMD_WRONG_VERSION" == N ]]; then
    if [[ -n "$CMD_VERSION" ]]; then
      print_check "Version: $CMD_VERSION"
    fi
  else
    print_baddish "Version: $CMD_VERSION"
  fi

  for error in "${CMD_ERRORS[@]}"; do
    print_baddish "$error"
  done

  [[ "$QUIET_MODE" == Y ]] && return 0

  OS=$(uname)

  if [[ "$CMD_MISSING" == Y || "$CMD_WRONG_VERSION" == Y || "$EXPLAIN_MORE" == Y ]]; then
    print_info "$CMD_EXPLAIN"
  fi

  if [[ "$CMD_MISSING" == Y || "$CMD_WRONG_VERSION" == Y ]]; then
    if [[ "$CMD_WRONG_VERSION" == Y ]]; then
      print_info "Try uninstalling the current version and reinstalling"
    fi
    if [[ -n "$CMD_INSTALL_LINK" ]]; then
      print_info "For installation instructions see: $CMD_INSTALL_LINK"
    fi
    if [[ -n "$CMD_INSTALL_BREW" && "$OS" == "Darwin" ]]; then
      print_info "Run the following to install: brew install $CMD_INSTALL_BREW"
    fi
    if [[ -n "$CMD_INSTALL_CMD" ]]; then
      print_info "Run the following to install: $CMD_INSTALL_CMD$ANSI_RESET"
    fi
    if [[ -n "$CMD_INSTALL_MULTI_LINK" ]]; then
      print_info "If you would like to have multiple versions installed see: $CMD_INSTALL_MULTI_LINK"
    fi
    if [[ -n "$CMD_INSTALL_SPECIAL" ]]; then
      print_info "$CMD_INSTALL_SPECIAL"
    fi
  elif [[ "$CMD_WRONG_VERSION" == Y ]]; then
    if [[ -n "$CMD_INSTALL_MULTI_LINK" ]]; then
      print_info "If you would like to have multiple versions installed see: $CMD_INSTALL_MULTI_LINK"
    fi
  fi

  if [[ ("$CMD_MISSING" == Y || "$CMD_WRONG_VERSION" == Y) && "$OS" == "Darwin" ]]; then
    ARCH="$(uname -m)"
    if [[ "$ARCH" == "arm64" ]]; then
      ARCH="Arm64/Apple Silicon"
    else
      ARCH="Intel/x86_64/Amd64"
    fi
    print_warn "Make sure to install either Universal or $ARCH versions of the software, if possible."
  fi
}

# init_checks configures any missing CMD_* environment that is missing
# and can be derived. It also sets up any other initial state.
init_checks() {
  if [[ -z "$CMD_LABEL" ]]; then
    CMD_LABEL=$(labelize "$CMD_NAME")
  fi

  if [[ -z "$CMD_REQUIRED_VERSION_OP" ]]; then
    CMD_REQUIRED_VERSION_OP=">="
  fi

  CMD_ERRORS=()
  CMD_MISSING=N
  CMD_WRONG_VERSION=N
}

DEINIT_ENV=(
  CMD_LABEL
  CMD_NAME
  CMD_PATH
  CMD_VERSION
  CMD_VERSION_FUNC
  CMD_VERSION_OPT
  CMD_REQUIRED_VERSION
  CMD_REQUIRED_VERSION_OP
  CMD_REQUIRED_VERSION_FILE
  CMD_REQUIRED_VERSION_FUNC
  CMD_INSTALL_CMD
  CMD_INSTALL_LINK
  CMD_INSTALL_BREW
  CMD_INSTALL_SPECIAL
  CMD_INSTALL_MULTI_LINK
)

# deinit_checks clears all configuration from the previous check so the next
# check has a nice clean environment to work with. Globals suck.
deinit_checks() {
  for name in "${DEINIT_ENV[@]}"; do
    eval "$name=''"
  done
}

# locate_command_path finds the location of CMD_NAME and puts it in CMD_PATH. If
# the command cannot be found CMD_PATH_ERROR is set instead.
locate_command_path() {
  if which "$CMD_NAME" > /dev/null 2>&1 ; then
    CMD_PATH=$(which "$CMD_NAME")
  else
    CMD_MISSING=Y
    record_error "The $CMD_NAME command is not in your PATH"
  fi
}

# find_command_version looks up the version of the installed command. It does
# that only if CMD_PATH has been set. The way it proceeds depends on the setting
# in CMD_VERSION_OPT and CMD_VERSION_FUNC
find_command_version() {
  [[ -z "$CMD_PATH" ]] && return 0
  [[ -n "$CMD_VERSION" ]] && return 0

  local version_output=$("$CMD_PATH" $CMD_VERSION_OPT)
  if [[ "$?" == "0" ]]; then
    if [[ -n "$CMD_VERSION_FUNC" ]]; then
      version_output=$(echo "$version_output" | "$CMD_VERSION_FUNC")
      if [[ "$?" == "0" ]]; then
        CMD_VERSION="$version_output"
      else
        CMD_MISSING=Y
        record_error "Unable to extract version from $CMD_PATH $CMD_VERSION_OPT output"
      fi
    else
      CMD_VERSION="$version_output"
    fi
  else
    record_error "$CMD_PATH $CMD_VERSION_OPT failed with exit code $?"
  fi
}

# verify_command_version does not run unless either CMD_REQUIRED_VERSION or both
# of CMD_REQUIRED_VERSION_FILE are set. If CMD_REQUIRED_VERSION_FUNC is set with
# CMD_REQUIRED_VERSION_FILE, that command will be used to find the version to use
# from the file given. Also, if CMD_VERSION is not set, no attempt to verify the
# version will be tried.
#
# It adds an error if the minimum version is not met.
verify_command_version() {
  [[ -z "$CMD_VERSION" ]] && return 0

  configure_minimum_version_from_file

  [[ -z "$CMD_REQUIRED_VERSION" ]] && return 0

  case "$CMD_REQUIRED_VERSION_OP" in
    ">=")
      if ! meets_minimum_version_requirement "$CMD_VERSION" "$CMD_REQUIRED_VERSION"; then
        CMD_WRONG_VERSION=Y
        record_error "$CMD_NAME version is $CMD_VERSION, but at least $CMD_REQUIRED_VERSION is required"
      fi
      ;;
    "="|"=="|"===")
      if ! meets_exact_version_requirement "$CMD_VERSION" "$CMD_REQUIRED_VERSION"; then
        CMD_WRONG_VERSION=Y
        record_error "$CMD_NAME version is $CMD_VERSION, but exactly $CMD_REQUIRED_VERSION is required"
      fi
      ;;
    *)
      panic Unknown CMD_REQUIRED_VERSION_OP '"'"$CMD_REQUIRED_VERSION_OP"'"' specified in configuration of '"'"$CMD_NAME"'"'
      ;;
  esac
}

# configure_minimum_version_from_file extracts the version to use from the
# CMD_REQUIRED_VERSION_FILE and filters out which version to use using
# CMD_REQUIRED_VERSION_FUNC.
configure_minimum_version_from_file() {
  [[ -z "$CMD_REQUIRED_VERSION_FILE" ]] && return 0
  [[ -z "$CMD_REQUIRED_VERSION_FUNC" ]] && return 0

  CMD_REQUIRED_VERSION="$(extract_from_file "$CMD_LABEL minimum required version" "$CMD_REQUIRED_VERSION_FILE" "$CMD_REQUIRED_VERSION_FUNC")"
}

# record_error adds an error to CMD_ERRORS
record_error() {
  CMD_ERRORS+=("$1")
}

# extract_semantic_version strips non-digital bits away and tries to pull out
# the longest string of digital elements of the form #(.#)* where each # is a
# string of digits. It takes arguments on input and echos the result to output.
extract_semantic_version() {
  perl -pe '($_)=/(\d+(?:[.]\d+)+)/'
}

# vercomp compares to semantic version strings. Given two strings as arguments
# it returns 0 if they are equal, 1 if the first is greater than the second, and
# 2 if the second is greater than the first. It only compares values in the form
# of #(.#)* where each # here is zero or more digits. The string must be
# preformatted in this form first.
#
# This slick little function comes from StackOverflow here (2022-04-01):
# https://stackoverflow.com/questions/4023830/how-to-compare-two-strings-in-dot-separated-version-format-in-bash
vercomp () {
    if [[ $1 == $2 ]]
    then
        return 0
    fi
    local IFS=.
    local i ver1=($1) ver2=($2)
    # fill empty fields in ver1 with zeros
    for ((i=${#ver1[@]}; i<${#ver2[@]}; i++))
    do
        ver1[i]=0
    done
    for ((i=0; i<${#ver1[@]}; i++))
    do
        if [[ -z ${ver2[i]} ]]
        then
            # fill empty fields in ver2 with zeros
            ver2[i]=0
        fi
        if ((10#${ver1[i]} > 10#${ver2[i]}))
        then
            return 1
        fi
        if ((10#${ver1[i]} < 10#${ver2[i]}))
        then
            return 2
        fi
    done
    return 0
}

# meets_minimum_version_requirement takes two arguments. The first is the version to
# check and the second is the version to check against. It returns true (0) if
# the first version is greater than or equal to the second version. The strings
# will have their semantic version numbers extracted prior to comparison and the
# comparison will be performed using vercomp.
meets_minimum_version_requirement() {
  local have=$(echo "$1" | extract_semantic_version)
  local required=$(echo "$2" | extract_semantic_version)

  vercomp "$have" "$required"
  case $? in
    0) return 0 ;; # equal => OK
    1) return 0 ;; # greater than => OK
  esac

  return 1 # less than => Not OK
}

# meets_exact_version_requirement takes two arguments. These semantic versions
# must be exactly equal for this to return a true value.
meets_exact_version_requirement() {
  local have=$(echo "$1" | extract_semantic_version)
  local required=$(echo "$2" | extract_semantic_version)

  vercomp "$have" "$required"
  case $? in
    0) return 0 ;; # equal => OK
  esac

  return 1 # less than => Not OK
}

# vim: ft=bash
