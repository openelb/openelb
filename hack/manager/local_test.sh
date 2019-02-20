#!/bin/bash

set -e

go build -ldflags "-w" -o bin/manager/manager cmd/manager/main.go
./bin/manager/manager -f config/bgp/config.toml