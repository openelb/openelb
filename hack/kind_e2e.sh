#!/bin/bash

KIND_LOG_LEVEL="1"

if [ -n "${DEBUG}" ]; then
  set -x
  KIND_LOG_LEVEL="6"
fi

set -o errexit
set -o nounset
set -o pipefail

OPENELB_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
source "${OPENELB_ROOT}"/hack/lib/init.sh

cleanup() {
  kind delete cluster \
    --verbosity="${KIND_LOG_LEVEL}" \
    --name "${KIND_CLUSTER_NAME}"
}

trap cleanup EXIT

export KIND_CLUSTER_NAME=${KIND_CLUSTER_NAME:-openelb-e2e}

if ! command -v kind --version &> /dev/null; then
  echo "kind is not installed. Use the package manager or visit the official site https://kind.sigs.k8s.io/"
  exit 1
fi

echo "Creating Kubernetes cluster with kind"

export K8S_VERSION=${K8S_VERSION:-v1.24.7}

kind create cluster \
  --verbosity="${KIND_LOG_LEVEL}" \
  --name "${KIND_CLUSTER_NAME}" \
  --config "${OPENELB_ROOT}"/test/e2e/kind.yaml \
  --retain \
  --image kindest/node:"${K8S_VERSION}"

echo "Kubernetes cluster:"
kubectl get nodes -o wide

echo ""
echo "Deploy OpenELB"
"${OPENELB_ROOT}"/hack/deploy_openelb.sh



# Install ginkgo
if ! command -v ginkgo version &> /dev/null; then
  echo ""
  echo "Install ginkgo"
  GO111MODULE=on go install github.com/onsi/ginkgo/v2/ginkgo
fi

# export E2E_BUILD_FLAGS=${E2E_BUILD_FLAGS:-'-ldflags "-w -s"'}
echo ""
echo "Run e2e test"
	# ginkgo build ${E2E_BUILD_FLAGS} ./test/e2e/
	# ginkgo --randomize-all -v --timeout=30m --focus="LB:OpenELB" ./test/e2e/e2e.test
    ginkgo --randomize-all -v --timeout=30m --focus="LB:OpenELB" ./test/e2e/
