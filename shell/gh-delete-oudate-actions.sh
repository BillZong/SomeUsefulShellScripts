#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=lib/common-cli.sh
source "$SCRIPT_DIR/lib/common-cli.sh"

PROGRAM_NAME=$(basename "$0")
PER_PAGE=100
JSON_OUTPUT=0
DRY_RUN=0
EXECUTE_MODE=0
CONFIRMED=0
OWNER=""
REPO=""
CUTOFF_EPOCH=""
GH_API_ERROR_STATUS=0
GH_API_ERROR_STDERR=""

if ! declare -F suss_die >/dev/null 2>&1; then
    suss_die() {
        printf '%s: %s\n' "$1" "$2" >&2
        exit 1
    }
fi

if ! declare -F suss_is_integer >/dev/null 2>&1; then
    suss_is_integer() {
        case "$1" in
            '' | *[!0-9-]* | -)
                return 1
                ;;
            *)
                return 0
                ;;
        esac
    }
fi

if ! declare -F suss_require_command >/dev/null 2>&1; then
    suss_require_command() {
        command -v "$1" >/dev/null 2>&1 || suss_die "$2" "required command not found: $1"
    }
fi

if ! declare -F suss_json_escape >/dev/null 2>&1; then
    suss_json_escape() {
        printf '%s' "$1" | \
            sed 's/\\/\\\\/g; s/"/\\"/g; s/'"$'\t''"/\\t/g; s/'"$'\r''"/\\r/g; s/'"$'\n''"/\\n/g'
    }
fi

usage() {
    cat <<EOF
Usage: $PROGRAM_NAME [options]

Inspect and optionally delete outdated GitHub Actions workflow runs.

Modes:
  --dry-run                 Inspect matching workflow runs without deleting.
  --execute --yes           Delete matching workflow runs after explicit confirmation.

Required options:
  -o, --owner OWNER         GitHub account or organization owner.
  -r, --repo REPO           Repository name.
  -t, --cutoff-epoch <unix-seconds>
                            Delete workflow runs with created_at older than the given Unix seconds.

Other options:
  -j, --json                Print machine-readable JSON output.
  -h, --help                Show this help text.

Examples:
  $PROGRAM_NAME --dry-run --owner acme --repo widget --cutoff-epoch 1700000000
  $PROGRAM_NAME --json --execute --yes --owner acme --repo widget --cutoff-epoch 1700000000
EOF
}

emit_error() {
    local message=$1
    local exit_code=${2:-1}
    local gh_stderr=${3:-}
    local deleted_count=0

    if [ "$JSON_OUTPUT" -eq 1 ]; then
        if [ -n "${DELETED_IDS_FILE:-}" ] && [ -f "${DELETED_IDS_FILE:-}" ]; then
            deleted_count=$(wc -l < "$DELETED_IDS_FILE" | awk '{print $1}')
        fi

        printf '{\n'
        printf '  "ok": false,\n'
        printf '  "error": "%s",\n' "$(suss_json_escape "$message")"
        printf '  "exitCode": %s' "$exit_code"
        if [ -n "$gh_stderr" ]; then
            printf ',\n'
            printf '  "ghStderr": "%s",\n' "$(suss_json_escape "$gh_stderr")"
        else
            printf ',\n'
        fi
        printf '  "deletedRunCount": %s,\n' "$deleted_count"
        printf '  "deletedRunIds": '
        if [ -n "${DELETED_IDS_FILE:-}" ] && [ -f "${DELETED_IDS_FILE:-}" ]; then
            json_print_ids "$DELETED_IDS_FILE"
        else
            printf '[]'
        fi
        printf '\n'
        printf '}\n'
    else
        printf '%s: %s\n' "$PROGRAM_NAME" "$message" >&2
        if [ -n "$gh_stderr" ]; then
            printf '%s\n' "$gh_stderr" >&2
        fi
        if [ -n "${DELETED_IDS_FILE:-}" ] && [ -f "${DELETED_IDS_FILE:-}" ]; then
            deleted_count=$(wc -l < "$DELETED_IDS_FILE" | awk '{print $1}')
            if [ "$deleted_count" -gt 0 ]; then
                printf 'deleted run ids before failure:\n' >&2
                while IFS= read -r deleted_id; do
                    [ -n "$deleted_id" ] || continue
                    printf '  - %s\n' "$deleted_id" >&2
                done < "$DELETED_IDS_FILE"
            fi
        fi
    fi

    exit "$exit_code"
}

json_print_runs() {
    local file=$1
    local first=1
    local id created_at name head_branch

    printf '['
    while IFS=$'\t' read -r id created_at name head_branch; do
        [ -n "$id" ] || continue

        if [ $first -eq 0 ]; then
            printf ','
        fi

        printf '\n    {'
        printf '"id":%s,' "$id"
        printf '"createdAt":"%s",' "$(suss_json_escape "$created_at")"
        printf '"name":"%s",' "$(suss_json_escape "$name")"
        printf '"headBranch":"%s"' "$(suss_json_escape "$head_branch")"
        printf '}'
        first=0
    done < "$file"

    if [ $first -eq 0 ]; then
        printf '\n  '
    fi
    printf ']'
}

