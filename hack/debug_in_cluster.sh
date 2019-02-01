#!/bin/bash
set -e
tag=`git rev-parse --short HEAD`
IMG=dockerhub.qingcloud.com/kubesphere/porter:$tag

./hack/deploy.sh $IMG

echo "deploying for testing"
kubectl apply -f deploy/release.yaml
kubectl delete pod controller-manager-0 -n porter-system
echo "Done! Let's roll"