#!/bin/bash

if [[ $# -lt 2 ]]; then
	echo "Usage: $0 repo-name pr-number"
	exit 1
fi

set -e

REPO=$1
: ${REPO:?"repo name must be given"}

PRN=$2
: ${PRN:?"pull request number must be given"}

git fetch $REPO "pull/$PRN/head:pr-$PRN"
git checkout "pr-$PRN"
git rebase master

