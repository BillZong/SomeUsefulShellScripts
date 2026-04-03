#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=lib/common-cli.sh
source "$SCRIPT_DIR/lib/common-cli.sh"

PROGRAM_NAME=$(basename "$0")

usage() {
    cat <<EOF
Usage: $PROGRAM_NAME [options]

Capture a one-shot CPU and memory sample for processes matching a name exactly.

Options:
  -h, --help                 Show this help message.
  -j, --json                 Print a JSON document instead of plain text.
  -p, --process-name NAME    Process name to match with pgrep -x.

Examples:
  $PROGRAM_NAME --process-name postgres
  $PROGRAM_NAME --json --process-name node
EOF
}

json_print_processes() {
    local file=$1
    local first=1
    local pid cpu_percent rss_kb vsz_kb

    printf '['
    while IFS=$'\t' read -r pid cpu_percent rss_kb vsz_kb; do
        [ -n "$pid" ] || continue
        if [ $first -eq 0 ]; then
            printf ','
        fi

        printf '\n    {'
        printf '"pid":%s,' "$pid"
        printf '"cpuPercent":%s,' "$cpu_percent"
        printf '"rssKb":%s,' "$rss_kb"
        printf '"vszKb":%s' "$vsz_kb"
        printf '}'
        first=0
    done < "$file"

    if [ $first -eq 0 ]; then
        printf '\n  '
    fi
    printf ']'
}

JSON_OUTPUT=0
PROCESS_NAME=""

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
        -p | --process-name)
            [ "$#" -ge 2 ] || suss_die "$PROGRAM_NAME" "missing value for --process-name"
            PROCESS_NAME=$2
            shift 2
            ;;
        --process-name=*)
            PROCESS_NAME=${1#*=}
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

[ -n "$PROCESS_NAME" ] || suss_die "$PROGRAM_NAME" "--process-name is required"

suss_require_command "pgrep" "$PROGRAM_NAME"
if ! command -v pidstat >/dev/null 2>&1; then
    if [[ "${OSTYPE:-}" == darwin* ]]; then
        suss_die "$PROGRAM_NAME" "pidstat command not found; install sysstat with 'brew install sysstat'"
    fi
    suss_die "$PROGRAM_NAME" "pidstat command not found; please install the sysstat package"
fi

TMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/watch-prog-memory.XXXXXX")
trap 'rm -rf "$TMP_DIR"' EXIT

PID_FILE="$TMP_DIR/pids.txt"
CPU_FILE="$TMP_DIR/cpu.tsv"
MEMORY_FILE="$TMP_DIR/memory.tsv"
RESULTS_FILE="$TMP_DIR/results.tsv"

pgrep -x "$PROCESS_NAME" > "$PID_FILE" 2>/dev/null || true
if [ ! -s "$PID_FILE" ]; then
    suss_die "$PROGRAM_NAME" "no running process matched: $PROCESS_NAME"
fi

LC_ALL=C sort -n "$PID_FILE" -o "$PID_FILE"
PID_CSV=$(paste -sd, "$PID_FILE")

pidstat -u -p "$PID_CSV" 1 1 \
    | awk '
        /^Average:/ && $3 ~ /^[0-9]+$/ {
            printf "%s\t%s\n", $3, $(NF - 2)
        }
    ' > "$CPU_FILE"

pidstat -r -p "$PID_CSV" 1 1 \
    | awk '
        /^Average:/ && $3 ~ /^[0-9]+$/ {
            printf "%s\t%s\t%s\n", $3, $(NF - 2), $(NF - 3)
        }
    ' > "$MEMORY_FILE"

: > "$RESULTS_FILE"
while IFS= read -r pid; do
    [ -n "$pid" ] || continue

    cpu_percent=$(awk -F $'\t' -v pid="$pid" '$1 == pid { print $2; exit }' "$CPU_FILE")
    rss_kb=$(awk -F $'\t' -v pid="$pid" '$1 == pid { print $2; exit }' "$MEMORY_FILE")
    vsz_kb=$(awk -F $'\t' -v pid="$pid" '$1 == pid { print $3; exit }' "$MEMORY_FILE")

    [ -n "$cpu_percent" ] || suss_die "$PROGRAM_NAME" "failed to parse CPU sample for pid: $pid"
    [ -n "$rss_kb" ] || suss_die "$PROGRAM_NAME" "failed to parse RSS sample for pid: $pid"
    [ -n "$vsz_kb" ] || suss_die "$PROGRAM_NAME" "failed to parse VSZ sample for pid: $pid"

    printf '%s\t%s\t%s\t%s\n' "$pid" "$cpu_percent" "$rss_kb" "$vsz_kb" >> "$RESULTS_FILE"
done < "$PID_FILE"

MATCHED_COUNT=$(wc -l < "$RESULTS_FILE" | awk '{print $1}')
TIMESTAMP=$(date "+%Y-%m-%dT%H:%M:%S%z")

if [ "$JSON_OUTPUT" -eq 1 ]; then
    printf '{\n'
    printf '  "ok": true,\n'
    printf '  "timestamp": "%s",\n' "$TIMESTAMP"
    printf '  "processName": "%s",\n' "$(suss_json_escape "$PROCESS_NAME")"
    printf '  "matchedCount": %s,\n' "$MATCHED_COUNT"
    printf '  "processes": '
    json_print_processes "$RESULTS_FILE"
    printf '\n}\n'
    exit 0
fi

printf 'timestamp: %s\n' "$TIMESTAMP"
printf 'process name: %s\n' "$PROCESS_NAME"
printf 'matched count: %s\n' "$MATCHED_COUNT"

while IFS=$'\t' read -r pid cpu_percent rss_kb vsz_kb; do
    [ -n "$pid" ] || continue
    printf '%s\t%s\t%s\t%s\n' "$pid" "$cpu_percent" "$rss_kb" "$vsz_kb"
done < "$RESULTS_FILE"
