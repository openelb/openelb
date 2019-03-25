#!/bin/bash

set -e

ORGANIZATION=magicsong
IMAGE=porter
TAG=""

while [[ $# -gt 0 ]]
do
key="$1"

case $key in
    -t|--tag)
    TAG=$2
    shift # past argument
    shift
    ;;
    -i|--image)
    IMAGE=$2
    shift # past argument
    shift # past value
    ;;
    -o|--organization)
    ORGANIZATION="$2"
    shift # past argument
    shift # past value
    ;;
    -u|--user)
    USERNAME=$2
    shift # past argument
    shift # past value
    ;;
    -p|--password)
    PASSWORD=$2
    shift # past argument
    shift # past value
    ;;
    *)    # unknown option
    POSITIONAL+=("$1") # save it in an array for later
    shift # past argument
    ;;
esac
done


if [ $TAG == "" ]; then
    echo "TAG is required"
    exit 1
fi

docker image rm $ORGANIZATION/$IMAGE:$TAG
echo "Image deleted"