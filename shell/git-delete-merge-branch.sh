#!/usr/bin/env bash

# Bash Version 3 required (it also works with ksh)
[[ ${BASH_VERSINFO[0]} -lt 3 ]] && exit 1

# Default git directory.
git_dir=${1-"."}

# The proper way to delete branches
git -C $git_dir branch --merged | egrep -v "(^\*|master|main|develop|dev)" | xargs -L1 git -C $git_dir branch -d

# The wrong way to delete branches
## git -C $git_dir branch -v | grep "\[gone\]" | awk '{print $1}' | while read b; if [ -n "$b" ]; then do git -C $git_dir branch -D $b; fi; done
## git -C $git_dir branch -v | grep "\[gone\]" | awk '{print $1}' | xargs -L1 git -C $git_dir branch -D