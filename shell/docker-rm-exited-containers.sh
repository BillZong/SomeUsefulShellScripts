#!/bin/bash

set -e

# remove all exited containers
docker rm $(docker ps -a -q -f status=exited)
