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
lint: golangci-lint fmt
	@$(GOLANGCI_LINT) config verify
	@$(GOLANGCI_LINT) run --fix

.PHONY: gen
gen: generate

.PHONY: internal/stokerhttp
internal/stokerhttp: swag
	@$(SWAG) fmt --dir $@
	@$(SWAG) init --dir $@ --output $@ --outputTypes json --parseInternal
	@sed -i 's/stokerhttp\.//g' $@/swagger.json
	@echo >> $@/swagger.json

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	@mkdir -p $(LOCALBIN)

GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
SWAG ?= $(LOCALBIN)/swag

GOLANGCI_LINT_VERSION ?= v1.64.5
SWAG_VERSION ?= v1.16.4

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT)
$(GOLANGCI_LINT): $(LOCALBIN)
	@$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

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
