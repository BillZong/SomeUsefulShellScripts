#!/bin/sh

if [[ $# -lt 2 ]]; then
	echo "Usage: $0 user-id:branch remote-name repo-name(default incubator-openwhisk)"
	exit 1
fi

set -e

REPO=${3:-incubator-openwhisk}
ID=${1%:*}
BR=${1#*:}

echo "git push git@github.com:$ID/$REPO.git HEAD:$BR $2"
#git push git@github.com:$ID/$REPO.git HEAD:$BR $2
