#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=lib/common-cli.sh
source "$SCRIPT_DIR/lib/common-cli.sh"

PROGRAM_NAME=$(basename "$0")

usage() {
    cat <<EOF
Usage: $PROGRAM_NAME [options]

Inspect local Docker images and report repository tags and architecture.

Options:
  -h, --help    Show this help message.
  -j, --json    Print a JSON document instead of plain text.

Examples:
  $PROGRAM_NAME
  $PROGRAM_NAME --json
EOF
}

normalize_repo_tags_json() {
    local value=$1

    if [ "$value" = "null" ]; then
        printf '[]'
        return
    fi

    printf '%s' "$value"
}

json_print_images() {
    local file=$1
    local first=1
    local image_id repo_tags_json architecture escaped_id escaped_architecture

    printf '['
    while IFS=$'\t' read -r image_id repo_tags_json architecture; do
        [ -n "$image_id" ] || continue
        escaped_id=$(suss_json_escape "$image_id")
        escaped_architecture=$(suss_json_escape "$architecture")
        repo_tags_json=$(normalize_repo_tags_json "$repo_tags_json")

        if [ $first -eq 0 ]; then
            printf ','
        fi

        printf '\n    {'
        printf '"id":"%s",' "$escaped_id"
        printf '"repoTags":%s,' "$repo_tags_json"
        printf '"architecture":"%s"' "$escaped_architecture"
        printf '}'
        first=0
    done < "$file"

    if [ $first -eq 0 ]; then
        printf '\n  '
    fi
    printf ']'
}

JSON_OUTPUT=0

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
            [ "$#" -eq 0 ] || suss_die "$PROGRAM_NAME" "unexpected positional arguments: $*"
            break
            ;;
        -*)
            suss_die "$PROGRAM_NAME" "unknown option: $1"
            ;;
        *)
            suss_die "$PROGRAM_NAME" "unexpected positional argument: $1"
            ;;
    esac
done

suss_require_command "docker" "$PROGRAM_NAME"

TMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/docker-show-images-arch.XXXXXX")
trap 'rm -rf "$TMP_DIR"' EXIT

IMAGE_IDS_FILE="$TMP_DIR/image-ids.txt"
RESULTS_FILE="$TMP_DIR/results.tsv"
DOCKER_STDERR_FILE="$TMP_DIR/docker.stderr"

docker image ls --no-trunc --quiet > "$IMAGE_IDS_FILE" 2> "$DOCKER_STDERR_FILE" \
    || suss_die "$PROGRAM_NAME" "$(tr '\n' ' ' < "$DOCKER_STDERR_FILE" | sed 's/[[:space:]]\+/ /g; s/^ //; s/ $//')"

LC_ALL=C sort -u "$IMAGE_IDS_FILE" -o "$IMAGE_IDS_FILE"
: > "$RESULTS_FILE"

while IFS= read -r image_id; do
    [ -n "$image_id" ] || continue

    inspect_output=$(docker image inspect --format '{{.ID}}{{printf "\t"}}{{json .RepoTags}}{{printf "\t"}}{{.Architecture}}' "$image_id" 2> "$DOCKER_STDERR_FILE") \
        || suss_die "$PROGRAM_NAME" "$(tr '\n' ' ' < "$DOCKER_STDERR_FILE" | sed 's/[[:space:]]\+/ /g; s/^ //; s/ $//')"

    printf '%s\n' "$inspect_output" >> "$RESULTS_FILE"
done < "$IMAGE_IDS_FILE"

if [ "$JSON_OUTPUT" -eq 1 ]; then
    printf '{\n'
    printf '  "ok": true,\n'
    printf '  "images": '
    json_print_images "$RESULTS_FILE"
    printf '\n}\n'
    exit 0
fi

if [ ! -s "$RESULTS_FILE" ]; then
    printf 'no images found\n'
    exit 0
fi

while IFS=$'\t' read -r image_id repo_tags_json architecture; do
    [ -n "$image_id" ] || continue
    repo_tags_json=$(normalize_repo_tags_json "$repo_tags_json")
    printf '%s\t%s\t%s\n' "$image_id" "$architecture" "$repo_tags_json"
done < "$RESULTS_FILE"
