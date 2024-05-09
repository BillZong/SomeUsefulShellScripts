#!/bin/bash

set -e

HELP="Usage: \n$0 {begin_date, eg:2023-05-09} {end_date, eg: 2024-05-09} [{count_directory}|"."] [{author_name}|{git config user name}]"

if [ "$#" -lt 2 ]; then
	echo $HELP
	exit -1
fi

BEGIN=$1
END=$2
DIRECTORY=${3:-'.'}
AUTHOR_NAME=$4

if [ "$AUTHOR_NAME" == "" ]; then
    AUTHOR_NAME=$(git config --get user.name)
fi

git --git-dir=$DIRECTORY/.git log --author="$AUTHOR_NAME" \
--after="$BEGIN" --before="$END" \
--pretty=tformat: --numstat \
| awk '{ add += $1 ; subs += $2 ; loc += $1 - $2 } \
END { printf "added lines: %s removed lines : \
%s total lines: %s\n",add,subs,loc }'
