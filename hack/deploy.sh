#!/bin/bash
set -e

IMG=$1
binary=$2

echo "[1] Building binary for $binary"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-w" -a -o  bin/${binary}/$binary cmd/${binary}/main.go

echo "[2] Binary build done, Build docker image $IMG of $binary"
docker build -f deploy/${binary}/Dockerfile -t ${IMG} bin/$binary/

echo "[3] Docker image build done, try to push to registry"
docker push $IMG

echo "[4] updating kustomize image patch file for $binary resource"
sed -i'' -e 's@image: .*@image: '"${IMG}"'@' ./config/default/${binary}_image_patch.yaml

if [ "$3" == "--private" ]; then
    echo "add pull registry to manifest"
    dockerconfig=`cat ~/.docker/config.json | base64 -w 0`
    sed -i -e 's/dockerconfigjson:.*/dockerconfigjson: '"$dockerconfig"'/' ./config/overlays/private_registry/manager_secret.yaml
    echo "Building yamls"
    kustomize build config/overlays/private_registry -o deploy/release.yaml
    exit 0   
fi


