#!/bin/bash
set -e
tag=`git rev-parse --short HEAD`
IMG=magicsong/porter:$tag
TEST_NS=porter-system

echo "Building manager"
./hack/deploy.sh $IMG manager
echo "Building manager Done"
echo "Building agent"
IMG=magicsong/porter-agent:$tag
./hack/deploy.sh $IMG agent
echo "Building agent Done"
if [ "$(uname)" == "Darwin" ]; then
    sed -i '' -e  's/namespace: .*/namespace: '"${TEST_NS}"'/' ./config/default/kustomization.yaml
else
    sed -i  -e  's/namespace: .*/namespace: '"${TEST_NS}"'/' ./config/default/kustomization.yaml
fi
echo "Building yamls"
kustomize build config/default -o deploy/porter.yaml
echo "Building yamls Done"

echo "deploying for testing"
kubectl apply -f deploy/porter.yaml
echo "Done! Let's roll"