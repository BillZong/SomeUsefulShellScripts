#!/bin/bash

set -e

# remove all dangling images
docker rmi $(docker images --filter "dangling=true" -q --no-trunc)
