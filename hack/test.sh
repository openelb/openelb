#!/bin/bash

set -e

sedopt="-i -e"
if [ "$(uname)" == "Darwin" ]; then
  sedopt="-i .bak -e"
fi

function cleanup() {
  result=$?
  exit $result
}
trap cleanup SIGINT SIGQUIT EXIT

# init vars
MANAGER_IMG=${MANAGER_IMG-"kubespheredev/porter"}:latest
echo "MANAGER_IMG = $MANAGER_IMG"
AGENT_IMG=${AGENT_IMG-"kubespheredev/porter-agent"}:latest
echo "AGENT_IMG = $AGENT_IMG"
SKIP_BUILD=no

#parse options
while [[ $# -gt 0 ]]; do
  key="$1"

  case $key in
  -s | --skip-build)
    SKIP_BUILD=yes
    shift # past argument
    ;;
  *) # unknown option
    POSITIONAL+=("$1") # save it in an array for later
    shift              # past argument
    ;;
  esac
done

# build porter-manager/porter-agent
if [ $SKIP_BUILD != "yes" ]; then
  echo "Building manager"
  ./hack/deploy.sh $MANAGER_IMG manager
  echo "Building manager Done"

  echo "Building agent"
  ./hack/deploy.sh $AGENT_IMG agent
  echo "Building agent Done"
fi

echo "[4] updating kustomize image patch file"
sed $sedopt 's@image: .*@image: '"${AGENT_IMG}"'@' ./config/dev/agent_image_patch.yaml
sed $sedopt 's@image: .*@image: '"${MANAGER_IMG}"'@' ./config/dev/manager_image_patch.yaml


echo "use command below to deploy porter for testing"
echo "kubectl apply -k ./config/dev/"
