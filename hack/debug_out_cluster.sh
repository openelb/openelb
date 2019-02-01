#!/bin/bash

set -e
master=root@192.168.98.2
OOS=linux GOARCH=amd64 go build -a -o bin/manager cmd/manager/main.go
scp bin/manager $master:/root/
scp config/bgp/config.toml $master:/root/
ssh $master "./manager -f config.toml"