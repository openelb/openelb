#!/bin/bash

docker rmi $(docker images | grep "porter" | awk '{print $3}') 