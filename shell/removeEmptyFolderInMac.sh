#!/bin/sh

find $1 -name '*.DS_Store' -type f -delete
find $1 -type d -empty -delete

