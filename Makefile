GO ?= go
GOLANGCI-LINT ?= golangci-lint
GIT ?= git

all: fmt lint

fmt generate:
	@$(GO) $@ ./...

lint:
	@$(GOLANGCI-LINT) run --fix

gen: generate

SEMVER ?= 1.2.6

release:
	@$(GIT) tag v$(SEMVER)
	@$(GIT) push --tags


.PHONY: all fmt generate lint proto gen release
