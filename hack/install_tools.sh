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
echo "Tools install done"
