#!/bin/bash
set -e

if [ $# -lt 2 ]; then
	echo "Usage: $0 {DIRECTORY} {DEPTH}"
	exit 1
fi

DIRECTORY=$1
DEPTH=$2

# perfect one
find $DIRECTORY -type d -depth $DEPTH \
	\! -name "\.*" \
	-exec echo "Working directory: "{} \; \
	-exec git --git-dir={}/.git --work-tree=$PWD/{} fetch --prune \; \
	-exec git --git-dir={}/.git --work-tree=$PWD/{} pull origin -r \;
