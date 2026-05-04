# gismanager Makefile.
#
# Every target shells into a Docker container — no host-side GDAL install. The
# dev image is built from the multi-stage Dockerfile (target=dev) on first use.
#
# Common commands:
#   make dev            interactive bash inside the dev container
#   make build          go build ./...
#   make test-unit      go test -race ./...
#   make test-integration  go test -tags=integration ./...   (added in PR 5)
#   make lint           golangci-lint run
#   make vuln           govulncheck ./...
#   make tidy           go mod tidy
#   make fmt            gofmt -s -w + goimports -w
#   make image          build the runtime image
#   make clean          tear down volumes (mod cache, build cache)
#
# All targets pass through any extra args via $(ARGS), e.g.:
#   make test-unit ARGS='-run TestXxx -v'

SHELL := /usr/bin/env bash
.SHELLFLAGS := -eu -o pipefail -c

DEV_SERVICE := dev
COMPOSE := docker compose
RUN := $(COMPOSE) run --rm -T $(DEV_SERVICE)
ARGS ?=

.PHONY: help dev shell build test test-unit test-integration lint lint-fix \
        vuln tidy fmt vet image clean compose-up compose-down

help: ## show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ---- interactive ----------------------------------------------------------

dev: ## open interactive bash in the dev container (with TTY)
	$(COMPOSE) run --rm $(DEV_SERVICE) bash

shell: dev ## alias for `make dev`

# ---- Go commands ----------------------------------------------------------

build: ## go build ./...
	$(RUN) go build $(ARGS) ./...

test: test-unit ## alias for test-unit

test-unit: ## go test -race ./... (no integration tag)
	$(RUN) go test -race -count=1 $(ARGS) ./...

test-integration: ## go test -tags=integration ./... (requires compose-up; added in PR 5)
	@echo "test-integration: integration suite is wired in PR 5; placeholder for now." >&2
	@exit 1

vet: ## go vet ./...
	$(RUN) go vet $(ARGS) ./...

tidy: ## go mod tidy
	$(RUN) go mod tidy $(ARGS)

fmt: ## gofmt -s -w + goimports -w
	$(RUN) bash -c 'gofmt -s -w . && goimports -w -local github.com/hishamkaram/gismanager .'

# ---- linting / vuln (live in PR 2) ----------------------------------------

lint: ## golangci-lint run (PR 2 wires the config)
	$(RUN) golangci-lint run $(ARGS) ./...

lint-fix: ## golangci-lint run --fix
	$(RUN) golangci-lint run --fix $(ARGS) ./...

vuln: ## govulncheck ./...
	$(RUN) govulncheck $(ARGS) ./...

# ---- images ---------------------------------------------------------------

image: ## build the runtime image (multi-stage `runtime` target)
	docker build --target runtime -t gismanager:local .

# ---- compose lifecycle ----------------------------------------------------

compose-up: ## start the dev container detached
	$(COMPOSE) up -d $(DEV_SERVICE)

compose-down: ## stop & remove dev container (keeps volumes)
	$(COMPOSE) down

clean: ## tear down volumes (clears Go mod + build caches)
	$(COMPOSE) down -v
