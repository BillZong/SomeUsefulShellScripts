#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=lib/common-cli.sh
source "$SCRIPT_DIR/lib/common-cli.sh"

PROGRAM_NAME=$(basename "$0")

usage() {
    cat <<EOF
Usage: $PROGRAM_NAME [options] [IMAGE...]

Show docker image architecture details for one or more images.

Options:
  -h, --help                  Show this help message.
  -j, --json                  Print a JSON document instead of plain text.

Examples:
  $PROGRAM_NAME
  $PROGRAM_NAME alpine:latest
  $PROGRAM_NAME --json ubuntu:24.04 busybox
EOF
}

json_array_from_csv() {
    local csv=$1
    local first=1
    local item

    printf '['
    OLD_IFS=$IFS
    IFS=','
    for item in $csv; do
        [ -n "$item" ] || continue
        if [ $first -eq 0 ]; then
            printf ','
        fi
        printf '"%s"' "$(suss_json_escape "$item")"
        first=0
    done
    IFS=$OLD_IFS
    printf ']'
}

JSON_OUTPUT=0
IMAGES=()

while [ "$#" -gt 0 ]; do
    case "$1" in
        -h | --help)
            usage
            exit 0
            ;;
        -j | --json)
            JSON_OUTPUT=1
            shift
            ;;
        --)
            shift
            while [ "$#" -gt 0 ]; do
                IMAGES+=("$1")
                shift
            done
            break
            ;;
        -*)
            suss_die "$PROGRAM_NAME" "unknown option: $1"
            ;;
        *)
            IMAGES+=("$1")
            shift
            ;;
    esac
done

suss_require_command "docker" "$PROGRAM_NAME"

if [ "${#IMAGES[@]}" -eq 0 ]; then
    while IFS= read -r image_id; do
        [ -n "$image_id" ] || continue
        IMAGES+=("$image_id")
    done < <(docker image ls -q | awk '!seen[$0]++')
fi

if [ "${#IMAGES[@]}" -eq 0 ]; then
    if [ "$JSON_OUTPUT" -eq 1 ]; then
        printf '{\n'
        printf '  "ok": true,\n'
        printf '  "images": []\n'
        printf '}\n'
        exit 0
    fi
    printf 'no images found\n'
    exit 0
fi

INSPECT_FILE=$(mktemp "${TMPDIR:-/tmp}/docker-show-images-arch.XXXXXX")
trap 'rm -f "$INSPECT_FILE"' EXIT

docker image inspect --format '{{.Id}}\t{{json .RepoTags}}\t{{.Architecture}}\t{{.Os}}\t{{.Variant}}' "${IMAGES[@]}" > "$INSPECT_FILE"

if [ "$JSON_OUTPUT" -eq 1 ]; then
    printf '{\n'
    printf '  "ok": true,\n'
    printf '  "images": [\n'
    first=1
    while IFS=$'\t' read -r image_id repo_tags architecture os_name variant; do
        [ -n "$image_id" ] || continue
        if [ $first -eq 0 ]; then
            printf ',\n'
        fi
        printf '    {\n'
        printf '      "id": "%s",\n' "$(suss_json_escape "$image_id")"
        printf '      "repo_tags": %s,\n' "${repo_tags:-[]}"
        printf '      "architecture": "%s",\n' "$(suss_json_escape "$architecture")"
        printf '      "os": "%s",\n' "$(suss_json_escape "$os_name")"
        printf '      "variant": "%s"\n' "$(suss_json_escape "$variant")"
        printf '    }'
        first=0
    done < "$INSPECT_FILE"
    printf '\n  ]\n'
    printf '}\n'
    exit 0
fi

while IFS=$'\t' read -r image_id repo_tags architecture os_name variant; do
    [ -n "$image_id" ] || continue
    printf '%s\t%s\t%s\t%s\t%s\n' "$image_id" "$repo_tags" "$architecture" "$os_name" "$variant"
done < "$INSPECT_FILE"
