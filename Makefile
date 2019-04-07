
# Image URL to use all building/pushing image targets
IMG_MANAGER ?= kubespheredev/porter:v0.1
IMG_AGENT ?= kubespheredev/porter-agent:v0.1
NAMESPACE ?= porter-system

all: test manager

# Run tests
test: fmt vet
	go test -v ./pkg/controller/... ./pkg/apis/... ./pkg/test/  ./test/e2eutil/ -coverprofile cover.out

# Build manager binary
manager: fmt vet
	go build -o bin/manager github.com/kubesphere/porter/cmd/manager

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet
	go run ./cmd/manager/main.go

# Install CRDs into a cluster
install: manifests
	kubectl apply -f config/crds

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	kubectl apply -f config/crds
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests:
	go run vendor/sigs.k8s.io/controller-tools/cmd/controller-gen/main.go all

# Run go fmt against code
fmt:
	go fmt ./pkg/... ./cmd/... ./test/...

# Run go vet against code
vet:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go vet ./pkg/... ./cmd/... ./test/...

# Generate code
generate:
	go run vendor/k8s.io/code-generator/cmd/deepcopy-gen/main.go -O zz_generated.deepcopy -i github.com/kubesphere/porter/pkg/apis/... -h hack/boilerplate.go.txt

debug: vet
	./hack/debug_in_cluster.sh
debug-out-of-cluster: vet
	./hack/manager/debug_out_cluster.sh

debug-log:
	kubectl logs -f -n porter-system controller-manager-0 -c manager

clean-up:
	./hack/cleanup.sh

release:
	./hack/deploy.sh ${IMG_MANAGER} manager
	./hack/deploy.sh ${IMG_AGENT} agent
	sed -i '' -e  's/namespace: .*/namespace: '"${NAMESPACE}"'/' ./config/default/kustomization.yaml
	kustomize build config/default -o deploy/porter.yaml
	@echo "Done, the yaml is in deploy folder named 'porter.yaml'"

release-with-private-registry: test
	./hack/deploy.sh ${IMG_MANAGER} manager --private
	./hack/deploy.sh ${IMG_AGENT} agent --private
	sed -i '' -e  's/namespace: .*/namespace: '"${NAMESPACE}"'/' ./config/default/kustomization.yaml
	@echo "Building yamls"
	kustomize build config/overlays/private_registry -o deploy/porter.yaml
	@echo "Done, the yaml is in deploy folder named 'porter.yaml'"

install-travis:
	chmod +x ./hack/*.sh
	./hack/install_tools.sh

e2e-test: vet
	./hack/e2e.sh
e2e-nobuild:
	./hack/e2e.sh --skip-build

docker-ut:
	docker run --rm -v "${PWD}":/usr/src/github.com/kubesphere/porter -w /usr/src/github.com/kubesphere/porter golang:1.11-alpine  go test -v ./pkg/nettool/