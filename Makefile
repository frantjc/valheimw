ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

GO ?= go
GIT ?= git

.PHONY: fmt vet test
fmt generate test:
	@$(GO) $@ ./...

.PHONY: download vendor verify
download vendor verify:
	@go mod $@

.PHONY: lint
lint: golangci-lint
	@$(GOLANGCI_LINT) run --fix

.PHONY: gen
gen: generate

SEMVER ?= 3.3.0

.PHONY: release
release:
	@$(GIT) tag v$(SEMVER)
	@$(GIT) push --tags

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	@mkdir -p $(LOCALBIN)

GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint

GOLANGCI_LINT_VERSION ?= v1.63.4

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT)
$(GOLANGCI_LINT): $(LOCALBIN)
	@$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
} ;
endef
