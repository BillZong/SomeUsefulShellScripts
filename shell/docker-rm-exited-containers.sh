#!/bin/bash

set -e

# remove all exited containers
containers=$(docker ps -a -q -f status=exited)
if [ -z "${containers}" ]; then
    echo "no exited containers found."
    exit 0
fi

docker rm $containers
