# syntax=docker/dockerfile:1.7

# Multi-stage Dockerfile for gismanager.
#
# All development, build, test, and CI work runs inside containers — no GDAL
# install on the host. Base image is the official OSGeo GDAL image so libgdal
# headers and the OGR drivers we need (Shapefile, GeoJSON, GeoPackage, KML,
# PostgreSQL/PostGIS) are present.
#
# Stages:
#   - dev:     Go toolchain + libgdal-dev + lint/vuln tools. The default target
#              for `make dev` / `make build` / `make test-unit` / CI.
#   - runtime: libgdal only + the compiled binaries. Produces the published
#              images.

ARG GDAL_VERSION=3.12.4
ARG GO_VERSION=1.25.9
ARG GOLANGCI_LINT_VERSION=v2.12.1
ARG GOVULNCHECK_VERSION=latest

# ---------------------------------------------------------------------------
# Stage 1: dev — Go + GDAL dev headers + tooling
# ---------------------------------------------------------------------------
FROM ghcr.io/osgeo/gdal:ubuntu-small-${GDAL_VERSION} AS dev

ARG GO_VERSION
ARG GOLANGCI_LINT_VERSION
ARG GOVULNCHECK_VERSION
ARG TARGETARCH

ENV DEBIAN_FRONTEND=noninteractive

# OS packages. Note: do NOT `apt-get install libgdal-dev` — the OSGeo base
# image already ships GDAL headers + libgdal.so.38 + gdal.pc. Adding the
# distro's libgdal-dev (Ubuntu 24's GDAL 3.6, libgdal.so.34) clobbers the
# bundled GDAL and produces binaries linked against the wrong soname. See
# https://github.com/OSGeo/gdal/blob/master/docker/README.md.
#
# We don't apt-install libpq-dev — lib/pq is pure-Go and doesn't need C
# headers. We also don't install postgresql-client (psql) — gismanager
# never shells out to psql; the integration tests connect via Go's
# database/sql.
#
# Acquire::Retries=3 is apt's native retry — preferred over a shell loop
# wrapper because it retries at the per-package level and survives
# partial-progress failures cleanly. Same flag in ci.yml + codeql.yml.
RUN apt-get -o Acquire::Retries=3 update \
    && apt-get -o Acquire::Retries=3 install -y --no-install-recommends \
        build-essential \
        git \
        pkg-config \
    && rm -rf /var/lib/apt/lists/*

# Install Go from the official tarball (Ubuntu's apt go is usually older than
# the active toolchain). Pin GO_VERSION at build time.
RUN set -eux; \
    arch="${TARGETARCH:-amd64}"; \
    curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${arch}.tar.gz" \
        | tar -C /usr/local -xz; \
    /usr/local/go/bin/go version

ENV PATH="/usr/local/go/bin:/root/go/bin:${PATH}" \
    GOPATH=/root/go \
    GOFLAGS="-buildvcs=false" \
    CGO_ENABLED=1

# Install golangci-lint v2, govulncheck, and goimports via go install (the
# install.sh script's checksum validation has been flaky for v2.12.1; go
# install is the authoritative path per the golangci-lint v2 docs).
# goimports lives outside golangci-lint because the project's `make fmt`
# target shells to it directly (matching the geoserver-client convention).
RUN go install "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}" \
    && go install "golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION}" \
    && go install golang.org/x/tools/cmd/goimports@latest \
    && cp /root/go/bin/golangci-lint /usr/local/bin/ \
    && cp /root/go/bin/govulncheck /usr/local/bin/ \
    && cp /root/go/bin/goimports /usr/local/bin/ \
    && golangci-lint --version \
    && govulncheck -version \
    && goimports --help 2>&1 | head -1

WORKDIR /workspace

# Default: drop into bash so `docker compose run --rm dev` is interactive.
CMD ["bash"]

# ---------------------------------------------------------------------------
# Stage 2: build — produce the static-ish binaries (CGo, dynamically linked)
# ---------------------------------------------------------------------------
FROM dev AS build

COPY . /workspace
RUN go mod download \
    && go build -trimpath -ldflags="-s -w" -o /out/gismanager ./cmd/gismanager \
    && go build -trimpath -ldflags="-s -w" -o /out/layerSchema ./cmd/layerSchema

# ---------------------------------------------------------------------------
# Stage 3: runtime — minimal image for shipping
# ---------------------------------------------------------------------------
FROM ghcr.io/osgeo/gdal:ubuntu-small-${GDAL_VERSION} AS runtime

# No apt-install: the OSGeo base already provides ca-certificates +
# libpq5 (lib/pq's runtime dep). Skipping apt makes the runtime stage
# a one-shot copy of the binaries.

COPY --from=build /out/gismanager /usr/local/bin/gismanager
COPY --from=build /out/layerSchema /usr/local/bin/layerSchema

ENTRYPOINT ["/usr/local/bin/gismanager"]
