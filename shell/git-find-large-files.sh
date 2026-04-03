#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=lib/common-cli.sh
source "$SCRIPT_DIR/lib/common-cli.sh"

PROGRAM_NAME=$(basename "$0")

usage() {
    cat <<EOF
Usage: $PROGRAM_NAME [options]

Find tracked git blob objects ordered by size.

Options:
  -h, --help                  Show this help message.
  -j, --json                  Print a JSON document instead of plain text.
  -d, --dir, --directory DIR  Repository directory. Defaults to current directory.
  -l, --limit N               Return at most N results. Defaults to 0 (no limit).

Examples:
  $PROGRAM_NAME
  $PROGRAM_NAME --directory /path/to/repo
  $PROGRAM_NAME --json --limit 20
EOF
}

format_size_human() {
    local bytes=$1

    awk -v bytes="$bytes" '
        function human(value, units, unit_count, i) {
            split("B KiB MiB GiB TiB PiB EiB", units, " ")
            unit_count = length(units)
            i = 1

            while (value >= 1024 && i < unit_count) {
                value /= 1024
                i++
            }

            if (i == 1) {
                printf "%.0f%s", value, units[i]
                return
            }

            if (value >= 10) {
                printf "%.1f%s", value, units[i]
                return
            }

            printf "%.2f%s", value, units[i]
        }

        BEGIN {
            human(bytes)
        }
    '
}

json_print_files() {
    local file=$1
    local first=1
    local object_id size_bytes path size_human escaped_path escaped_human

    printf '['
    while IFS=$'\t' read -r object_id size_bytes path; do
        [ -n "$object_id" ] || continue
        size_human=$(format_size_human "$size_bytes")
        escaped_path=$(suss_json_escape "$path")
        escaped_human=$(suss_json_escape "$size_human")

        if [ $first -eq 0 ]; then
            printf ','
        fi

        printf '\n    {'
        printf '"object_id":"%s",' "$object_id"
        printf '"path":"%s",' "$escaped_path"
        printf '"size_bytes":%s,' "$size_bytes"
        printf '"size_human":"%s"' "$escaped_human"
        printf '}'
        first=0
    done < "$file"

    if [ $first -eq 0 ]; then
        printf '\n  '
    fi
    printf ']'
}

JSON_OUTPUT=0
DIRECTORY="."
LIMIT=0

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
        -d | --dir | --directory)
            [ "$#" -ge 2 ] || suss_die "$PROGRAM_NAME" "missing value for --directory"
            DIRECTORY=$2
            shift 2
            ;;
        --directory=*)
            DIRECTORY=${1#*=}
            shift
            ;;
        -l | --limit)
            [ "$#" -ge 2 ] || suss_die "$PROGRAM_NAME" "missing value for --limit"
            LIMIT=$2
            shift 2
            ;;
        --limit=*)
            LIMIT=${1#*=}
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

suss_require_command "git" "$PROGRAM_NAME"
[ -d "$DIRECTORY" ] || suss_die "$PROGRAM_NAME" "directory does not exist: $DIRECTORY"

if ! suss_is_integer "$LIMIT" || [ "$LIMIT" -lt 0 ]; then
    suss_die "$PROGRAM_NAME" "--limit must be a non-negative integer"
fi

if ! git -C "$DIRECTORY" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    suss_die "$PROGRAM_NAME" "not a git repository: $DIRECTORY"
fi

TMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/git-find-large-files.XXXXXX")
trap 'rm -rf "$TMP_DIR"' EXIT

RAW_RESULTS_FILE="$TMP_DIR/raw.tsv"
RESULTS_FILE="$TMP_DIR/results.tsv"

git -C "$DIRECTORY" rev-list --objects --all \
    | git -C "$DIRECTORY" cat-file --batch-check='%(objecttype) %(objectname) %(objectsize) %(rest)' \
    | sed -n 's/^blob //p' \
    | awk '
        {
            object_id = $1
            size_bytes = $2
            path = ""

            if (NF >= 3) {
                $1 = ""
                $2 = ""
                sub(/^  */, "")
                path = $0
            }

            printf "%s\t%s\t%s\n", object_id, size_bytes, path
        }
    ' \
    | LC_ALL=C sort -t $'\t' -k2,2nr -k1,1 > "$RAW_RESULTS_FILE"

if [ "$LIMIT" -gt 0 ]; then
    sed -n "1,${LIMIT}p" "$RAW_RESULTS_FILE" > "$RESULTS_FILE"
else
    cp "$RAW_RESULTS_FILE" "$RESULTS_FILE"
fi

TOTAL_COUNT=$(wc -l < "$RAW_RESULTS_FILE" | awk '{print $1}')
RETURNED_COUNT=$(wc -l < "$RESULTS_FILE" | awk '{print $1}')
TRUNCATED=false
if [ "$RETURNED_COUNT" -lt "$TOTAL_COUNT" ]; then
    TRUNCATED=true
fi

if [ "$JSON_OUTPUT" -eq 1 ]; then
    printf '{\n'
    printf '  "ok": true,\n'
    printf '  "directory": "%s",\n' "$(suss_json_escape "$DIRECTORY")"
    printf '  "limit": %s,\n' "$LIMIT"
    printf '  "total_count": %s,\n' "$TOTAL_COUNT"
    printf '  "returned_count": %s,\n' "$RETURNED_COUNT"
    printf '  "truncated": %s,\n' "$TRUNCATED"
    printf '  "files": '
    json_print_files "$RESULTS_FILE"
    printf '\n}\n'
    exit 0
fi

if [ "$RETURNED_COUNT" -eq 0 ]; then
    printf 'no tracked blob objects found\n'
    exit 0
fi

while IFS=$'\t' read -r object_id size_bytes path; do
    [ -n "$object_id" ] || continue
    printf '%s\t%s\t%s\t%s\n' "$object_id" "$size_bytes" "$(format_size_human "$size_bytes")" "$path"
done < "$RESULTS_FILE"

if [ "$TRUNCATED" = true ]; then
    printf '\nshowing %s of %s results\n' "$RETURNED_COUNT" "$TOTAL_COUNT"
fi
