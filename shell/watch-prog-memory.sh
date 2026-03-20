#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=lib/common-cli.sh
source "$SCRIPT_DIR/lib/common-cli.sh"

PROGRAM_NAME=$(basename "$0")

usage() {
    cat <<EOF
Usage: $PROGRAM_NAME [options]

Sample RSS memory usage for processes matching a program name.

Options:
  -h, --help                  Show this help message.
  -j, --json                  Print a JSON document instead of plain text.
  -p, --program NAME          Program name or pattern to match.
  -o, --output-file PATH      Append a TSV snapshot to a log file.
      --dry-run               Show what would be written without touching the output file.

Examples:
  $PROGRAM_NAME --program postgres
  $PROGRAM_NAME --json -p claude
  $PROGRAM_NAME -p redis-server --output-file /tmp/redis-memory.tsv
EOF
}

JSON_OUTPUT=0
PROGRAM_PATTERN=""
OUTPUT_FILE=""
DRY_RUN=0

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
        -p | --program)
            [ "$#" -ge 2 ] || suss_die "$PROGRAM_NAME" "missing value for --program"
            PROGRAM_PATTERN=$2
            shift 2
            ;;
        --program=*)
            PROGRAM_PATTERN=${1#*=}
            shift
            ;;
        -o | --output-file)
            [ "$#" -ge 2 ] || suss_die "$PROGRAM_NAME" "missing value for --output-file"
            OUTPUT_FILE=$2
            shift 2
            ;;
        --output-file=*)
            OUTPUT_FILE=${1#*=}
            shift
            ;;
        --dry-run)
            DRY_RUN=1
            shift
            ;;
        -*)
            suss_die "$PROGRAM_NAME" "unknown option: $1"
            ;;
        *)
            if [ -z "$PROGRAM_PATTERN" ]; then
                PROGRAM_PATTERN=$1
                shift
                continue
            fi
            suss_die "$PROGRAM_NAME" "unexpected positional argument: $1"
            ;;
    esac
done

[ -n "$PROGRAM_PATTERN" ] || suss_die "$PROGRAM_NAME" "program name is required"
suss_require_command "ps" "$PROGRAM_NAME"

TIMESTAMP=$(date "+%Y-%m-%d %H:%M:%S")
TMP_FILE=$(mktemp "${TMPDIR:-/tmp}/watch-prog-memory.XXXXXX")
trap 'rm -f "$TMP_FILE"' EXIT

ps -Ao pid=,rss=,comm=,args= \
    | awk -v pattern="$PROGRAM_PATTERN" -v self_pid="$$" '
        index($0, pattern) > 0 && $1 != self_pid && $3 != "awk" {
            pid = $1
            rss = $2
            command = $3
            $1 = ""
            $2 = ""
            $3 = ""
            sub(/^[[:space:]]+/, "", $0)
            printf "%s\t%s\t%s\t%s\n", pid, rss, command, $0
        }
    ' > "$TMP_FILE"

match_count=$(awk 'END {print NR + 0}' "$TMP_FILE")
total_rss_kb=$(awk -F '\t' '{sum += $2} END {print sum + 0}' "$TMP_FILE")

if [ -n "$OUTPUT_FILE" ]; then
    log_line=$(printf '%s\t%s\t%s\n' "$TIMESTAMP" "$PROGRAM_PATTERN" "$total_rss_kb")
    if [ "$DRY_RUN" -eq 0 ]; then
        mkdir -p "$(dirname "$OUTPUT_FILE")"
        printf '%s\n' "$log_line" >> "$OUTPUT_FILE"
    fi
fi

if [ "$JSON_OUTPUT" -eq 1 ]; then
    printf '{\n'
    printf '  "ok": true,\n'
    printf '  "timestamp": "%s",\n' "$(suss_json_escape "$TIMESTAMP")"
    printf '  "program": "%s",\n' "$(suss_json_escape "$PROGRAM_PATTERN")"
    printf '  "match_count": %s,\n' "$match_count"
    printf '  "total_rss_kb": %s,\n' "$total_rss_kb"
    if [ -n "$OUTPUT_FILE" ]; then
        printf '  "output_file": "%s",\n' "$(suss_json_escape "$OUTPUT_FILE")"
        if [ "$DRY_RUN" -eq 1 ]; then
            printf '  "dry_run": true,\n'
        else
            printf '  "dry_run": false,\n'
        fi
    fi
    printf '  "processes": [\n'
    first=1
    while IFS=$'\t' read -r pid rss_kb command args; do
        [ -n "$pid" ] || continue
        if [ $first -eq 0 ]; then
            printf ',\n'
        fi
        printf '    {\n'
        printf '      "pid": %s,\n' "$pid"
        printf '      "rss_kb": %s,\n' "$rss_kb"
        printf '      "command": "%s",\n' "$(suss_json_escape "$command")"
        printf '      "args": "%s"\n' "$(suss_json_escape "$args")"
        printf '    }'
        first=0
    done < "$TMP_FILE"
    printf '\n  ]\n'
    printf '}\n'
    exit 0
fi

printf 'timestamp: %s\n' "$TIMESTAMP"
printf 'program: %s\n' "$PROGRAM_PATTERN"
printf 'matched processes: %s\n' "$match_count"
printf 'total rss (KB): %s\n' "$total_rss_kb"
if [ -n "$OUTPUT_FILE" ]; then
    if [ "$DRY_RUN" -eq 1 ]; then
        printf 'dry run log target: %s\n' "$OUTPUT_FILE"
    else
        printf 'log appended: %s\n' "$OUTPUT_FILE"
    fi
fi

while IFS=$'\t' read -r pid rss_kb command args; do
    [ -n "$pid" ] || continue
    printf 'pid=%s rss_kb=%s command=%s args=%s\n' "$pid" "$rss_kb" "$command" "$args"
done < "$TMP_FILE"
