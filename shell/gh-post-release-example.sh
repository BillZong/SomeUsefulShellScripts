#!/bin/bash

process_ref() {
	result=`echo "$1" | grep -E "refs/tags"`
	if [ "$result" != "" ]; then
		echo "make project output and upload to Github release"
		echo ""
		make
		gh-upload-release.sh -k my_token -r -p output
	else
		echo "would not handle not tag push"
	fi
}

while read REF; do echo $REF; process_ref $REF; done
