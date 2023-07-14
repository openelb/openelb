#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail


OPENELB_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
source "${OPENELB_ROOT}"/hack/lib/init.sh

# Use kubesphere and latest tag as default image
VIPTAG="${VIPTAG:-0.35}"
TAG="${TAG:-latest}"
REPO="${REPO:-kubesphere}"
TIMEOUT="${TIMEOUT:-300}"


function wait_for_openelb_ready() {
    echo ""
    echo "Waiting for deploy/openelb-controller ready"
    kubectl -n openelb-system rollout status deploy/openelb-controller --timeout=${TIMEOUT}s

    echo "Waiting for daemonset/openelb-speaker ready" 
    kubectl -n openelb-system rollout status ds/openelb-speaker --timeout=${TIMEOUT}s
}


# Use KIND_LOAD_IMAGE=y .hack/deploy-openelb.sh to load
# the built docker image into kind before deploying.
if [[ "${KIND_LOAD_IMAGE:-}" == "y" ]]; then
    kind load docker-image "$REPO/openelb-controller:$TAG" --name="${KIND_CLUSTER_NAME:-kind}"
    kind load docker-image "$REPO/openelb-speaker:$TAG" --name="${KIND_CLUSTER_NAME:-kind}"
    kind load docker-image "$REPO/kube-keepalived-vip:$VIPTAG" --name="${KIND_CLUSTER_NAME:-kind}"
fi

kubectl apply -f ${OPENELB_ROOT}/deploy/openelb.yaml
kubectl set image -n openelb-system deployment/openelb-controller *="$REPO/openelb-controller:$TAG"
kubectl set image -n openelb-system daemonset/openelb-speaker *="$REPO/openelb-speaker:$TAG"

wait_for_openelb_ready
