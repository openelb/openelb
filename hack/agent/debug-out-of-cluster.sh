#!/bin/bash

#!/bin/bash

set -e
master=root@192.168.98.2
echo "Building binary"
OOS=linux GOARCH=amd64 go build -ldflags "-w" -o bin/agent/agent cmd/agent/main.go
echo "transport to remote"
scp bin/agent/agent $master:/root/
echo "Starting"
ssh $master "./agent " ##> agent_log.txt 2>&1 &