#!/bin/bash

set -e

sedopt="-i -e"
if [ "$(uname)" == "Darwin" ]; then
    sedopt="-i .bak -e"
fi

function cleanup(){
    result=$?
    set +e
    if [ $MODE == "test" ]; then
        echo "Cleaning Namespace"
        kubectl delete ns $TEST_NS
    fi
    exit $result
}


dest="./deploy/porter.yaml"
tag=`git rev-parse --short HEAD`
MANAGER_IMG=kubespheredev/porter:$tag
AGENT_IMG=kubespheredev/porter-agent:$tag
TEST_NS=porter-test-$tag
SKIP_BUILD=no
MODE=test
##cleanning before running
kubectl get ns $TEST_NS 2>&1 | grep "not found" || kubectl delete ns $TEST_NS

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
    -m|--mode)
    MODE="$2"
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

trap cleanup SIGINT SIGQUIT EXIT

if [ $SKIP_BUILD != "yes" ]; then
    if [ x$SKIP_BUILD_MANAGER != "xtrue" ]; then
        echo "Building manager"
        ./hack/deploy.sh $MANAGER_IMG manager
        echo "Building manager Done"
    fi
    if [ x$SKIP_BUILD_AGENT != "xtrue" ]; then
        echo "Building agent"
        ./hack/deploy.sh $AGENT_IMG agent
        echo "Building agent Done"
    fi
fi

echo "[4] updating kustomize image patch file"
sed $sedopt 's@image: .*@image: '"${AGENT_IMG}"'@' ./config/dev/agent_image_patch.yaml
sed $sedopt 's@image: .*@image: '"${MANAGER_IMG}"'@' ./config/dev/manager_image_patch.yaml
sed $sedopt  's/namespace: .*/namespace: '"${TEST_NS}"'/' ./config/dev/kustomization.yaml

kubectl create ns  $TEST_NS --dry-run -oyaml | kubectl apply -f -
YAML_PATH=/tmp/porter.yaml
kustomize build config/dev -o $YAML_PATH

echo "Current Namespace is $TEST_NS'"
if [ $MODE == "debug" ] ; then
    echo "deploying for testing"
    kubectl apply -f $YAML_PATH
    kubectl create configmap bgp-cfg --dry-run -oyaml --from-file=./config/bgp/config.toml -n $TEST_NS | kubectl apply -f -
    echo "Done! Let's roll"
else
###./hack/certs.sh --service webhook-server-service --namespace $TEST_NS --secret webhook-server-secret
    kubectl apply -k config/crd
    export TEST_NS
    export YAML_PATH
    export MASTER_IP=192.168.98.2
    export ROUTER_IP=192.168.98.8
    ginkgo -v ./test/e2e/
fi