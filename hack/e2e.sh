#!/bin/bash

set -e

kubectl cluster-info
function cleanup(){
    result=$?
    set +e
    echo "Cleaning Namespace"
    kubectl delete ns $TEST_NS > /dev/null
    if [ $SKIP_BUILD == "no" ]; then
        docker image rm $IMG
    fi
    exit $result
}


dest="./deploy/porter.yaml"
tag=`git rev-parse --short HEAD`
IMG=kubespheredev/porter:$tag
TEST_NS=porter-test-$tag
SKIP_BUILD=no

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

trap cleanup EXIT SIGINT SIGQUIT

if [ $SKIP_BUILD != "yes" ];then
    echo "Building manager"
    ./hack/deploy.sh $IMG manager
    echo "Building manager Done"
    echo "Building agent"
    IMG=kubespheredev/porter-agent:$tag
    ./hack/deploy.sh $IMG agent
    echo "Building agent Done"
fi

if [ "$(uname)" == "Darwin" ]; then
    sed -i '' -e  's/namespace: .*/namespace: '"${TEST_NS}"'/' ./config/default/kustomization.yaml
else
    sed -i  -e  's/namespace: .*/namespace: '"${TEST_NS}"'/' ./config/default/kustomization.yaml
fi

kubectl create ns  $TEST_NS
kubectl apply -f ./config/crds/
###./hack/certs.sh --service webhook-server-service --namespace $TEST_NS --secret webhook-server-secret
export TEST_NS

echo "Current Namespace is $TEST_NS'"
ginkgo -v ./test/e2e/