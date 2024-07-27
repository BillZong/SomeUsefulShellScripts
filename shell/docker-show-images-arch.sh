#!/bin/bash

set -e

# show all images architecture
images=$(docker image ls -q)
if [ -z "${images}" ]; then
    echo "no images found."
    exit 0
fi

docker inspect -f "{{.ID}} {{.RepoTags}} {{.Architecture}}" $images