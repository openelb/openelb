#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail


# Use kubesphere and latest tag as default image
OPENELB_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
REPO=${REPO:-kubesphere}
TAG=${TAG:-latest}
TIMEOUT=${TIMEOUT:-300}

function wait_for_openelb_ready() {
    echo ""
    echo "Waiting for deploy/openelb-controller ready"
    kubectl -n openelb-system rollout status deploy/openelb-controller --timeout=${TIMEOUT}s

    echo "Waiting for daemonset/openelb-speaker ready" 
    kubectl -n openelb-system rollout status ds/openelb-speaker --timeout=${TIMEOUT}s
}

if [[ "${BUILD_IMAGE:-}" == "y" ]]; then
    echo ""
    echo "Build OpenELB images"
    DRY_RUN=true REPO=${REPO} TAG=${TAG} "${OPENELB_ROOT}"/hack/docker_build.sh
fi

if [[ "${KIND_LOAD_IMAGE:-}" == "y" ]]; then
    echo ""
    echo "Load OpenELB images into kind"
    # load the built docker image into kind before deploying.
    kind load docker-image "$REPO/openelb-controller:$TAG" --name="${KIND_CLUSTER_NAME:-kind}"
    kind load docker-image "$REPO/openelb-speaker:$TAG" --name="${KIND_CLUSTER_NAME:-kind}"
fi

echo "Apply OpenELB"
kubectl apply -f ${OPENELB_ROOT}/deploy/openelb.yaml
kubectl set image -n openelb-system deployment/openelb-controller *="$REPO/openelb-controller:$TAG"
kubectl set image -n openelb-system daemonset/openelb-speaker *="$REPO/openelb-speaker:$TAG"

wait_for_openelb_ready
