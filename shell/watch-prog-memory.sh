#!/bin/bash

set -e

# make sure pidstat is installed. If not, make sure to install sysstat package.
if ! command -v pidstat &> /dev/null
then
    # If it's in MacOS, suggest installing via brew
    if [[ "$OSTYPE" == "darwin"* ]]; then
        echo "Could not find pidstat, please install sysstat package via 'brew install sysstat'."
        echo "Then run it in crontab with: */1 * * * * $0 <program_name> [/path/to/output.log]"
        exit 1
    else
        echo "Could not find pidstat, please install sysstat package."
        echo "Then run it in crontab with: */1 * * * * $0 <program_name> [/path/to/output.log]"
        exit 1
    fi
fi

prog_name=$1
if [ -z "$prog_name" ]; then
    echo "Usage: $0 <program_name>"
    exit 1
fi

output_file=${2:-/tmp/memory/$prog_name.log}
mkdir -p $(dirname $output_file)

prog_mem=$(pidstat  -r -u -h -C $prog_name |awk 'NR==4{print $12}')
time=$(date "+%Y-%m-%d %H:%M:%S")
echo $time"\tmemory(Byte)\t"$prog_mem >>$output_file