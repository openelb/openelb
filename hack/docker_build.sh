#!/usr/bin/env bash

set -ex
set -o pipefail

OPENELB_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
source "${OPENELB_ROOT}/hack/lib/init.sh"

# push to kubesphere with default latest tag
TAG=${TAG:-$(cat VERSION)}
REPO=${REPO:-kubesphere}

# If set, just building, no pushing
DRY_RUN=${DRY_RUN:-}

# support other container tools. e.g. podman
CONTAINER_CLI=${CONTAINER_CLI:-docker}
CONTAINER_BUILDER=${CONTAINER_BUILDER:-build}

# use host os and arch as default target os and arch
TARGETOS=${TARGETOS:-$(kube::util::host_os)}
TARGETARCH=${TARGETARCH:-$(kube::util::host_arch)}

${CONTAINER_CLI} "${CONTAINER_BUILDER}" \
  --build-arg TARGETARCH="${TARGETARCH}" \
  --build-arg TARGETOS="${TARGETOS}" \
  --output type=docker \
  -f build/controller/Dockerfile \
  -t "${REPO}"/openelb-controller:"${TAG}" .


${CONTAINER_CLI} "${CONTAINER_BUILDER}" \
  --build-arg "TARGETARCH=${TARGETARCH}" \
  --build-arg "TARGETOS=${TARGETOS}" \
  --output type=docker \
  -f build/speaker/Dockerfile \
  -t "${REPO}"/openelb-speaker:"${TAG}" .

# ${CONTAINER_CLI} "${CONTAINER_BUILDER}" \
#   --build-arg "TARGETARCH=${TARGETARCH}" \
#   --build-arg "TARGETOS=${TARGETOS}" \
#   --output type=docker \
#   -f build/proxy/Dockerfile \
#   -t "${REPO}"/openelb-proxy:"${TAG}" .

# ${CONTAINER_CLI} "${CONTAINER_BUILDER}" \
#   --build-arg "TARGETARCH=${TARGETARCH}" \
#   --build-arg "TARGETOS=${TARGETOS}" \
#   --output type=docker \
#   -f build/forward/Dockerfile \
#   -t "${REPO}"/openelb-forward:"${TAG}" .


if [[ -z "${DRY_RUN:-}" ]]; then
  ${CONTAINER_CLI} push "${REPO}"/openelb-controller:"${TAG}"
  ${CONTAINER_CLI} push "${REPO}"/openelb-speaker:"${TAG}"
#   ${CONTAINER_CLI} push "${REPO}"/openelb-proxy:"${TAG}"
#   ${CONTAINER_CLI} push "${REPO}"/openelb-forward:"${TAG}"
fi
