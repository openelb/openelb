# How to Build Porter Project

### Prerequisites

1. Go 1.12
2. Docker
3. Kustomize/Kubebuilder (Install via `./hack/install_tools.sh`)
4. controller-gen (Install via `go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.0`)

### Steps

1. Execute `git clone https://github.com/kubesphere/porter.git`, then enter into the folder.
2. Execute `make generate` to generate crds, webhook, etc.
3. Modify the IMG name in the Makefile, then run `make release`, and the final yaml file is under `/deploy`.
4. Execute `kubectl apply -f deploy/release.yaml` to deploy porter as a plugin.
