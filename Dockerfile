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

# Base image is digest-pinned to immunize the build from upstream re-tags
# (the OSGeo image is occasionally re-pushed for security patches; without a
# digest pin a "no-op" CI re-run can suddenly pull a different filesystem).
#
# Tag-equivalent at pin time: ghcr.io/osgeo/gdal:ubuntu-full-3.12.4 (multi-arch
# manifest list covering linux/amd64 and linux/arm64).
#
# v1.4 swapped the base from `ubuntu-small` to `ubuntu-full` to bring the
# Apache Parquet driver (and therefore GeoParquet support) into both the
# dev image and the published runtime image. Trade-off: image size grew
# from ~2 GB to ~4 GB. Operators who don't need GeoParquet can pin
# GDAL_BASE_DIGEST back to the ubuntu-small manifest list for a lighter
# image — see docs/conversions.md.
#
# To re-pin (when bumping GDAL or pulling in upstream patches):
#   docker buildx imagetools inspect ghcr.io/osgeo/gdal:ubuntu-full-<new>
# captures the manifest list digest. Keep the version in the comment
# above in sync with the digest below.
ARG GDAL_BASE_DIGEST=sha256:5828162cffed3af330034ae0c3ada30deb1cfdaecf37585f96ed0924f1d1dfb7

# ---------------------------------------------------------------------------
# Stage 1: dev — Go + GDAL dev headers + tooling
# ---------------------------------------------------------------------------
FROM ghcr.io/osgeo/gdal@${GDAL_BASE_DIGEST} AS dev

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
# Switch to Azure's Ubuntu mirror first (co-located with the GHA runner
# network; significantly more reliable than the public archive.ubuntu.com
# mirror, which periodically returns connection-refused). The 5-second
# reachability probe before the rewrite means developers building locally
# OFF the Azure network (i.e. anywhere outside GHA) automatically fall
# back to the original archive.ubuntu.com sources rather than hanging
# on a dead Azure-mirror connection. Then Acquire::Retries=3 absorbs
# any per-package transient at the apt layer.
RUN set -eux; \
    if [ -f /etc/apt/sources.list.d/ubuntu.sources ] && \
       curl -fsS --connect-timeout 5 -o /dev/null \
            http://azure.archive.ubuntu.com/ubuntu/ 2>/dev/null; then \
        echo "Azure Ubuntu mirror reachable — switching apt sources"; \
        sed -i \
            -e 's|http://archive.ubuntu.com|http://azure.archive.ubuntu.com|g' \
            -e 's|http://security.ubuntu.com|http://azure.archive.ubuntu.com|g' \
            /etc/apt/sources.list.d/ubuntu.sources; \
    else \
        echo "Azure Ubuntu mirror unreachable — keeping default sources"; \
    fi; \
    apt-get -o Acquire::Retries=3 update; \
    apt-get -o Acquire::Retries=3 install -y --no-install-recommends \
        build-essential \
        git \
        pkg-config; \
    rm -rf /var/lib/apt/lists/*

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

# Version metadata baked in via -ldflags. Defaults work for `make image`
# locally; the release workflow (.github/workflows/release.yml) overrides
# all three from the tag.
ARG VERSION=dev
ARG COMMIT=unknown
ARG DATE=unknown

COPY . /workspace
RUN set -eux; \
    LDFLAGS="-s -w \
      -X github.com/hishamkaram/gismanager/v2/cmd/internal/cli.Version=${VERSION} \
      -X github.com/hishamkaram/gismanager/v2/cmd/internal/cli.Commit=${COMMIT} \
      -X github.com/hishamkaram/gismanager/v2/cmd/internal/cli.Date=${DATE}"; \
    go mod download; \
    for bin in gismanager layerSchema gisconvert; do \
      go build -trimpath -ldflags="$LDFLAGS" -o "/out/$bin" "./cmd/$bin"; \
    done

# ---------------------------------------------------------------------------
# Stage 3: runtime — minimal image for shipping
# ---------------------------------------------------------------------------
FROM ghcr.io/osgeo/gdal@${GDAL_BASE_DIGEST} AS runtime

# No apt-install: the OSGeo base already provides ca-certificates +
# libpq5 (lib/pq's runtime dep). Skipping apt makes the runtime stage
# a one-shot copy of the binaries.

COPY --from=build /out/gismanager /usr/local/bin/gismanager
COPY --from=build /out/layerSchema /usr/local/bin/layerSchema
COPY --from=build /out/gisconvert /usr/local/bin/gisconvert

ENTRYPOINT ["/usr/local/bin/gismanager"]
