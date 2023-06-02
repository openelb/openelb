#!/usr/bin/env bash

set -ex
set -o pipefail

OPENELB_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
source "${OPENELB_ROOT}/hack/lib/init.sh"

# push to kubesphere with default latest tag
TAG=${TAG:-$(cat VERSION)}
REPO=${REPO:-kubesphere}
PUSH=${PUSH:-}

# support other container tools. e.g. podman
CONTAINER_CLI=${CONTAINER_CLI:-docker}
CONTAINER_BUILDER=${CONTAINER_BUILDER:-"buildx build"}

# If set, just building, no pushing
if [[ -z "${DRY_RUN:-}" ]]; then
  PUSH="--push"
fi

# supported platforms
PLATFORMS=linux/amd64,linux/arm64


# shellcheck disable=SC2086 # inteneded splitting of CONTAINER_BUILDER
${CONTAINER_CLI} ${CONTAINER_BUILDER} \
  --platform ${PLATFORMS} \
  ${PUSH} \
  -f build/controller/Dockerfile \
  -t "${REPO}"/openelb-controller:"${TAG}" .


# # shellcheck disable=SC2086 # intended splitting of CONTAINER_BUILDER
# ${CONTAINER_CLI} ${CONTAINER_BUILDER} \
#   --platform ${PLATFORMS} \
#   ${PUSH} \
#   -f build/speaker/Dockerfile \
#   -t "${REPO}"/openelb-speaker:"${TAG}" .


# shellcheck disable=SC2086 # inteneded splitting of CONTAINER_BUILDER
${CONTAINER_CLI} ${CONTAINER_BUILDER} \
  --platform ${PLATFORMS} \
  ${PUSH} \
  -f build/proxy/Dockerfile \
  -t "${REPO}"/openelb-proxy:"${TAG}" .


# shellcheck disable=SC2086 # inteneded splitting of CONTAINER_BUILDER
${CONTAINER_CLI} ${CONTAINER_BUILDER} \
  --platform ${PLATFORMS} \
  ${PUSH} \
  -f build/forward/Dockerfile \
  -t "${REPO}"/openelb-forward:"${TAG}" .
