#!/bin/bash

docker image ls | grep porter | grep -v infra | awk '{print $1":"$2}' | xargs docker rmi