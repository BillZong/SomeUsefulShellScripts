#!/bin/bash

DEPTH=${1:-1}

# perfect one
find . -type d -depth $DEPTH \
	\! -name "\.*" \
	-exec git --git-dir={}/.git --work-tree=$PWD/{} fetch --prune \; \
	-exec git --git-dir={}/.git --work-tree=$PWD/{} pull origin -r \;
