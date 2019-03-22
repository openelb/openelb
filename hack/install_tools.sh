#!/bin/bash

set -e

echo "install kubebuilder"

version=1.0.8 # latest stable version
arch=amd64

# download the release
curl -L -O "https://github.com/kubernetes-sigs/kubebuilder/releases/download/v${version}/kubebuilder_${version}_linux_${arch}.tar.gz"

# extract the archive
tar -zxvf kubebuilder_${version}_linux_${arch}.tar.gz
sudo mv kubebuilder_${version}_linux_${arch} /usr/local/kubebuilder

# update your PATH to include /usr/local/kubebuilder/bin
export PATH=$PATH:/usr/local/kubebuilder/bin

# echo "install kustomize"

# wget https://github.com/kubernetes-sigs/kustomize/releases/download/v1.0.11/kustomize_1.0.11_linux_amd64 
# chmod u+x kustomize_1.0.11_linux_amd64
# mv kustomize_1.0.11_linux_amd64 /home/travis/bin/kustomize

echo "Tools install done"