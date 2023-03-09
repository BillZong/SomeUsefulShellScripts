#!/usr/bin/env bash

# Bash Version 3 required (it also works with ksh)
[[ ${BASH_VERSINFO[0]} -lt 3 ]] && exit 1

# Defaults
dir="."
branch="master"
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
      echo "Usage: $0 [options...]"
      echo " Clean local git cache."
      echo "options:"
      echo " -b, --branch                        Git branch, default master"
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
    -b|--branch)
      branch="${ARGS[i+1]}"
      unset 'ARGS[i]'
      ;;
    --)
      # End of arguments
      unset 'ARGS[i]'
      break
      ;;
    *)              
      # Skip unset if our argument has not been matched
      continue
      ;;
  esac
  unset 'ARGS[i]'
done

pushd $dir

# update local ref
git update-ref -d refs/original/refs/heads/$branch
git reflog expire --expire=now --all
git gc --prune=now

popd