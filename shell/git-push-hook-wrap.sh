#!/bin/sh

GIT_DIR_="$(git rev-parse --git-path hooks)"
# BRANCH="$(git rev-parse --symbolic --abbrev-ref $(git symbolic-ref HEAD))"

PRE_PUSH="${GIT_DIR_}/pre-push"
POST_PUSH="${GIT_DIR_}/post-push"

test -x "$PRE_PUSH" && 
	$PRE_PUSH "$@"

git push "$@"

test $? -eq 0 && 
	test -x "$POST_PUSH" && 
	$POST_PUSH "$@"
