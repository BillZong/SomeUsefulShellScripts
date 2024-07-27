#!/bin/bash

if [[ $# -eq 0 ]]; then
    echo "Usage: $0 append_shell_rc_file"
    exit 1
fi

RC_FILE=$1

if [ -f "$RC_FILE" ]; then
    v=$(grep "timecurl=" $RC_FILE)
    if [ "$v" = "" ]; then
        cat >>$RC_FILE<<EOF
# time curl with format
alias timecurl="curl -w '     time_namelookup:  %{time_namelookup}s\n        time_connect:  %{time_connect}s\n     time_appconnect:  %{time_appconnect}s\n    time_pretransfer:  %{time_pretransfer}s\n       time_redirect:  %{time_redirect}s\n  time_starttransfer:  %{time_starttransfer}s\n                     ----------\n          time_total:  %{time_total}s\n' \$@"
EOF
        source $RC_FILE
    fi
fi
