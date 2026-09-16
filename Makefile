REGISTRY ?= ghcr.io
USERNAME ?= sergelogvinov
OCIREPO ?= $(REGISTRY)/$(USERNAME)
HELMREPO ?= $(REGISTRY)/$(USERNAME)/charts
PLATFORM ?= linux/arm64,linux/amd64
PUSH ?= false

SHA ?= $(shell git describe --match=none --always --abbrev=7 --dirty)
TAG ?= $(shell git describe --tag --always --match v[0-9]\*)
GO_LDFLAGS := -ldflags "-w -s -X main.version=$(TAG) -X main.commit=$(SHA)"

OS ?= $(shell go env GOOS)
ARCH ?= $(shell go env GOARCH)
ARCHS ?= amd64 arm64

COSING_ARGS ?=
BUNDLE_DIR ?= bundle

############

MCPB_OS ?= $(OS)
ifeq ($(OS),windows)
MCPB_OS = win32
endif

############

# Help Menu

define HELP_MENU_HEADER
# Getting Started

To build this project, you must have the following installed:

- git
- make
- golang 1.26+
- golangci-lint

endef

export HELP_MENU_HEADER

help: ## This help menu
	@echo "$$HELP_MENU_HEADER"
	@grep -E '^[a-zA-Z0-9%_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

############
#
# Build Abstractions
#

build-all-archs:
	@for arch in $(ARCHS); do $(MAKE) ARCH=$${arch} build ; done

.PHONY: clean
clean: ## Clean
	rm -rf bin/ dist/ .cache/ .gocache/
	rm -rf proxmox-mcp_*.mcpb

.PHONY: tools
tools:
	go install github.com/google/go-licenses@latest

.PHONY: build ## Build
build:
	CGO_ENABLED=0 GOOS=$(OS) GOARCH=$(ARCH) go build $(GO_LDFLAGS) \
		-o bin/proxmox-mcp-$(ARCH) ./cmd/proxmox-mcp

.PHONY: run
run: ## Run
	go run $(GO_LDFLAGS) ./cmd/proxmox-mcp server --port 8080 --log-level=debug --allow-destructive --extensions=all

.PHONY: run-mcp
run-mcp: ## Run mcp
	go run $(GO_LDFLAGS) ./cmd/proxmox-mcp mcp --log-level=debug --allow-destructive --extensions=all

.PHONY: lint
lint: ## Lint Code
	golangci-lint run --config .golangci.yml

.PHONY: vet
vet: ## Vet Code
	go vet ./...

.PHONY: unit
unit: ## Unit Tests
	go test -tags=unit $(shell go list ./...) $(TESTARGS)

.PHONY: test
test: lint unit ## Run all tests

.PHONY: install
install: build
	cp ./bin/proxmox-mcp-$(ARCH) ~/go/bin/proxmox-mcp

.PHONY: licenses
licenses:
	go-licenses check ./... --disallowed_types=forbidden,restricted,unknown

.PHONY: conformance
conformance: ## Conformance
	docker run --rm -it -v $(PWD):/src -w /src ghcr.io/siderolabs/conform:v0.1.0-alpha.31 enforce

############

.PHONY: helm-unit
helm-unit: ## Helm Unit Tests
	@helm lint charts/proxmox-mcp
	@helm template -f charts/proxmox-mcp/ci/values.yaml proxmox-mcp charts/proxmox-mcp >/dev/null

.PHONY: helm-login
helm-login: ## Helm Login
	@echo "${HELM_TOKEN}" | helm registry login $(REGISTRY) --username $(USERNAME) --password-stdin

.PHONY: helm-release
helm-release: ## Helm Release
	@rm -rf dist/
	@helm package charts/proxmox-mcp -d dist
	@helm push dist/proxmox-mcp-*.tgz oci://$(HELMREPO) 2>&1 | tee dist/.digest
	@cosign sign --yes $(COSING_ARGS) $(HELMREPO)/proxmox-mcp@$$(cat dist/.digest | awk -F "[, ]+" '/Digest/{print $$NF}')

############

.PHONY: docs
docs:
	helm version
	yq -i '.appVersion = "$(TAG)"' charts/proxmox-mcp/Chart.yaml
	helm template -n mcps proxmox-mcp \
		-f charts/proxmox-mcp/values.edge.yaml \
		charts/proxmox-mcp > docs/deploy/proxmox-mcp.yml
	helm template -n mcps proxmox-mcp \
		--set-string image.tag=$(TAG) \
		--set createNamespace=true \
		charts/proxmox-mcp > docs/deploy/proxmox-mcp-release.yml
	helm-docs --sort-values-order=file charts/proxmox-mcp

.PHONY: release-mcpb
release-mcpb: ## Release MCPB bundle
	@rm -rf $(BUNDLE_DIR)/server
	@mkdir -p $(BUNDLE_DIR)/server
	cp -r bin/default_$(OS)_$(ARCH)_*/proxmox-mcp* $(BUNDLE_DIR)/server/
	jq --arg v "$(TAG)" --arg p "$(MCPB_OS)" '.version = $$v | .compatibility.platforms = [$$p]' manifest.json > $(BUNDLE_DIR)/manifest.json
	npx @anthropic-ai/mcpb pack $(BUNDLE_DIR) proxmox-mcp_$(OS)_$(ARCH).mcpb

############
#
# Docker Abstractions
#

.PHONY: docker-init
docker-init:
	docker run --rm --privileged multiarch/qemu-user-static:register --reset

	docker context create multiarch ||:
	docker buildx create --name multiarch --driver docker-container --use ||:
	docker context use multiarch
	docker buildx inspect --bootstrap multiarch

image-%:
	docker buildx build $(BUILD_ARGS) \
		--build-arg TAG=$(TAG) \
		--build-arg SHA=$(SHA) \
		-t $(OCIREPO)/$*:$(TAG) \
		--target $* \
		-f Dockerfile .

.PHONY: images-checks
images-checks: images
	trivy image --exit-code 1 --ignore-unfixed --severity HIGH,CRITICAL --no-progress $(OCIREPO)/proxmox-mcp:$(TAG)

.PHONY: images-cosign
images-cosign:
	@cosign sign --yes $(COSING_ARGS) --recursive $(OCIREPO)/proxmox-mcp:$(TAG)

.PHONY: images
images: image-proxmox-mcp ## Build images
