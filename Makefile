GO ?= go
GOLANGCI-LINT ?= golangci-lint
GIT ?= git
DOCKER ?= docker

all: fmt lint

fmt generate:
	@$(GO) $@ ./...

lint:
	@$(GOLANGCI-LINT) run --fix

gen: generate

SEMVER ?= 3.0.0

release:
	@$(GIT) tag v$(SEMVER)
	@$(GIT) push --tags

command/image.tar:
	@$(DOCKER) build -t tmp command
	@$(DOCKER) save tmp -o command/image.tar
	@$(DOCKER) rmi tmp

.PHONY: all fmt generate lint proto gen release command/image.tar
