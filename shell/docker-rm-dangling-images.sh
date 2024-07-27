#!/bin/bash

set -e

# remove all dangling images
images=$(docker images --filter "dangling=true" -q --no-trunc)
if [ -z "${images}" ]; then
    echo "no danling images found."
    exit 0
fi

docker rmi $images
