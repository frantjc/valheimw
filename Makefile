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

SEMVER ?= 3.0.1

release:
	@$(GIT) tag v$(SEMVER)
	@$(GIT) push --tags

TAG ?= boil.frantj.cc/base:${SEMVER}

command/image.tar:
	@$(DOCKER) build -t ${TAG} command
	@$(DOCKER) save ${TAG} -o command/image.tar
	@$(DOCKER) rmi ${TAG}

.PHONY: all fmt generate lint proto gen release command/image.tar
