#!/bin/bash
docker image prune -f
docker image ls | grep openelb | grep -v infra | awk '{print $1":"$2}' | xargs docker rmi