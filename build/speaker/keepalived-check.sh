#!/bin/sh

TYPE="$1"
NAME="$2"
STATE="$3"

dir_path="/var/run/keepalived/state"
mkdir -p "$dir_path" 2>/dev/null

echo -n "${STATE}" > $dir_path/${NAME}
exit 0