#!/usr/bin/env bash

# Bash Version 3 required (it also works with ksh)
[[ ${BASH_VERSINFO[0]} -lt 3 ]] && exit 1

# Defaults
dir="."
tagname=""
msg=""
withlog=0
enablesign=1
previous_version=""
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
      echo "Usage: $0 [options...] messages..."
      echo " -d, --dir, --directory <directory>  Git directory, current directory by default."
      echo " -l, --log                           Including git log from last tag, disable by default."
      echo " -m, --message <tag message>         Tag message."
      echo " -n, --name <name>                   Tag name."
      echo " --not-sign                          Not sign, enable signing by default."
      echo " -p, --previous_tag                  Previous tag to include commits, default is tag which nearlist current commits."
      echo " -h, --help                          Get help for commands"
      exit 0
      ;;
    -d|--dir|--directory)
      # Use +1 to access next array element and unset it
      dir="${ARGS[i+1]}"
      unset 'ARGS[i]'
      skip=1
      ;;
    -l|--log)
      # It is a bool flag
      withlog=1
      ;;
    -m|--message)
      # Use +1 to access next array element and unset it
      msg="${ARGS[i+1]}"
      unset 'ARGS[i]'
      skip=1
      ;;
    -n|--name)
      # Use +1 to access next array element and unset it
      tagname="${ARGS[i+1]}"
      unset 'ARGS[i]'
      skip=1
      ;;
    --not-sign)
      # It is a bool flag
      enablesign=0
      ;;
    -p|--previous_tag)
      # Use +1 to access next array element and unset it
      previous_version="${ARGS[i+1]}"
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

if [ "$tagname" == "" ]; then
  # must be a tag name
  echo "Must provide tag name"
  exit 1;
fi

# gitcmd="git -C $dir tag --create-reflog"
gitcmd="git -C $dir tag"

# enable signing
if [[ $enablesign -ne 0 ]]; then
  gitcmd="$gitcmd -s"
fi

msgfile="/tmp/git-tag.log"
echo -n "$msg" > $msgfile

# including git logs
if [[ $withlog -ne 0 ]]; then
  # write changes log to it
  if [ "$msg" != "" ]; then
  echo "" >> $msgfile
  echo "" >> $msgfile
  fi

  cat >> $msgfile << EOF
\#\# Changes

EOF

  # previous version
  if [[ "$previous_version" == "" ]]; then
    previous_version=$(git -C $dir tag --sort=-committerdate | tac | sed -n '1p')
  fi
  echo "previous version=$previous_version"
  
  # update config to disable pager on log
  git -C $dir config --add pager.log false

  # get changes to file
  git -C $dir log --oneline --pretty="- %s" ...$previous_version >> $msgfile

  # recover config
  git -C $dir config --unset-all pager.log
fi

echo "current version=$tagname"

gitcmd="$gitcmd -F $msgfile $tagname"

eval $gitcmd

# remove tmp file
rm -f $msgfile