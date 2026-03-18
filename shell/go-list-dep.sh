#!/usr/bin/env bash

set -euo pipefail

PROGRAM_NAME=$(basename "$0")
DEFAULT_TEST_IMPORT_DEPTH=${TESTIMPORTS:-1}

usage() {
    cat <<EOF
Usage: $PROGRAM_NAME [options] [PACKAGE...]

List Go package dependencies, optionally including transitive test-import
dependencies up to a configurable depth.

Options:
  -h, --help                  Show this help message.
  -j, --json                  Print a JSON document instead of plain text.
      --include-stdlib        Keep Go standard library packages in output.
      --test-import-depth N   Recursively follow TestImports to depth N.
                              Defaults to TESTIMPORTS or 1.

Examples:
  $PROGRAM_NAME .
  $PROGRAM_NAME --test-import-depth 0 ./...
  $PROGRAM_NAME --json --include-stdlib fmt
EOF
}

die() {
    echo "[$PROGRAM_NAME] $*" >&2
    exit 1
}

is_integer() {
    case "$1" in
        '' | -) return 1 ;;
        -[0-9]* | [0-9]*) return 0 ;;
        *) return 1 ;;
    esac
}

json_escape() {
    printf '%s' "$1" | sed \
        -e 's/\\/\\\\/g' \
        -e 's/"/\\"/g'
}

json_array_from_args() {
    local first=1
    local item escaped

    printf '['
    for item in "$@"; do
        escaped=$(json_escape "$item")
        if [ $first -eq 0 ]; then
            printf ','
        fi
        printf '"%s"' "$escaped"
        first=0
    done
    printf ']'
}

json_array_from_file() {
    local file=$1
    local first=1
    local item escaped

    printf '['
    while IFS= read -r item; do
        [ -n "$item" ] || continue
        escaped=$(json_escape "$item")
        if [ $first -eq 0 ]; then
            printf ','
        fi
        printf '"%s"' "$escaped"
        first=0
    done < "$file"
    printf ']'
}

filter_stdlib_file() {
    local input_file=$1
    local output_file=$2

    if [ "$INCLUDE_STDLIB" -eq 1 ]; then
        cp "$input_file" "$output_file"
        return
    fi

    if [ ! -s "$input_file" ]; then
        : > "$output_file"
        return
    fi

    comm -23 "$input_file" "$STDLIB_FILE" > "$output_file" || true
}

collect_deps() {
    local depth=$1
    shift

    local deps_file="$TMP_DIR/deps-${depth}-$$-${RANDOM}.txt"
    local raw_test_imports_file="$TMP_DIR/test-imports-raw-${depth}-$$-${RANDOM}.txt"
    local filtered_test_imports_file="$TMP_DIR/test-imports-filtered-${depth}-$$-${RANDOM}.txt"
    local next_packages=()
    local package_name

    go list -f '{{range .Deps}}{{printf "%s\n" .}}{{end}}' "$@" \
        | sed '/^$/d' \
        | sort -u > "$deps_file"
    cat "$deps_file"

    if [ "$depth" -le 0 ]; then
        return
    fi

    go list -f '{{range .TestImports}}{{printf "%s\n" .}}{{end}}' "$@" \
        | sed '/^$/d' \
        | sort -u > "$raw_test_imports_file"

    filter_stdlib_file "$raw_test_imports_file" "$filtered_test_imports_file"

    if [ ! -s "$filtered_test_imports_file" ]; then
        return
    fi

    while IFS= read -r package_name; do
        [ -n "$package_name" ] || continue
        next_packages+=("$package_name")
    done < "$filtered_test_imports_file"

    if [ "${#next_packages[@]}" -gt 0 ]; then
        collect_deps "$((depth - 1))" "${next_packages[@]}"
    fi
}

JSON_OUTPUT=0
INCLUDE_STDLIB=0
TEST_IMPORT_DEPTH=$DEFAULT_TEST_IMPORT_DEPTH
PACKAGES=()

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
        --include-stdlib)
            INCLUDE_STDLIB=1
            shift
            ;;
        --test-import-depth)
            [ "$#" -ge 2 ] || die "missing value for --test-import-depth"
            TEST_IMPORT_DEPTH=$2
            shift 2
            ;;
        --test-import-depth=*)
            TEST_IMPORT_DEPTH=${1#*=}
            shift
            ;;
        --)
            shift
            while [ "$#" -gt 0 ]; do
                PACKAGES+=("$1")
                shift
            done
            break
            ;;
        -*)
            die "unknown option: $1"
            ;;
        *)
            PACKAGES+=("$1")
            shift
            ;;
    esac
done

if ! is_integer "$TEST_IMPORT_DEPTH"; then
    die "--test-import-depth must be an integer"
fi

if [ "${#PACKAGES[@]}" -eq 0 ]; then
    PACKAGES=(.)
fi

command -v go >/dev/null 2>&1 || die "go command not found"

TMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/go-list-dep.XXXXXX")
trap 'rm -rf "$TMP_DIR"' EXIT

STDLIB_FILE="$TMP_DIR/stdlib.txt"
RAW_RESULTS_FILE="$TMP_DIR/results-raw.txt"
RESULTS_FILE="$TMP_DIR/results.txt"
FILTERED_RESULTS_FILE="$TMP_DIR/results-filtered.txt"

go list -e -f '{{if .Standard}}{{.ImportPath}}{{end}}' std | sed '/^$/d' | sort -u > "$STDLIB_FILE"
collect_deps "$TEST_IMPORT_DEPTH" "${PACKAGES[@]}" | sed '/^$/d' | sort -u > "$RAW_RESULTS_FILE"
filter_stdlib_file "$RAW_RESULTS_FILE" "$FILTERED_RESULTS_FILE"
mv "$FILTERED_RESULTS_FILE" "$RESULTS_FILE"

if [ "$JSON_OUTPUT" -eq 1 ]; then
    printf '{\n'
    printf '  "ok": true,\n'
    printf '  "packages": '
    json_array_from_args "${PACKAGES[@]}"
    printf ',\n'
    if [ "$INCLUDE_STDLIB" -eq 1 ]; then
        printf '  "include_stdlib": true,\n'
    else
        printf '  "include_stdlib": false,\n'
    fi
    printf '  "test_import_depth": %s,\n' "$TEST_IMPORT_DEPTH"
    printf '  "dependencies": '
    json_array_from_file "$RESULTS_FILE"
    printf '\n}\n'
    exit 0
fi

cat "$RESULTS_FILE"
