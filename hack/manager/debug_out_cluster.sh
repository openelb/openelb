#!/bin/bash

set -e
master=root@192.168.98.2
GOOS=linux GOARCH=amd64 go build -ldflags "-w" -o bin/manager/manager cmd/manager/main.go
scp bin/manager/manager $master:/root/
scp config/bgp/config.toml $master:/root/
ssh $master "./manager -f config.toml"