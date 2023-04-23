#!/usr/bin/env bash

# Bash Version 3 required (it also works with ksh)
[[ ${BASH_VERSINFO[0]} -lt 3 ]] && exit 1

# Defaults.
dir="."
remote="origin"
branches=()

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
      echo "  Delete all remote tracking branches except main|master|dev|develop."
      echo "  And don't forget to add specific remote branch tracking to your local git configuration."
      echo "example:"
      echo "  git config --add remote.origin.fetch +refs/heads/develop:refs/remotes/origin/develop"
      echo "options:"
      echo "  -d, --dir, --directory <directory>  Git directory, current directory by default."
      echo "  -r, --remote <remote-repository>    Remote repository name, \"origin\" by default."
      echo "  -h, --help                          Get help for commands"
      exit 0
      ;;
    -d|--dir|--directory)
      # Use +1 to access next array element and unset it
      dir="${ARGS[i+1]}"
      unset 'ARGS[i]'
      skip=1
      ;;
    -r|--remote)
       # Use +1 to access next array element and unset it
      remote="${ARGS[i+1]}"
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

#git branch --remotes | egrep -v "(^\*|master|main|develop|dev)" | grep $remote | xargs -L1 git branch --delete --remotes
git branch --remotes | grep $remote |  egrep -v "/(master|main|develop|dev)" | xargs -L1 git branch --delete --remotes

popd