json_print_ids() {
    local file=$1
    local first=1
    local id

    printf '['
    while IFS= read -r id; do
        [ -n "$id" ] || continue

        if [ $first -eq 0 ]; then
            printf ','
        fi

        printf '%s' "$id"
        first=0
    done < "$file"
    printf ']'
}

gh_api() {
    local stderr_file output status

    stderr_file=$(mktemp "${TMPDIR:-/tmp}/gh-delete-oudate-actions-ghstderr.XXXXXX")
    set +e
    output=$(gh api "$@" 2>"$stderr_file")
    status=$?
    set -e

    if [ "$status" -ne 0 ]; then
        GH_API_ERROR_STATUS=$status
        GH_API_ERROR_STDERR=$(cat "$stderr_file")
        rm -f "$stderr_file"
        return "$status"
    fi

    GH_API_ERROR_STATUS=0
    GH_API_ERROR_STDERR=""
    rm -f "$stderr_file"

    if [ -n "$output" ]; then
        printf '%s\n' "$output"
    fi
}

fetch_total_count() {
    local output_file=$1

    gh_api -X GET -H "Accept: application/vnd.github+json" \
        "/repos/$OWNER/$REPO/actions/runs" \
        -q '.total_count' \
        -F per_page=1 \
        -F page=1 > "$output_file" || emit_error "gh api request failed" "$GH_API_ERROR_STATUS" "$GH_API_ERROR_STDERR"
}

collect_candidates() {
    local output_file=$1
    local total_count=$2
    local pages page query

    : > "$output_file"

    if [ "$total_count" -le 0 ]; then
        return
    fi

    pages=$(( (total_count + PER_PAGE - 1) / PER_PAGE ))
    query=".workflow_runs[] | select(.created_at != null and (.created_at | fromdateiso8601) < $CUTOFF_EPOCH) | [.id, .created_at, (.name // \"\"), (.head_branch // \"\")] | @tsv"

    page=1
    while [ "$page" -le "$pages" ]; do
        gh_api -X GET -H "Accept: application/vnd.github+json" \
            "/repos/$OWNER/$REPO/actions/runs" \
            -q "$query" \
            -F per_page="$PER_PAGE" \
            -F page="$page" >> "$output_file" || emit_error "gh api request failed" "$GH_API_ERROR_STATUS" "$GH_API_ERROR_STDERR"
        page=$((page + 1))
    done
}

delete_candidates() {
    local candidates_file=$1
    local deleted_ids_file=$2
    local id created_at name head_branch

    : > "$deleted_ids_file"

    while IFS=$'\t' read -r id created_at name head_branch; do
        [ -n "$id" ] || continue
        gh_api -X DELETE -H "Accept: application/vnd.github+json" "/repos/$OWNER/$REPO/actions/runs/$id" || emit_error "gh api request failed" "$GH_API_ERROR_STATUS" "$GH_API_ERROR_STDERR"
        printf '%s\n' "$id" >> "$deleted_ids_file"
    done < "$candidates_file"
}

print_plain_output() {
    local candidates_file=$1
    local deleted_ids_file=$2
    local matched_count=$3
    local deleted_count=$4
    local id created_at name head_branch

    printf 'Repository: %s/%s\n' "$OWNER" "$REPO"
    printf 'Cutoff epoch: %s\n' "$CUTOFF_EPOCH"
    if [ "$DRY_RUN" -eq 1 ]; then
        printf 'Mode: dry-run\n'
    else
        printf 'Mode: execute\n'
    fi
    if [ "$CONFIRMED" -eq 1 ]; then
        printf 'Confirmed: yes\n'
    else
        printf 'Confirmed: no\n'
    fi
    printf 'Matched runs: %s\n' "$matched_count"
    printf 'Deleted runs: %s\n' "$deleted_count"

    if [ "$matched_count" -gt 0 ]; then
        printf 'Matched run details:\n'
        while IFS=$'\t' read -r id created_at name head_branch; do
            [ -n "$id" ] || continue
            printf '  - id=%s created_at=%s name=%s head_branch=%s\n' "$id" "${created_at:-<none>}" "${name:-<none>}" "${head_branch:-<none>}"
        done < "$candidates_file"
    fi

    if [ "$deleted_count" -gt 0 ]; then
        printf 'Deleted run ids:\n'
        while IFS= read -r id; do
            [ -n "$id" ] || continue
            printf '  - %s\n' "$id"
        done < "$deleted_ids_file"
    fi
}

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
        --dry-run)
            DRY_RUN=1
            shift
            ;;
        --execute)
            EXECUTE_MODE=1
            shift
            ;;
        --yes)
            CONFIRMED=1
            shift
            ;;
        -o | --owner)
            [ "$#" -ge 2 ] || emit_error "missing value for --owner"
            OWNER=$2
            shift 2
            ;;
        --owner=*)
            OWNER=${1#*=}
            shift
            ;;
        -r | --repo)
            [ "$#" -ge 2 ] || emit_error "missing value for --repo"
            REPO=$2
            shift 2
            ;;
        --repo=*)
            REPO=${1#*=}
            shift
            ;;
        -t | --cutoff-epoch)
            [ "$#" -ge 2 ] || emit_error "missing value for --cutoff-epoch"
            CUTOFF_EPOCH=$2
            shift 2
            ;;
        --cutoff-epoch=*)
            CUTOFF_EPOCH=${1#*=}
            shift
            ;;
        --)
            shift
            [ "$#" -eq 0 ] || emit_error "unexpected positional arguments: $*"
            ;;
        -*)
            emit_error "unknown option: $1"
            ;;
        *)
            emit_error "unexpected positional argument: $1"
            ;;
    esac
