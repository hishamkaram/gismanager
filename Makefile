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
COMPOSE_TEST := $(COMPOSE) -f docker-compose.test.yml
RUN := $(COMPOSE) run --rm -T $(DEV_SERVICE)
ARGS ?=
GEOSERVER_VERSION ?= 2.28.0

.PHONY: help dev shell build build-cli test test-unit test-integration lint lint-fix \
        vuln tidy fmt vet image clean fetch-testdata ci benchmark \
        compose-up compose-down compose-test-up compose-test-down compose-test-logs

help: ## show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ---- interactive ----------------------------------------------------------

dev: ## open interactive bash in the dev container (with TTY)
	$(COMPOSE) run --rm $(DEV_SERVICE) bash

shell: dev ## alias for `make dev`

# ---- Go commands ----------------------------------------------------------

build: ## go build ./...
	$(RUN) go build $(ARGS) ./...

# Version metadata injected into the cmd/internal/cli package via -ldflags.
# Override any of these by exporting them in the environment before invoking
# `make build-cli` — useful in release workflows that pin a specific tag.
LDFLAGS_VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS_COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS_DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS_PKG     := github.com/hishamkaram/gismanager/cmd/internal/cli
LDFLAGS         := -X $(LDFLAGS_PKG).Version=$(LDFLAGS_VERSION) \
                   -X $(LDFLAGS_PKG).Commit=$(LDFLAGS_COMMIT) \
                   -X $(LDFLAGS_PKG).Date=$(LDFLAGS_DATE)

build-cli: ## go build ./cmd/... with version ldflags injected
	$(RUN) go build -ldflags='$(LDFLAGS)' $(ARGS) ./cmd/...

test: test-unit ## alias for test-unit

test-unit: ## go test -race ./... (no integration tag)
	$(RUN) go test -race -count=1 $(ARGS) ./...

test-integration: fetch-testdata ## go test -tags=integration ./... inside the test-runner container against the compose-test stack (requires compose-test-up first)
	$(COMPOSE_TEST) run --rm -T test-runner go test -race -tags=integration -timeout=15m -count=1 $(ARGS) ./...

# CI umbrella: the four checks every PR's CI matrix runs in ~3 min.
# Excludes test-integration on purpose — that needs a live stack and
# is its own `make test-integration` target. Run that separately via
# `make compose-test-up && make test-integration && make compose-test-down`
# (or via the `/integration-test` slash command).
ci: lint vet test-unit vuln ## fast CI proxy: lint + vet + unit + vuln (no integration)

# Benchmarks. The repo currently has no `func Benchmark*` cases, but
# `make benchmark` is wired so a future `BenchmarkPublishAll` /
# `BenchmarkConvertVector` can be exercised the same way as the rest
# of the test surface. Pass `-benchtime=`, `-cpu=`, etc. via ARGS.
benchmark: ## go test -bench=. -benchmem ./... (no benchmarks yet; placeholder for future)
	$(RUN) go test -bench=. -benchmem -run='^$$' $(ARGS) ./...

fetch-testdata: ## download/refresh testdata fixtures listed in testdata/manifest.sha256 (idempotent; runs on host since curl + sha256sum are stdlib)
	@bash scripts/fetch-testdata.sh

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

compose-test-up: ## boot integration stack (GeoServer + PostGIS); GEOSERVER_VERSION=2.27.4 for LTS leg
	GEOSERVER_VERSION=$(GEOSERVER_VERSION) $(COMPOSE_TEST) up -d --wait geoserver postgis

compose-test-down: ## tear down integration stack + volumes
	$(COMPOSE_TEST) down -v

compose-test-logs: ## tail integration stack logs
	$(COMPOSE_TEST) logs -f

clean: ## tear down volumes (clears Go mod + build caches)
	$(COMPOSE) down -v
	$(COMPOSE_TEST) down -v 2>/dev/null || true
