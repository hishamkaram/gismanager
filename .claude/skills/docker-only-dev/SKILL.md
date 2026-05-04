---
description: Reference for the Docker-only dev pattern gismanager uses. Loads automatically when the user asks how to build/test/lint, or when guidance involves running Go commands locally.
when_to_use: User says "how do I build", "run go test", "run lint", "where does this run", "install GDAL", "add libgdal-dev", or any path that would otherwise lead to a host-side install.
---

# Docker-only dev

gismanager pins to **`ghcr.io/osgeo/gdal:ubuntu-small-3.12.4`** and shells every Go command into a dev container. The host machine never installs GDAL, libpq-dev, or any of the heavy native deps. This is a **rule**, not a convention — see `feedback_dockerize_native_deps.md` in this user's auto-memory.

## What's in the dev image

The multi-stage `Dockerfile` at the repo root has three targets:

| Target | Contents |
|---|---|
| `dev` | OSGeo GDAL base + Go 1.25.9 + golangci-lint v2.12.1 + govulncheck + goimports + libpq-dev + pkg-config + build-essential |
| `build` | Compiles binaries from source |
| `runtime` | OSGeo GDAL base + libpq5 + the gismanager + layerSchema binaries |

All Go invocations the user makes locally happen inside `dev`.

## How to run anything

The `Makefile` is the canonical entry point. Every target shells through `docker compose run --rm -T dev <command>`:

| You want to … | Run |
|---|---|
| Open an interactive bash shell inside the container | `make dev` |
| `go build ./...` | `make build` |
| Unit tests (`go test -race ./...`) | `make test-unit` |
| Integration tests (boots GeoServer + PostGIS first) | `make compose-test-up && make test-integration` |
| `golangci-lint run` (or with autofix) | `make lint` / `make lint-fix` |
| `govulncheck ./...` | `make vuln` |
| `gofmt -s -w .` + `goimports -w .` | `make fmt` |
| `go mod tidy` | `make tidy` |
| Build the runtime image | `make image` |
| Tear everything down (clears volumes too) | `make clean` |

If a contributor wants to add a new target, add it to the Makefile in the same shape: `$(RUN) <go invocation> $(ARGS)`. Don't add `go` invocations that run on the host directly.

## What NOT to do

- ❌ `apt-get install libgdal-dev` on the host (or in any host-side script / README / contributing doc).
- ❌ `apt-get install libgdal-dev` inside the dev image — the OSGeo base already provides headers at `/usr/include/gdal_*.h` + `gdal.pc`. Adding apt's libgdal-dev shadows the bundled GDAL with an older soname (`libgdal.so.34` vs the bundled `libgdal.so.38`) and produces binaries that fail to load at runtime. See `gismanager-quirks` quirk #5.
- ❌ Pinning `:latest` for the OSGeo image. Always pin to a specific GDAL minor (`ubuntu-small-3.12.4`).
- ❌ Running `go test` directly on the host — even if the host has Go installed, the test will fail to find `<gdal.h>`. The error is misleading; users won't know the test was intended to run in Docker.
- ❌ Adding a "fast path" that skips Docker. There isn't one. The CGo dep makes the host-vs-container distinction load-bearing.

## VS Code

Open the source tree inside the dev container via `.devcontainer/devcontainer.json`. That's how `gopls` resolves the `<gdal.h>` CGo headers — without devcontainer, the editor flags every `lukeroth/gdal` import as unresolved.

## CI

The same dev image is used in CI:

```yaml
container: ghcr.io/osgeo/gdal:ubuntu-small-3.12.4
```

Each job apt-installs a minimal helper set (build-essential, pkg-config, ca-certificates, git) on top of the base — but never `libgdal-dev`. The `actions/setup-go` step pulls Go 1.25 into the container.

## Image hygiene

The runtime image (`make image`) is `~500 MB`. Most of that is the OSGeo base plus the libpq runtime; the gismanager binary itself is `~10 MB`. Don't try to `FROM scratch` or `FROM gcr.io/distroless/...` — CGo + libgdal needs glibc + the OSGeo libgdal at runtime. The same base image used at build time is the cleanest runtime base.

If image size becomes a concern, evaluate `osgeo/gdal:alpine-small-X.Y.Z` (musl-based, smaller). Alpine + CGo + GDAL has historically been touchier than Ubuntu — verify the integration suite end-to-end before switching the pinned base.
