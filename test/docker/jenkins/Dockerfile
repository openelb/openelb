FROM golang:1.11

RUN apt-get update && apt-get install -y apt-transport-https jq openssl libltdl7 \
    && curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add - \
    && echo "deb https://apt.kubernetes.io/ kubernetes-xenial main" | tee -a /etc/apt/sources.list.d/kubernetes.list \
    && apt-get update \
    && apt-get install -y kubectl \
    && curl -O -L https://github.com/kubernetes-sigs/kustomize/releases/download/v1.0.11/kustomize_1.0.11_linux_amd64 \
    && chmod +x kustomize_1.0.11_linux_amd64 \
    && mv kustomize_1.0.11_linux_amd64 /usr/bin/kustomize \
    && go get github.com/onsi/ginkgo/ginkgo \
    && curl -SL https://github.com/osrg/gobgp/releases/download/v2.3.0/gobgp_2.3.0_linux_amd64.tar.gz | tar xvz -C /usr/local/bin/  
