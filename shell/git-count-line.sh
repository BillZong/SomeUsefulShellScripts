#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=lib/common-cli.sh
source "$SCRIPT_DIR/lib/common-cli.sh"

PROGRAM_NAME=$(basename "$0")

usage() {
    cat <<EOF
Usage: $PROGRAM_NAME [options]
       $PROGRAM_NAME <begin_date> <end_date> [directory] [author_name]

Count added and removed lines in a git repository for a given author and date range.

Options:
  -h, --help                  Show this help message.
  -j, --json                  Print a JSON document instead of plain text.
  -b, --begin-date DATE       Inclusive begin date, for example 2024-01-01.
  -e, --end-date DATE         Inclusive end date, for example 2026-01-01.
  -d, --directory PATH        Repository directory. Defaults to current directory.
  -a, --author NAME           Author name. Defaults to git config user.name.

Examples:
  $PROGRAM_NAME 2024-01-01 2026-01-01 .
  $PROGRAM_NAME --begin-date 2024-01-01 --end-date 2026-01-01 --author "BillZong"
  $PROGRAM_NAME --json -b 2024-01-01 -e 2026-01-01 -d .
EOF
}

JSON_OUTPUT=0
BEGIN_DATE=""
END_DATE=""
DIRECTORY="."
AUTHOR_NAME=""
POSITIONAL=()

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
        -b | --begin-date)
            [ "$#" -ge 2 ] || suss_die "$PROGRAM_NAME" "missing value for --begin-date"
            BEGIN_DATE=$2
            shift 2
            ;;
        --begin-date=*)
            BEGIN_DATE=${1#*=}
            shift
            ;;
        -e | --end-date)
            [ "$#" -ge 2 ] || suss_die "$PROGRAM_NAME" "missing value for --end-date"
            END_DATE=$2
            shift 2
            ;;
        --end-date=*)
            END_DATE=${1#*=}
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
        -a | --author)
            [ "$#" -ge 2 ] || suss_die "$PROGRAM_NAME" "missing value for --author"
            AUTHOR_NAME=$2
            shift 2
            ;;
        --author=*)
            AUTHOR_NAME=${1#*=}
            shift
            ;;
        --)
            shift
            while [ "$#" -gt 0 ]; do
                POSITIONAL+=("$1")
                shift
            done
            break
            ;;
        -*)
            suss_die "$PROGRAM_NAME" "unknown option: $1"
            ;;
        *)
            POSITIONAL+=("$1")
            shift
            ;;
    esac
done

if [ -z "$BEGIN_DATE" ] && [ "${#POSITIONAL[@]}" -ge 1 ]; then
    BEGIN_DATE=${POSITIONAL[0]}
fi
if [ -z "$END_DATE" ] && [ "${#POSITIONAL[@]}" -ge 2 ]; then
    END_DATE=${POSITIONAL[1]}
fi
if [ "$DIRECTORY" = "." ] && [ "${#POSITIONAL[@]}" -ge 3 ]; then
    DIRECTORY=${POSITIONAL[2]}
fi
if [ -z "$AUTHOR_NAME" ] && [ "${#POSITIONAL[@]}" -ge 4 ]; then
    AUTHOR_NAME=${POSITIONAL[3]}
fi

[ -n "$BEGIN_DATE" ] || suss_die "$PROGRAM_NAME" "begin date is required"
[ -n "$END_DATE" ] || suss_die "$PROGRAM_NAME" "end date is required"
[ -d "$DIRECTORY" ] || suss_die "$PROGRAM_NAME" "directory does not exist: $DIRECTORY"

if ! git -C "$DIRECTORY" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    suss_die "$PROGRAM_NAME" "not a git repository: $DIRECTORY"
fi

if [ -z "$AUTHOR_NAME" ]; then
    AUTHOR_NAME=$(git -C "$DIRECTORY" config --get user.name || true)
fi

[ -n "$AUTHOR_NAME" ] || suss_die "$PROGRAM_NAME" "author name is required and could not be inferred from git config"

stats=$(
    git -C "$DIRECTORY" log --author="$AUTHOR_NAME" \
        --after="$BEGIN_DATE" --before="$END_DATE" \
        --pretty=tformat: --numstat \
        | awk '
            $1 ~ /^[0-9]+$/ { add += $1 }
            $2 ~ /^[0-9]+$/ { subs += $2 }
            END {
                printf "%s\t%s\t%s\n", add + 0, subs + 0, (add - subs) + 0
            }
        '
)

added_lines=$(printf '%s' "$stats" | awk -F '\t' '{print $1}')
removed_lines=$(printf '%s' "$stats" | awk -F '\t' '{print $2}')
total_lines=$(printf '%s' "$stats" | awk -F '\t' '{print $3}')

if [ "$JSON_OUTPUT" -eq 1 ]; then
    printf '{\n'
    printf '  "ok": true,\n'
    printf '  "begin_date": "%s",\n' "$(suss_json_escape "$BEGIN_DATE")"
    printf '  "end_date": "%s",\n' "$(suss_json_escape "$END_DATE")"
    printf '  "directory": "%s",\n' "$(suss_json_escape "$DIRECTORY")"
    printf '  "author_name": "%s",\n' "$(suss_json_escape "$AUTHOR_NAME")"
    printf '  "added_lines": %s,\n' "$added_lines"
    printf '  "removed_lines": %s,\n' "$removed_lines"
    printf '  "total_lines": %s\n' "$total_lines"
    printf '}\n'
    exit 0
fi

printf 'added lines: %s\n' "$added_lines"
printf 'removed lines: %s\n' "$removed_lines"
printf 'total lines: %s\n' "$total_lines"
