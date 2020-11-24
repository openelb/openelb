#!/bin/bash

set -e

echo "install kubebuilder"
os=$(go env GOOS)
arch=$(go env GOARCH)
# download kubebuilder and extract it to tmp
curl -L https://go.kubebuilder.io/dl/2.3.1/${os}/${arch} | tar -xz -C /tmp/
# move to a long-term location and put it on your path
# (you'll need to set the KUBEBUILDER_ASSETS env var if you put it somewhere else)
sudo mv /tmp/kubebuilder_2.3.1_${os}_${arch} /usr/local/kubebuilder
export PATH=$PATH:/usr/local/kubebuilder/bin

echo "install kustomize"
if [ ! -f /usr/local/bin/kustomize ]; then
  cd /usr/local/bin
  curl -s "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh" | bash
fi

export PATH=$PATH:/usr/local/kubebuilder/bin:/usr/local/bin/
echo "Tools install done"
