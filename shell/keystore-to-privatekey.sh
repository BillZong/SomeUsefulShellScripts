#!/bin/sh

# Usage description
HELP="Usage:
  Export private key from keystore using node js.

  Note: node js must be installed, and this repo should be downloaded, too.

  Example: $0 -d keystore-parent-dir -a ethereum-address -p password

* Params:
  -h
    Print help info.
  -d
    Data directory, which contains the keystore/ directory.
  -a
    Ethereum address.
  -p
    Password.
"

while getopts ho:r:t: flag; do
  case "${flag}" in
  h)
    printf "$HELP"
    exit 0
    ;;
  esac
done

file=$(which $0)
path=$(dirname "$file")

# change working dir
cd "$path/../js/keystore-to-privatekey"

# use node js script, which must install package
if [[ ! -d "node_module" ]]; then
	npm install --save-dev
fi

node main.js $@