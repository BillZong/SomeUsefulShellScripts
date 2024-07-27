#!/usr/bin/sh

# This is a demo shell script from pandoc demo "ProGit"
# It uses the perl command and regex expression to change
# file's content to image we need.

perl -i -0pe \
	's/^Insert\s*(.*)\.png\s*\n([^\n]*)$/!\[\2](..\/figures\/\1-tn.png)/mg' \
	*/*.markdown

