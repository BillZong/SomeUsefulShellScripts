#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=lib/common-cli.sh
source "$SCRIPT_DIR/lib/common-cli.sh"

PROGRAM_NAME=$(basename "$0")

usage() {
    cat <<EOF
Usage: $PROGRAM_NAME [options]

Inspect git repositories under a directory and summarize their working tree status.

Options:
  -h, --help                  Show this help message.
  -j, --json                  Print a JSON document instead of plain text.
  -d, --directory PATH        Root directory to scan. Defaults to current directory.
      --depth N               Max directory depth to scan for nested repositories. Defaults to 2.

Examples:
  $PROGRAM_NAME
  $PROGRAM_NAME --directory ~/workspace --depth 3
  $PROGRAM_NAME --json -d . --depth 2
EOF
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
        -*)
            suss_die "$PROGRAM_NAME" "unknown option: $1"
            ;;
        *)
            suss_die "$PROGRAM_NAME" "unexpected positional argument: $1"
            ;;
    esac
done

[ -d "$DIRECTORY" ] || suss_die "$PROGRAM_NAME" "directory does not exist: $DIRECTORY"
if ! suss_is_integer "$DEPTH" || [ "$DEPTH" -lt 0 ]; then
    suss_die "$PROGRAM_NAME" "--depth must be a non-negative integer"
fi

ROOT_DIR=$(cd "$DIRECTORY" && pwd)
TMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/git-status-subdir.XXXXXX")
trap 'rm -rf "$TMP_DIR"' EXIT

REPOS_FILE="$TMP_DIR/repos.txt"
STATUS_FILE="$TMP_DIR/status.txt"

if git -C "$ROOT_DIR" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    top_level=$(git -C "$ROOT_DIR" rev-parse --show-toplevel)
    if [ "$top_level" = "$ROOT_DIR" ]; then
        printf '%s\n' "$top_level" >> "$REPOS_FILE"
    fi
fi

find "$ROOT_DIR" \
    -type d -name .git -print \
    -o \( -path '*/.*' -o -path '*/node_modules' \) -prune \
    | while IFS= read -r git_dir; do
        dirname "$git_dir"
    done >> "$REPOS_FILE"

sort -u "$REPOS_FILE" -o "$REPOS_FILE"

if [ "$JSON_OUTPUT" -eq 1 ]; then
    printf '{\n'
    printf '  "ok": true,\n'
    printf '  "directory": "%s",\n' "$(suss_json_escape "$ROOT_DIR")"
    printf '  "depth": %s,\n' "$DEPTH"
    printf '  "repositories": [\n'
fi

first_repo=1
while IFS= read -r repo; do
    [ -n "$repo" ] || continue

    relative_path=${repo#"$ROOT_DIR"/}
    if [ "$repo" = "$ROOT_DIR" ]; then
        relative_path="."
    fi

    repo_depth=0
    if [ "$relative_path" != "." ]; then
        repo_depth=$(printf '%s' "$relative_path" | awk -F/ '{print NF}')
    fi
    if [ "$repo_depth" -gt "$DEPTH" ]; then
        continue
    fi

    git -C "$repo" status --porcelain=2 --branch > "$STATUS_FILE"

    branch=$(awk '/^# branch.head / {print $3; exit}' "$STATUS_FILE")
    [ -n "$branch" ] || branch="HEAD"

    upstream=$(awk '/^# branch.upstream / {print $3; exit}' "$STATUS_FILE")
    ahead=0
    behind=0
    ab_line=$(awk '/^# branch.ab / {print $0; exit}' "$STATUS_FILE")
    if [ -n "$ab_line" ]; then
        ahead=$(printf '%s\n' "$ab_line" | awk '{sub(/^\+/,"",$3); print $3 + 0}')
        behind=$(printf '%s\n' "$ab_line" | awk '{sub(/^-/,"",$4); print $4 + 0}')
    fi

    staged_count=$(awk '/^1 / || /^2 / {if (substr($3,1,1) != ".") c++} END {print c + 0}' "$STATUS_FILE")
    unstaged_count=$(awk '/^1 / || /^2 / {if (substr($3,2,1) != ".") c++} END {print c + 0}' "$STATUS_FILE")
    untracked_count=$(awk '/^\? / {c++} END {print c + 0}' "$STATUS_FILE")
    conflicted_count=$(awk '/^u / {c++} END {print c + 0}' "$STATUS_FILE")
    clean=true
    if [ "$staged_count" -gt 0 ] || [ "$unstaged_count" -gt 0 ] || [ "$untracked_count" -gt 0 ] || [ "$conflicted_count" -gt 0 ]; then
        clean=false
    fi

    if [ "$JSON_OUTPUT" -eq 1 ]; then
        if [ $first_repo -eq 0 ]; then
            printf ',\n'
        fi
        printf '    {\n'
        printf '      "path": "%s",\n' "$(suss_json_escape "$repo")"
        printf '      "relative_path": "%s",\n' "$(suss_json_escape "$relative_path")"
        printf '      "branch": "%s",\n' "$(suss_json_escape "$branch")"
        printf '      "upstream": "%s",\n' "$(suss_json_escape "$upstream")"
        printf '      "ahead": %s,\n' "$ahead"
        printf '      "behind": %s,\n' "$behind"
        printf '      "staged_count": %s,\n' "$staged_count"
        printf '      "unstaged_count": %s,\n' "$unstaged_count"
        printf '      "untracked_count": %s,\n' "$untracked_count"
        printf '      "conflicted_count": %s,\n' "$conflicted_count"
        printf '      "clean": %s\n' "$clean"
        printf '    }'
        first_repo=0
        continue
    fi

    printf '[%s] branch=%s upstream=%s ahead=%s behind=%s staged=%s unstaged=%s untracked=%s conflicted=%s clean=%s\n' \
        "$relative_path" "$branch" "$upstream" "$ahead" "$behind" "$staged_count" "$unstaged_count" "$untracked_count" "$conflicted_count" "$clean"
done < "$REPOS_FILE"

if [ "$JSON_OUTPUT" -eq 1 ]; then
    printf '\n  ]\n'
    printf '}\n'
fi
