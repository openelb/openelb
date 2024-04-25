#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

OPENELB_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
source "${OPENELB_ROOT}/hack/lib/init.sh"

VERBOSE=${VERBOSE:-"0"}
if [[ "${VERBOSE}" == "1" ]];then
    set -x
fi

OUTPUT_DIR=bin
BUILDPATH=./${1:?"path to build"}
OUT=${OUTPUT_DIR}/${1:?"output path"}


GOBINARY=${GOBINARY:-go}
BUILD_GOOS=${GOOS:-$(go env GOOS)}
BUILD_GOARCH=${GOARCH:-$(go env GOARCH)}
LDFLAGS=$(kube::version::ldflags)

# forgoing -i (incremental build) because it will be deprecated by tool chain.
GOOS=${BUILD_GOOS} CGO_ENABLED=0 GOARCH=${BUILD_GOARCH} ${GOBINARY} build \
        -ldflags="${LDFLAGS}" \
        -o "${OUT}" \
        "${BUILDPATH}"
