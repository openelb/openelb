#!/bin/bash
set -e

IMG=$1
echo "Building binary"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o bin/manager cmd/manager/main.go

echo "Binary build done, Build docker image, $IMG"
docker build -f deploy/Dockerfile -t ${IMG} bin/

echo "Docker image build done, try to push to registry"
docker push $IMG

echo "updating kustomize image patch file for manager resource"
sed -i'' -e 's@image: .*@image: '"${IMG}"'@' ./config/default/manager_image_patch.yaml

if [ "$2" == "--private" ]; then
    echo "add pull registry to manifest"
    dockerconfig=`cat ~/.docker/config.json | base64 -w 0`
    sed -i -e 's/dockerconfigjson:.*/dockerconfigjson: '"$dockerconfig"'/' ./config/overlays/private_registry/manager_secret.yaml
    echo "Building yamls"
    kustomize build config/overlays/private_registry -o deploy/release.yaml
    exit 0   
fi

echo "Building yamls"
kustomize build config/default -o deploy/release.yaml

