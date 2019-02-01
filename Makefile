
# Image URL to use all building/pushing image targets
IMG ?= kubespheredev/porter:0.0.1

all: test manager

# Run tests
test: generate fmt vet manifests
	go test ./pkg/... ./cmd/... -coverprofile cover.out

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
	go fmt ./pkg/... ./cmd/...

# Run go vet against code
vet:
	go vet ./pkg/... ./cmd/...

# Generate code
generate:
	go generate ./pkg/... ./cmd/...

# Push the docker image
docker-push:
	docker push ${IMG}

binary:
	go build -o bin/manager ./cmd/manager/main.go

debug:
	./hack/debug_in_cluster.sh
debug-out-of-cluster:
	./hack/debug_out_cluster.sh

debug-log:
	kubectl logs -f -n porter-system controller-manager-0 -c manager

clean-up:
	docker rmi $(docker images | grep "kubesphere/porter" | awk '{print $3}') 

release: fmt vet
	./hack/deploy.sh ${IMG}
	@echo "Done, the yaml is in deploy folder named 'release.yaml'"
