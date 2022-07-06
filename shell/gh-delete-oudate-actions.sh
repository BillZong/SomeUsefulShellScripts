#!/bin/bash

# Usage description
HELP="Usage:
  Delete all outdated actions (mode: all) using gh cli.

  Note: gh cli must be authenticated login. We only support Github right now.

  Example: $0 -o Github-owner -r repo-name -t outdated-unix-timestamp
           URL params: https://github.com/Owner/Repo

* Params:
  -h
    Print help info.
  -o
    Owner. Github account or organization account.
  -r
    Repository.
  -t
    Outdated timestamp. Actions created timestamp less than this would be deleted.
"

while getopts ho:r:t: flag; do
  case "${flag}" in
  h)
    printf "$HELP"
    exit 0
    ;;
  o)
    case $OPTARG in
    '') die "invalid string $OPTARG" ;;
    *) owner=$OPTARG ;;
    esac
    ;;
  r)
    case $OPTARG in
    '') die "invalid string $OPTARG" ;;
    *) repo=$OPTARG ;;
    esac
    ;;
  t)
    case $OPTARG in
    '' | *[!-0-9]* | - | *?-*) die "invalid number $OPTARG" ;;
    *) timestamp=$OPTARG ;;
    esac
    ;;
  esac
done

delete_oudate_actions() {
  # delete all oudated actions
  # 1. get the total count
  # 2. loop to find all id and timestamps list
  # 3. use pipe to join list and delete together
  local count=`gh api -X GET -H "Accept: application/vnd.github+json" /repos/$owner/$repo/actions/runs \
    -q '.total_count' -F per_page=1 -F page=1`

  local i=0
  
  while [[ $i -le $count ]]
  do
    gh api -X GET -H "Accept: application/vnd.github+json" \
      /repos/$owner/$repo/actions/runs \
      -F per_page=100 -F page=1 \
      -q ".workflow_runs[] | select (.created_at | . == null or fromdateiso8601 < $timestamp).id" | \
        while read id; do \
          gh api -X DELETE -H "Accept: application/vnd.github+json" /repos/$owner/$repo/actions/runs/$id; \
        done;

    ((i = i + 100))
  done
}

delete_oudate_actions
