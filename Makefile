GO ?= go
GOLANGCI-LINT ?= golangci-lint
GIT ?= git

fmt generate test:
	@$(GO) $@ ./...

lint:
	@$(GOLANGCI-LINT) run --fix

gen: generate

SEMVER ?= 3.1.2

release:
	@$(GIT) tag v$(SEMVER)
	@$(GIT) push --tags

.PHONY: fmt generate lint gen release
