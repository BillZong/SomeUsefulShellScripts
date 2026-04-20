#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=lib/common-cli.sh
source "$SCRIPT_DIR/lib/common-cli.sh"

PROGRAM_NAME=$(basename "$0")

usage() {
    cat <<EOF
Usage: $PROGRAM_NAME [options]

Inspect one directory level and report entry sizes.

Options:
  -h, --help               Show this help message.
  -j, --json               Print a JSON document instead of plain text.
  -d, --directory PATH     Directory to inspect. Defaults to current directory.

Examples:
  $PROGRAM_NAME
  $PROGRAM_NAME --directory ~/workspace
  $PROGRAM_NAME --json --directory .
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

encode_base64() {
    printf '%s' "$1" | base64 | tr -d '\n'
}

decode_base64() {
    local value=$1

    if printf '%s' "$value" | base64 --decode >/dev/null 2>&1; then
        printf '%s' "$value" | base64 --decode
        return
    fi

    printf '%s' "$value" | base64 -D
}

json_print_entries() {
    local file=$1
    local first=1
    local size_bytes path_base64 path size_human escaped_path escaped_human

    printf '['
    while IFS=$'\t' read -r size_bytes path_base64; do
        [ -n "$size_bytes" ] || continue
        path=$(decode_base64 "$path_base64")
        size_human=$(format_size_human "$size_bytes")
        escaped_path=$(suss_json_escape "$path")
        escaped_human=$(suss_json_escape "$size_human")

        if [ $first -eq 0 ]; then
            printf ','
        fi

        printf '\n    {'
        printf '"path":"%s",' "$escaped_path"
        printf '"sizeBytes":%s,' "$size_bytes"
        printf '"sizeHuman":"%s"' "$escaped_human"
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

suss_require_command "base64" "$PROGRAM_NAME"
suss_require_command "du" "$PROGRAM_NAME"
suss_require_command "find" "$PROGRAM_NAME"

[ -d "$DIRECTORY" ] || suss_die "$PROGRAM_NAME" "directory does not exist: $DIRECTORY"

TMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/du-dir.XXXXXX")
trap 'rm -rf "$TMP_DIR"' EXIT

RAW_RESULTS_FILE="$TMP_DIR/raw.tsv"
RESULTS_FILE="$TMP_DIR/results.tsv"

: > "$RAW_RESULTS_FILE"

while IFS= read -r -d '' entry_path; do
    size_kb=$(du -sk "$entry_path" | awk '{print $1}')
    size_bytes=$((size_kb * 1024))
    path_base64=$(encode_base64 "$entry_path")
    printf '%s\t%s\n' "$size_bytes" "$path_base64" >> "$RAW_RESULTS_FILE"
done < <(find "$DIRECTORY" -mindepth 1 -maxdepth 1 -print0)

LC_ALL=C sort -t $'\t' -k1,1nr -k2,2 "$RAW_RESULTS_FILE" > "$RESULTS_FILE"

if [ "$JSON_OUTPUT" -eq 1 ]; then
    printf '{\n'
    printf '  "ok": true,\n'
    printf '  "directory": "%s",\n' "$(suss_json_escape "$DIRECTORY")"
    printf '  "entries": '
    json_print_entries "$RESULTS_FILE"
    printf '\n}\n'
    exit 0
fi

if [ ! -s "$RESULTS_FILE" ]; then
    printf 'no entries found\n'
    exit 0
fi

while IFS=$'\t' read -r size_bytes path_base64; do
    [ -n "$size_bytes" ] || continue
    path=$(decode_base64 "$path_base64")
    printf '%s\t%s\t%s\n' "$size_bytes" "$(format_size_human "$size_bytes")" "$path"
done < "$RESULTS_FILE"
