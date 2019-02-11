#!/bin/bash

set -e

go build -o bin/routeprint tools/route_print.go
scp bin/routeprint root@192.168.98.2:/root/

ssh root@192.168.98.2 "./routeprint lo"