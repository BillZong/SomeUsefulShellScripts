#!/bin/bash

if [[ $# -eq 2 ]]; then
	echo "Usage: $0 file-to-remove-comments"
	exit 1
fi

grep -o '^[^#]*' $1