done

[ -n "$OWNER" ] || emit_error "missing required --owner"
[ -n "$REPO" ] || emit_error "missing required --repo"
[ -n "$CUTOFF_EPOCH" ] || emit_error "missing required --cutoff-epoch"

if ! suss_is_integer "$CUTOFF_EPOCH" || [ "$CUTOFF_EPOCH" -lt 0 ]; then
    emit_error "--cutoff-epoch must be a non-negative integer"
fi

if [ "$DRY_RUN" -eq 1 ] && [ "$EXECUTE_MODE" -eq 1 ]; then
    emit_error "choose either --dry-run or --execute"
fi
if [ "$CONFIRMED" -eq 1 ] && [ "$EXECUTE_MODE" -eq 0 ]; then
    emit_error "--yes requires --execute"
fi
if [ "$EXECUTE_MODE" -eq 1 ] && [ "$CONFIRMED" -eq 0 ]; then
    emit_error "--execute requires --yes"
fi
if [ "$DRY_RUN" -eq 0 ] && [ "$EXECUTE_MODE" -eq 0 ]; then
    emit_error "refusing destructive execution without --dry-run or --execute --yes"
fi

if ! command -v gh >/dev/null 2>&1; then
    emit_error "required command not found: gh"
fi

TMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/gh-delete-oudate-actions.XXXXXX")
trap 'rm -rf "$TMP_DIR"' EXIT

CANDIDATES_FILE="$TMP_DIR/candidates.tsv"
DELETED_IDS_FILE="$TMP_DIR/deleted-ids.txt"
TOTAL_COUNT_FILE="$TMP_DIR/total-count.txt"
fetch_total_count "$TOTAL_COUNT_FILE"
TOTAL_COUNT=$(tr -d '\n' < "$TOTAL_COUNT_FILE")

if ! suss_is_integer "$TOTAL_COUNT" || [ "$TOTAL_COUNT" -lt 0 ]; then
    emit_error "unexpected total_count from gh api: $TOTAL_COUNT"
fi

collect_candidates "$CANDIDATES_FILE" "$TOTAL_COUNT"
MATCHED_COUNT=$(wc -l < "$CANDIDATES_FILE" | awk '{print $1}')
DELETED_COUNT=0

if [ "$EXECUTE_MODE" -eq 1 ]; then
    delete_candidates "$CANDIDATES_FILE" "$DELETED_IDS_FILE"
    DELETED_COUNT=$(wc -l < "$DELETED_IDS_FILE" | awk '{print $1}')
else
    : > "$DELETED_IDS_FILE"
fi

if [ "$JSON_OUTPUT" -eq 1 ]; then
    printf '{\n'
    printf '  "ok": true,\n'
    printf '  "repository": "%s",\n' "$(suss_json_escape "$OWNER/$REPO")"
    printf '  "owner": "%s",\n' "$(suss_json_escape "$OWNER")"
    printf '  "repo": "%s",\n' "$(suss_json_escape "$REPO")"
    printf '  "cutoffEpoch": %s,\n' "$CUTOFF_EPOCH"
    if [ "$DRY_RUN" -eq 1 ]; then
        printf '  "dryRun": true,\n'
        printf '  "mode": "dry-run",\n'
    else
        printf '  "dryRun": false,\n'
        printf '  "mode": "execute",\n'
    fi
    if [ "$CONFIRMED" -eq 1 ]; then
        printf '  "confirmed": true,\n'
    else
        printf '  "confirmed": false,\n'
    fi
    printf '  "matchedRunCount": %s,\n' "$MATCHED_COUNT"
    printf '  "deletedRunCount": %s,\n' "$DELETED_COUNT"
    printf '  "matchedRuns": '
    json_print_runs "$CANDIDATES_FILE"
    printf ',\n'
    printf '  "deletedRunIds": '
    json_print_ids "$DELETED_IDS_FILE"
    printf '\n}\n'
    exit 0
fi

print_plain_output "$CANDIDATES_FILE" "$DELETED_IDS_FILE" "$MATCHED_COUNT" "$DELETED_COUNT"
