#!/bin/bash

set -e

# show all container command
containers=$(docker ps -a -q)
if [ -z "${containers}" ]; then
    echo "no containers found."
    exit 0
fi

docker inspect -f "{{.Name}} {{.Config.Cmd}}" $containers
