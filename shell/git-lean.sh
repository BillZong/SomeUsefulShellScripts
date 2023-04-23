#/usr/bin/env bash

# Bash Version 3 required (it also works with ksh)
[[ ${BASH_VERSINFO[0]} -lt 3 ]] && exit 1

# Defaults.
dir="."

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
      echo "  Repack and lean local git repository objects. Currently the action will not break"
      echo "  out remote tracking of repository."
      echo "options:"
      echo "  -d, --dir, --directory <directory>  Git directory, current directory by default."
      echo "  -h, --help                          Get help for commands"
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
  esac
  unset 'ARGS[i]'
done

pushd $dir

git reflog expire --expire=now --all
git fsck --full --unreachable

# repack the old files
## git repack -a -d -f --depth=250 --window=250
git repack -A -d -f --depth=250 --window=250

# Bad practice. Careful to make sure it will not drop anything you want.
## git gc --aggressive --prune=now
git gc  --auto --prune=now

popd
