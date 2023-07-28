#!/bin/bash

if [ $# -lt 1 ]; then
  echo "Usage: $0 dir_path_to_show -r ..."
  exit -1
fi

du -sh $1/.[!.]* $1/* | sort -h
