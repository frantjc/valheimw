ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

GO ?= go
KUBECTL ?= kubectl
DOCKER ?= docker

.PHONY: apply
apply: manifests
	@$(KUBECTL) apply -f internal/stoker/stokercr/config/crd

KIND_CLUSTER_NAME ?= sindri

.PHONY: dev
dev: $(KIND)
	@if ! $(KIND) get clusters | grep -q "^$(KIND_CLUSTER_NAME)$$"; then \
		$(KIND) create cluster --config hack/kind.yml --kubeconfig dev/config --name $(KIND_CLUSTER_NAME); \
	else \
		$(KIND) get kubeconfig --name $(KIND_CLUSTER_NAME) > dev/config; \
	fi
	@$(MAKE) apply KUBECONFIG=dev/config
	@if ! $(KUBECTL) --kubeconfig dev/config get ns sindri-system; then \
		$(KUBECTL) --kubeconfig dev/config create ns sindri-system; \
	fi
	@$(DOCKER) compose create buildkitd
	@if ! $(DOCKER) network inspect sindri_default --format '{{range .Containers}}{{.Name}}{{end}}' | grep -q "$(KIND_CLUSTER_NAME)-control-plane"; then \
		$(DOCKER) network connect sindri_default $(KIND_CLUSTER_NAME)-control-plane; \
	fi
	@$(KIND) get kubeconfig --name $(KIND_CLUSTER_NAME) --internal > dev/internal
	@KUBECONFIG=./dev/internal $(DOCKER) compose up --build --detach stoker migrate
	@$(GO) run ./cmd/kubectl-approve_steamapps --kubeconfig dev/config --all
	@STOKER_URL=http://localhost:5050 yarn $@

.PHONY: manifests
manifests: internal/stoker/stokercr/config/crd

.PHONY: internal/stoker/stokercr/config/crd
internal/stoker/stokercr/config/crd: controller-gen
	@$(CONTROLLER_GEN) crd webhook paths="./..." output:crd:artifacts:config=$@

.PHONY: config
config: manifests

.PHONY: generate
generate: controller-gen
	@$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt vet test
fmt vet test:
	@$(GO) $@ ./...

.PHONY: lint
lint: golangci-lint fmt
	@$(GOLANGCI_LINT) config verify
	@$(GOLANGCI_LINT) run --fix

.PHONY: gen
gen: generate

.PHONY: internal/stoker/swagger.json
internal/stoker/swagger.json: swag
	@$(SWAG) fmt -g api.go --dir internal/stoker
	@$(SWAG) init -g api.go --dir internal/stoker --output internal/stoker --outputTypes json --parseInternal
	@sed 's/stoker\.//g' $@ > internal/stoker/swagger.json.tmp
	@cat internal/stoker/swagger.json.tmp > $@
	@rm internal/stoker/swagger.json.tmp
	@echo >> $@

.PHONY: swagger
swagger: internal/stoker/swagger.json

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	@mkdir -p $(LOCALBIN)

CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
KIND ?= $(LOCALBIN)/kind
SWAG ?= $(LOCALBIN)/swag

CONTROLLER_TOOLS_VERSION ?= v0.17.1
GOLANGCI_LINT_VERSION ?= v2.1.5
KIND_VERSION ?= v0.29.0
SWAG_VERSION ?= v1.16.4

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN)
$(CONTROLLER_GEN): $(LOCALBIN)
	@$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT)
$(GOLANGCI_LINT): $(LOCALBIN)
	@$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

.PHONY: kind
kind: $(KIND)
$(KIND): $(LOCALBIN)
	@$(call go-install-tool,$(KIND),sigs.k8s.io/kind,$(KIND_VERSION))

.PHONY: swag
swag: $(SWAG)
$(SWAG): $(LOCALBIN)
	@$(call go-install-tool,$(SWAG),github.com/swaggo/swag/cmd/swag,$(SWAG_VERSION))

define go-install-tool
@[ -f "$(1)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
} ;
endef
