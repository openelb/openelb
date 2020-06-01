#!/bin/bash

set -e

echo "install kubebuilder"
os=$(go env GOOS)
arch=$(go env GOARCH)
# download kubebuilder and extract it to tmp
curl -sL https://go.kubebuilder.io/dl/2.0.0-rc.0/${os}/${arch} | tar -xz -C /tmp/
sudo mv /tmp/kubebuilder_2.0.0-rc.0_${os}_${arch} /usr/local/kubebuilder


echo "install kustomize"
if [ ! -f /usr/local/bin/kustomize ];then
  cd /usr/local/bin
  curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"  | bash
fi

export PATH=$PATH:/usr/local/kubebuilder/bin:/usr/local/bin/
echo "Tools install done"