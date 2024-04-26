#!/usr/bin/env bash


OPENELB_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
source "${OPENELB_ROOT}/hack/lib/init.sh"

# Image URL to use all building/pushing image targets
BRANCH=${BRANCH:-release}
REPO=${REPO:-kubesphere}
TAG=${TAG:-$(cat VERSION)}
IMG_CONTROLLER=${IMG_CONTROLLER:-${REPO}/openelb-controller:${TAG}}
IMG_SPEAKER=${IMG_SPEAKER:-${REPO}/openelb-speaker:${TAG}}

if [[ $(uname) == Darwin ]]; then
    sed -i '' -e 's@image: .*@image: '"${IMG_CONTROLLER}"'@' ./config/${BRANCH}/manager_image_patch.yaml
    sed -i '' -e 's@image: .*@image: '"${IMG_SPEAKER}"'@' ./config/${BRANCH}/speaker_image_patch.yaml
else 
    sed -i -e 's@image: .*@image: '"${IMG_CONTROLLER}"'@' ./config/${BRANCH}/manager_image_patch.yaml
    sed -i -e 's@image: .*@image: '"${IMG_SPEAKER}"'@' ./config/${BRANCH}/speaker_image_patch.yaml
fi

kustomize build config/${BRANCH} -o deploy/openelb.yaml

# patch comment
images="image: registry.k8s.io/ingress-nginx/kube-webhook-certgen:v1.1.1"
comment="# If you cannot access \"registry.k8s.io/ingress-nginx/kube-webhook-certgen\", you can replace it with \"kubesphere/kube-webhook-certgen\"\n        $images"
sed -i -e "s|$images|$comment|g" deploy/openelb.yaml

echo "Done, the yaml is in deploy folder named 'openelb.yaml'"