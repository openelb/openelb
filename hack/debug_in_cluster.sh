#!/bin/bash
set -e
tag=`git rev-parse --short HEAD`
MANAGER_IMG=magicsong/porter:$tag
AGENT_IMG=magicsong/porter-agent:$tag
TEST_NS=porter-system
SKIP_BUILD=no

sedopt="-i -e"
if [ "$(uname)" == "Darwin" ]; then
    sedopt="-i .bak -e"
fi 

set +e
kubectl create ns $TEST_NS
set -e

while [[ $# -gt 0 ]]
do
key="$1"

case $key in
    -s|--skip-build)
    SKIP_BUILD=yes
    shift # past argument
    ;;
    -n|--NAMESPACE)
    TEST_NS=$2
    shift # past argument
    shift # past value
    ;;
    -t|--tag)
    tag="$2"
    shift # past argument
    shift # past value
    ;;
    --default)
    DEFAULT=YES
    shift # past argument
    ;;
    *)    # unknown option
    POSITIONAL+=("$1") # save it in an array for later
    shift # past argument
    ;;
esac
done


if [ $SKIP_BUILD != "yes" ];then
    echo "Building manager"
    ./hack/deploy.sh $MANAGER_IMG manager
    echo "Building manager Done"
    echo "Building agent"
    ./hack/deploy.sh $AGENT_IMG agent
    echo "Building agent Done"
fi

sed $sedopt  's/namespace: .*/namespace: '"${TEST_NS}"'/' ./config/dev/kustomization.yaml

echo "[4] updating kustomize image patch file"
sed $sedopt 's@image: .*@image: '"${AGENT_IMG}"'@' ./config/dev/agent_image_patch.yaml
sed $sedopt 's@image: .*@image: '"${MANAGER_IMG}"'@' ./config/dev/manager_image_patch.yaml

echo "Building yamls"
kustomize build config/dev -o deploy/porter.yaml
echo "Building yamls Done"

echo "deploying for testing"
kubectl apply -f deploy/porter.yaml
echo "Done! Let's roll"