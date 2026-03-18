#!/usr/bin/env bash

if [ -n "${SUSS_COMMON_CLI_INCLUDED:-}" ]; then
    return 0
fi
SUSS_COMMON_CLI_INCLUDED=1

suss_die() {
    local program_name=$1
    shift
    echo "[$program_name] $*" >&2
    exit 1
}

suss_require_command() {
    local command_name=$1
    local program_name=$2
    command -v "$command_name" >/dev/null 2>&1 || suss_die "$program_name" "$command_name command not found"
}

suss_is_integer() {
    [[ "$1" =~ ^-?[0-9]+$ ]]
}

suss_json_escape() {
    local s="$1"
    s="${s//\\/\\\\}"
    s="${s//\"/\\\"}"
    s="${s//$'\n'/\\n}"
    s="${s//$'\t'/\\t}"
    s="${s//$'\r'/\\r}"
    s="${s//$'\b'/\\b}"
    s="${s//$'\f'/\\f}"
    printf '%s' "$s"
}
