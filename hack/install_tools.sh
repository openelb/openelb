#!/bin/bash

set -e

echo "install kubebuilder"

os=$(go env GOOS)
arch=$(go env GOARCH)

# download kubebuilder and extract it to tmp
curl -sL https://go.kubebuilder.io/dl/2.0.0-rc.0/${os}/${arch} | tar -xz -C /tmp/

sudo mv /tmp/kubebuilder_2.0.0-rc.0_${os}_${arch} /usr/local/kubebuilder
export PATH=$PATH:/usr/local/kubebuilder/bin


# echo "install kustomize"

# wget https://github.com/kubernetes-sigs/kustomize/releases/download/v3.1.0/kustomize_3.1.0_linux_amd64
# chmod u+x kustomize_3.1.0_linux_amd64
# mv kustomize_3.1.0_linux_amd64 /home/travis/bin/kustomize

echo "Tools install done"