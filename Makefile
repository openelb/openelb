
CRD_OPTIONS ?= "crd:crdVersions=v1,allowDangerousTypes=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

.PHONY: all
all: test controller speaker server

# Run go fmt against code
fmt:
	go fmt ./pkg/... ./cmd/... ./api/... ./pkg/controllers/...

# Run go vet against code
vet:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go vet ./pkg/... ./cmd/...

# Run tests
test: fmt vet
	KUBEBUILDER_ASSETS="$(shell $(GOBIN)/setup-envtest use -p path 1.19.x)" go test -v  ./api/... ./pkg/controllers/... ./pkg/...  -coverprofile cover.out

.PHONY: binary
# Build all of binary
binary: | controller speaker server; $(info $(M)...Build all of binary.) @ ## Build all of binary.

# Build controller binary
controller: ; $(info $(M)...Begin to build openelb-controller binary.)  @ ## Build controller.
	hack/gobuild.sh cmd/controller;

# Build speaker binary
speaker: ; $(info $(M)...Begin to build openelb-speaker binary.)  @ ## Build speaker.
	hack/gobuild.sh cmd/speaker;

# Build apiserver binary
apiserver: ; $(info $(M)...Begin to build openelb-apiserver binary.)  @ ## Build apiserver.
	hack/gobuild.sh cmd/apiserver;


# build in docker
container: ;$(info $(M)...Begin to build the docker image.)  @ ## Build the docker image.
	DRY_RUN=true hack/docker_build.sh

container-push: ;$(info $(M)...Begin to build and push.)  @ ## Build and Push.
	hack/docker_build.sh

container-cross: ; $(info $(M)...Begin to build container images for multiple platforms.)  @ ## Build container images for multiple platforms. Currently, only linux/amd64,linux/arm64 are supported.
	DRY_RUN=true hack/docker_build_multiarch.sh

container-cross-push: ; $(info $(M)...Begin to build and push.)  @ ## Build and Push.
	hack/docker_build_multiarch.sh

deploy: generate
	hack/generate_manifests.sh

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths=./api/...
	$(CONTROLLER_GEN) $(CRD_OPTIONS) paths="./api/..." output:artifacts:config=config/crd/bases
	$(CONTROLLER_GEN) webhook paths="./api/..." paths="./pkg/controllers/..." output:artifacts:config=config/webhook
	$(CONTROLLER_GEN) rbac:roleName=openelb-manager-role paths="./api/..." paths="./pkg/controllers/..." output:artifacts:config=config/rbac

controller-gen:
ifeq (, $(shell which controller-gen))
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.12.1
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

clean-up:
	./hack/cleanup.sh

install-tools:
	echo "install kubebuilder/kustomize etc."
	chmod +x ./hack/*.sh
	./hack/install_tools.sh
