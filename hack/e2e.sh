#!/bin/bash

set -e

function cleanup(){
    result=$?
    set +e
    echo "Cleaning"
    kubectl delete ns $TEST_NS
    #docker image rm $IMG
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
    -l|--lib)
    LIBPATH="$2"
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
    IMG=magicsong/porter-agent:$tag
    ./hack/deploy.sh $IMG agent
    echo "Building agent Done"
fi

if [ "$(uname)" == "Darwin" ]; then
    sed -i '' -e  's/namespace: .*/namespace: '"${TEST_NS}"'/' ./config/default/kustomization.yaml
else
    sed -i  -e  's/namespace: .*/namespace: '"${TEST_NS}"'/' ./config/default/kustomization.yaml
fi

echo "Building yamls"
kustomize build config/default -o deploy/porter.yaml
echo "Building yamls Done"
kubectl create ns  $TEST_NS
kustomize build config/default -o $dest
kubectl apply -f $dest
###./hack/certs.sh --service webhook-server-service --namespace $TEST_NS --secret webhook-server-secret

export TEST_NS
export BIRD_IP=192.168.98.5
ginkgo -v ./test/e2e/