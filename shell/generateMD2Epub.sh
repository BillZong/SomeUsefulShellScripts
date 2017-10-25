#!/bin/bash

if [ $# -lt 1 ]; then
  echo "请携带输出文件的名称, 如: abc.epub"
  exit 1
fi

fileNames=""
dir=$(eval pwd)
for file in $(ls $dir | grep "\.md")
  do
    name=$(ls $file)
    fileNames=$fileNames" "$name
  done

pandoc -S -t epub3 -o $1 title.txt $fileNames
