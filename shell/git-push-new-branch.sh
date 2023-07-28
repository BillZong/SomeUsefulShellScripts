#!/bin/bash

REMOTE_REPO=${1:-origin}
#git push --set-upstream $REMOTE_REPO `git rev-parse --abbrev-ref HEAD`
# or git version >= 2.22
git push --set-upstream $REMOTE_REPO `git branch --show-current`
