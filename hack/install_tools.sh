#!/bin/bash

set -e

echo "install kubebuilder"

if [[ ! -f /usr/local/bin/kubebuilder ]]; then
    cd /usr/local/bin
    curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/$(go env GOOS)/$(go env GOARCH)
    chmod +x kubebuilder
fi

echo "install kustomize"
if [[ ! -f /usr/local/bin/kustomize ]]; then
  cd /usr/local/bin
  curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash
fi

export PATH=$PATH:/usr/local/kubebuilder/bin:/usr/local/bin/

echo "setup envtest"

# TODO: update to latest
go install sigs.k8s.io/controller-runtime/tools/setup-envtest@v0.0.0-20230926180527-c93e2abcb28e

echo "Tools install done"
