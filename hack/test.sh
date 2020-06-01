#!/bin/bash

set -e

sedopt="-i -e"
if [ "$(uname)" == "Darwin" ]; then
    sedopt="-i .bak -e"
fi

function cleanup(){
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
TEST_NS="porter-testns"
MODE=test

#parse options
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

echo "Current Namespace is $TEST_NS'"
if [ $MODE == "debug" ] ; then
    echo "deploying for testing"
    kubectl apply -k ./config/dev/
    kubectl create configmap bgp-cfg --dry-run -oyaml --from-file=./config/bgp/config.toml -n $TEST_NS | kubectl apply -f -
    echo "Done! Let's roll"
else
###./hack/certs.sh --service webhook-server-service --namespace $TEST_NS --secret webhook-server-secret
    #User must provide MASTER_IP & ROUTER_IP
    if [ $MASTER_IP ];then
	    echo "MASTER_IP = $MASTER_IP"
    else
	    echo "MASTER_IP IS NOT EXISTS"
	    exit 1
    fi
    if [ $ROUTER_IP ];then
	    echo "ROUTER_IP = $ROUTER_IP"
    else
	    echo "ROUTER_IP IS NOT EXISTS"
	    exit 1
    fi


    export TEST_NS
    kubectl apply -k ./config/dev/

    ginkgo -v ./test/e2e/
fi