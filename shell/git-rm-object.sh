#!/usr/bin/env bash

# Bash Version 3 required (it also works with ksh)
[[ ${BASH_VERSINFO[0]} -lt 3 ]] && exit 1

# Defaults
dir="."
files=()
# for skipping value
skip=''

# Put all arguments in a new array (because BASH_ARGV is read only)
ARGS=( "$@" )

for i in "${!ARGS[@]}"; do
  [[ -n "$skip" ]] && {
    skip=''
    continue
  }
  case "${ARGS[i]}" in
    -h|--help)
      echo "Usage: $0 [options...] file-objects-to-remove..."
      echo " -d, --dir, --directory <directory>  Git directory, default ."
      echo " -h, --help                          Get help for commands"
      exit 0
      ;;
    -d|--dir|--directory)
      # Use +1 to access next array element and unset it
      dir="${ARGS[i+1]}"
      unset 'ARGS[i]'
      skip=1
      ;;
    --)
      # End of arguments
      unset 'ARGS[i]'
      break
      ;;
    *)
      files+=("${ARGS[i]}")
      unset 'ARGS[i]'
      ;;
  esac
  unset 'ARGS[i]'
done

pushd $dir

for f in "${files[@]}"
do
  git filter-branch --force --index-filter "git rm --cached -f --ignore-unmatch $f" --prune-empty --tag-name-filter cat -- --all
done

popd