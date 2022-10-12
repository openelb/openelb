
# Image URL to use all building/pushing image targets
BRANCH ?= release
RELEASE_TAG = $(shell cat VERSION)
DOCKER_USERNAME ?= kubesphere
IMG_MANAGER ?= $(DOCKER_USERNAME)/openelb:$(RELEASE_TAG)
IMG_AGENT ?= $(DOCKER_USERNAME)/openelb-agent:$(RELEASE_TAG)
IMG_PROXY ?= $(DOCKER_USERNAME)/openelb-proxy:$(RELEASE_TAG)
IMG_FORWARD ?= $(DOCKER_USERNAME)/openelb-forward:$(RELEASE_TAG)

CRD_OPTIONS ?= "crd:trivialVersions=true"

ifeq (,$(shell git status --porcelain 2>/dev/null))
GIT_TREE_STATE="clean"
else
GIT_TREE_STATE="dirty"
endif
GIT_COMMIT = $(shell git rev-parse HEAD)
GIT_REPO = $(shell git config --get remote.origin.url)
DATE = $(shell date +"%Y-%m-%d_%H:%M:%S")
LDFLAGS= " \
	-X 'github.com/openelb/openelb/pkg/version.gitVersion=$(RELEASE_TAG)' \
	-X 'github.com/openelb/openelb/pkg/version.gitCommit=$(GIT_COMMIT)' \
	-X 'github.com/openelb/openelb/pkg/version.gitTreeState=$(GIT_TREE_STATE)' \
	-X 'github.com/openelb/openelb/pkg/version.buildDate=$(DATE)' "

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run go fmt against code
fmt:
	go fmt ./pkg/... ./cmd/...   ./api/... ./pkg/controllers/...

# Run go vet against code
vet:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go vet ./pkg/... ./cmd/...  ./pkg/controllers/...

# Run tests
test: fmt vet
	KUBEBUILDER_ASSETS="$(shell $(GOBIN)/setup-envtest use -p path 1.19.x)" go test -v  ./api/... ./pkg/controllers/... ./pkg/...  -coverprofile cover.out

# Build manager binary
manager: fmt vet
	#CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o bin/manager github.com/openelb/openelb/cmd/manager
	CGO_ENABLED=0 go build  -o bin/manager -ldflags ${LDFLAGS} github.com/openelb/openelb/cmd/manager


deploy: generate
ifeq ($(uname), Darwin)
	sed -i '' -e 's@image: .*@image: '"${IMG_AGENT}"'@' ./config/${BRANCH}/agent_image_patch.yaml
	sed -i '' -e 's@image: .*@image: '"${IMG_MANAGER}"'@' ./config/${BRANCH}/manager_image_patch.yaml
	sed -i '' -e 's@NodeProxyDefaultForwardImage      string = \".*\"@NodeProxyDefaultForwardImage      string = \"'"${IMG_FORWARD}"'\"@' ./pkg/constant/constant.go
	sed -i '' -e 's@NodeProxyDefaultProxyImage        string = \".*\"@NodeProxyDefaultProxyImage        string = \"'"${IMG_PROXY}"'\"@' ./pkg/constant/constant.go
else
	sed -i -e 's@image: .*@image: '"${IMG_AGENT}"'@' ./config/${BRANCH}/agent_image_patch.yaml
	sed -i -e 's@image: .*@image: '"${IMG_MANAGER}"'@' ./config/${BRANCH}/manager_image_patch.yaml
	sed -i -e 's@NodeProxyDefaultForwardImage      string = \".*\"@NodeProxyDefaultForwardImage      string = \"'"${IMG_FORWARD}"'\"@' ./pkg/constant/constant.go
	sed -i -e 's@NodeProxyDefaultProxyImage        string = \".*\"@NodeProxyDefaultProxyImage        string = \"'"${IMG_PROXY}"'\"@' ./pkg/constant/constant.go
endif
	kustomize build config/${BRANCH} -o deploy/openelb.yaml
	@echo "Done, the yaml is in deploy folder named 'openelb.yaml'"

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths=./api/...
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=openelb-manager-role webhook paths="./api/..." paths="./pkg/controllers/..." output:crd:artifacts:config=config/crd/bases

controller-gen:
ifeq (, $(shell which controller-gen))
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.0
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

clean-up:
	./hack/cleanup.sh

release: deploy
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/manager-linux-amd64 github.com/openelb/openelb/cmd/manager
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/agent-linux-amd64 github.com/openelb/openelb/cmd/agent
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build  -o bin/gobgp-linux-amd64 github.com/osrg/gobgp/cmd/gobgp
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bin/manager-linux-arm64 github.com/openelb/openelb/cmd/manager
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bin/agent-linux-arm64 github.com/openelb/openelb/cmd/agent
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build  -o bin/gobgp-linux-arm64 github.com/osrg/gobgp/cmd/gobgp
	DOCKER_CLI_EXPERIMENTAL=enabled docker buildx build --platform linux/amd64,linux/arm64 -t ${IMG_AGENT} -f ./cmd/agent/Dockerfile .  --push
	DOCKER_CLI_EXPERIMENTAL=enabled docker buildx build --platform linux/amd64,linux/arm64 -t ${IMG_MANAGER} -f ./cmd/manager/Dockerfile .  --push
	DOCKER_CLI_EXPERIMENTAL=enabled docker buildx build --platform linux/amd64,linux/arm64 -t ${IMG_PROXY} -f ./images/proxy/Dockerfile . --push
	DOCKER_CLI_EXPERIMENTAL=enabled docker buildx build --platform linux/amd64,linux/arm64 -t ${IMG_FORWARD} -f ./images/forward/Dockerfile . --push

install-tools:
	echo "install kubebuilder/kustomize etc."
	chmod +x ./hack/*.sh
	./hack/install_tools.sh
