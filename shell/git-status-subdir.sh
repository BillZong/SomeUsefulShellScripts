#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=lib/common-cli.sh
source "$SCRIPT_DIR/lib/common-cli.sh"

PROGRAM_NAME=$(basename "$0")

usage() {
    cat <<EOF
Usage: $PROGRAM_NAME [options]

Inspect git repositories beneath a directory and report branch and status.

Options:
  -h, --help               Show this help message.
  -j, --json               Print a JSON document instead of plain text.
  -d, --directory PATH     Root directory to scan. Defaults to current directory.
      --depth N            Maximum directory depth to scan for child repositories.
                           Defaults to 2.

Examples:
  $PROGRAM_NAME
  $PROGRAM_NAME --directory ~/workspace --depth 4
  $PROGRAM_NAME --json --directory . --depth 2
EOF
}

json_array_from_file() {
    local file=$1
    local first=1
    local item escaped

    printf '['
    while IFS= read -r item; do
        escaped=$(suss_json_escape "$item")
        if [ $first -eq 0 ]; then
            printf ','
        fi
        printf '"%s"' "$escaped"
        first=0
    done < "$file"
    printf ']'
}

json_print_repositories() {
    local file=$1
    local first=1
    local path branch is_clean porcelain_file escaped_path escaped_branch

    printf '['
    while IFS=$'\t' read -r path branch is_clean porcelain_file; do
        [ -n "$path" ] || continue
        escaped_path=$(suss_json_escape "$path")
        escaped_branch=$(suss_json_escape "$branch")

        if [ $first -eq 0 ]; then
            printf ','
        fi

        printf '\n    {'
        printf '"path":"%s",' "$escaped_path"
        printf '"branch":"%s",' "$escaped_branch"
        printf '"isClean":%s,' "$is_clean"
        printf '"porcelain":'
        json_array_from_file "$porcelain_file"
        printf '}'
        first=0
    done < "$file"

    if [ $first -eq 0 ]; then
        printf '\n  '
    fi
    printf ']'
}

resolve_branch_name() {
    local repo_path=$1
    local branch

    if branch=$(git -C "$repo_path" symbolic-ref --quiet --short HEAD 2>/dev/null); then
        printf '%s\n' "$branch"
        return 0
    fi

    if branch=$(git -C "$repo_path" rev-parse --abbrev-ref HEAD 2>/dev/null); then
        printf '%s\n' "$branch"
        return 0
    fi

    return 1
}

JSON_OUTPUT=0
DIRECTORY="."
DEPTH=2

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
        -d | --directory)
            [ "$#" -ge 2 ] || suss_die "$PROGRAM_NAME" "missing value for --directory"
            DIRECTORY=$2
            shift 2
            ;;
        --directory=*)
            DIRECTORY=${1#*=}
            shift
            ;;
        --depth)
            [ "$#" -ge 2 ] || suss_die "$PROGRAM_NAME" "missing value for --depth"
            DEPTH=$2
            shift 2
            ;;
        --depth=*)
            DEPTH=${1#*=}
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

suss_require_command "find" "$PROGRAM_NAME"
suss_require_command "git" "$PROGRAM_NAME"

[ -d "$DIRECTORY" ] || suss_die "$PROGRAM_NAME" "directory does not exist: $DIRECTORY"

if ! suss_is_integer "$DEPTH" || [ "$DEPTH" -lt 0 ]; then
    suss_die "$PROGRAM_NAME" "--depth must be a non-negative integer"
fi

TMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/git-status-subdir.XXXXXX")
trap 'rm -rf "$TMP_DIR"' EXIT

REPO_PATHS_FILE="$TMP_DIR/repositories.txt"
RESULTS_FILE="$TMP_DIR/results.tsv"
GIT_MARKER_MAX_DEPTH=$((DEPTH + 1))

find "$DIRECTORY" -mindepth 2 -maxdepth "$GIT_MARKER_MAX_DEPTH" \( -type d -o -type f \) -name .git -print \
    | sed 's#/$##' \
    | sed 's#/.git$##' \
    | LC_ALL=C sort -u > "$REPO_PATHS_FILE"

: > "$RESULTS_FILE"

repo_index=0
while IFS= read -r repo_path; do
    [ -n "$repo_path" ] || continue

    branch=$(resolve_branch_name "$repo_path") \
        || suss_die "$PROGRAM_NAME" "failed to resolve branch for repository: $repo_path"

    porcelain_file="$TMP_DIR/porcelain-${repo_index}.txt"
    git -C "$repo_path" status --porcelain > "$porcelain_file" \
        || suss_die "$PROGRAM_NAME" "failed to inspect repository status: $repo_path"

    is_clean=false
    if [ ! -s "$porcelain_file" ]; then
        is_clean=true
    fi

    printf '%s\t%s\t%s\t%s\n' "$repo_path" "$branch" "$is_clean" "$porcelain_file" >> "$RESULTS_FILE"
    repo_index=$((repo_index + 1))
done < "$REPO_PATHS_FILE"

if [ "$JSON_OUTPUT" -eq 1 ]; then
    printf '{\n'
    printf '  "ok": true,\n'
    printf '  "directory": "%s",\n' "$(suss_json_escape "$DIRECTORY")"
    printf '  "depth": %s,\n' "$DEPTH"
    printf '  "repositories": '
    json_print_repositories "$RESULTS_FILE"
    printf '\n}\n'
    exit 0
fi

if [ ! -s "$RESULTS_FILE" ]; then
    printf 'no git repositories found\n'
    exit 0
fi

while IFS=$'\t' read -r repo_path branch is_clean porcelain_file; do
    [ -n "$repo_path" ] || continue
    printf 'path: %s\n' "$repo_path"
    printf 'branch: %s\n' "$branch"
    printf 'clean: %s\n' "$is_clean"
    if [ -s "$porcelain_file" ]; then
        printf 'porcelain:\n'
        while IFS= read -r line; do
            printf '  %s\n' "$line"
        done < "$porcelain_file"
    fi
    printf '\n'
done < "$RESULTS_FILE"
