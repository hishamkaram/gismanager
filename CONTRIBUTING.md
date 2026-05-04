# Contributing to gismanager

Thanks for your interest in contributing! This document describes how to get a development environment running and what we expect from pull requests.

## Development environment — Docker only

**This project does NOT install GDAL on the host machine.** All build, test, integration, and CI work runs inside containers. Requirements:

- Docker + Docker Compose v2
- `make`
- `git`

Clone and bootstrap:

```bash
git clone https://github.com/hishamkaram/gismanager
cd gismanager
make dev          # opens an interactive bash inside the dev container
```

The first `make dev` builds the dev image (Go 1.25.9 + golangci-lint v2.12.1 + govulncheck + goimports + libgdal-dev headers, all on top of `ghcr.io/osgeo/gdal:ubuntu-small-3.12.4`). Subsequent runs are cached.

Run targets non-interactively:

```bash
make build              # go build ./...
make test-unit          # unit tests, no Docker stack required
make test-integration   # integration tests against compose-managed GeoServer + PostGIS
make lint               # golangci-lint v2
make vuln               # govulncheck
make fmt                # gofmt -s -w + goimports -w
make tidy               # go mod tidy
```

For the integration suite:

```bash
make compose-test-up                    # default — GeoServer 2.28.0
GEOSERVER_VERSION=2.27.4 make compose-test-up   # LTS leg
make test-integration                   # runs go test -tags=integration ./...
make compose-test-down
```

See [`docker/README.md`](docker/README.md) for what's in the GeoServer image. (Or the Dockerfile itself, which is short.)

## Make targets

| Target | What it does |
|---|---|
| `make help` | Print this list. |
| `make dev` | Open an interactive bash inside the dev container. |
| `make build` | `go build ./...` |
| `make test-unit` | Unit tests with `-race` |
| `make test-integration` | Integration tests (`-tags=integration`) against the compose-test stack |
| `make lint` / `make lint-fix` | `golangci-lint run` (with `--fix` variant) |
| `make vuln` | `govulncheck ./...` |
| `make tidy` | `go mod tidy` |
| `make fmt` | `gofmt -s -w` + `goimports -w` |
| `make image` | Build the runtime image (`gismanager:local`) |
| `make compose-test-up` / `compose-test-down` / `compose-test-logs` | Manage the integration stack |
| `make clean` | Tear down dev + test compose volumes |

## Pull requests

1. **Branch from `master`.** Use a short descriptive name (`fix/url-escape`, `feat/new-format`). Never commit directly to `master`.
2. **One concern per PR.** Smaller PRs review faster.
3. **Conventional Commits.** Commit messages follow [Conventional Commits 1.0](https://www.conventionalcommits.org/) (`feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`, `build:`, `ci:`).
4. **Tests are mandatory.** New code needs unit tests (`*_test.go`, no build tag). Behavior changes also need an integration test entry (`*_integration_test.go` with `//go:build integration`). **Both layers run on every PR**: matrix CI exercises GeoServer 2.27.4 LTS + 2.28.0 stable; both legs must pass.
5. **Lint clean.** `make lint` must pass with zero warnings.
6. **No new runtime dependencies** without prior discussion. Test-only deps are fine.
7. **No GDAL on the host.** Anything that requires GDAL must run inside the dev container. PRs that add `apt-get install libgdal-dev` to host docs / scripts will be rejected.
8. **No `panic(` in library code.** Panics may live only in `cmd/*` `main.go` entry points (and PR 4 already removed the panics there). Library code returns errors wrapped with sentinels from `errors.go`.
9. **All CI checks must pass before merge.** The required gates on every PR: `Lint`, `Unit tests (Go 1.25)`, `govulncheck`, `Analyze (Go)` (CodeQL), `GeoServer 2.27.4`, `GeoServer 2.28.0`. Don't merge with any check red, pending, or skipped.
10. **Squash merge.** PRs are squash-merged into `master` so each merge produces exactly one commit on the trunk.

## Project layout

```
.
├── *.go                          # Library: ManagerConfig, GdalLayer, GISError, sentinels (datastore.go, errors.go, layer.go, log.go, manager.go, utils.go, vars.go)
├── *_test.go                     # Unit tests
├── *_integration_test.go         # Integration tests (//go:build integration)
├── cmd/
│   ├── gismanager/               # Full pipeline CLI
│   └── layerSchema/              # Read-only schema printer
├── internal/
│   └── zipx/                     # stdlib archive/zip extractor (zip-slip rejection + size cap)
├── docs/                         # Architecture, version compat
├── docker/
│   ├── Dockerfile                # GeoServer image used by the integration stack
│   ├── env/geoserver.env         # GeoServer JVM + admin password config
│   └── postgis/init/             # PostGIS init SQL (CREATE EXTENSION postgis)
├── docker-compose.yml            # Dev stack (just the dev shell)
├── docker-compose.test.yml       # Integration stack (GeoServer + PostGIS + test-runner)
├── Dockerfile                    # Multi-stage: dev (Go + tools) / build / runtime (binaries + libgdal)
├── .devcontainer/                # VS Code devcontainer config
├── .github/                      # Workflows, issue + PR templates, CODEOWNERS, Dependabot
├── .claude/                      # Claude Code config: agents, skills, commands
└── testdata/                     # GIS file fixtures for unit + integration tests
```

## Reporting bugs / asking questions

- **Bugs:** open a GitHub issue with the bug-report template.
- **Security issues:** see [SECURITY.md](SECURITY.md). Do not open a public issue.
- **Questions:** GitHub Discussions if available, otherwise an issue.

## Releasing

Releases are cut by maintainers via tags (`v1.x.y`). After a tag is pushed, `release.yml` (added in a future PR) will assemble release notes from Conventional Commit messages.

`CHANGELOG.md` follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). The `[Unreleased]` block at the top of the file accumulates entries between tags; cutting a release promotes the block contents into a dated stanza and leaves a fresh `[Unreleased]` placeholder.

## Working with Claude Code in this repo

The repo ships project-scoped Claude Code config so any contributor using Claude Code (CLI, IDE extensions, web app) gets the same conventions and shortcuts. Auto-loaded from `CLAUDE.md` (root) and `.claude/`.

| Type | Name | Purpose |
|---|---|---|
| Slash command | `/integration-test [version]` | Boot the docker-compose-test stack and run the integration suite (default 2.28; `2.27` for the LTS leg). |
| Slash command | `/lint-fix` | Run `golangci-lint --fix` + `gofmt` + `goimports` inside the dev container; report the diff and any remaining manual fixes. |
| Skill | `/gismanager-quirks` | Catalog of GDAL-binding + wire-format quirks gismanager works around (auto-loads when relevant). |
| Skill | `/docker-only-dev` | Reference for the "shell every Go command into Docker" pattern (auto-loads when relevant). |
| Subagent | `go-reviewer` | Idiom + sentinel-error + slog review of changed Go files. |
| Subagent | `gdal-reviewer` | CGo + lukeroth/gdal-binding review (catches OGR API drift, leaked feature handles, etc.). |
| Subagent | `integration-runner` | Boots the gismanager test stack, runs integration tests, diagnoses failures from container logs. |

Personal per-machine settings live in `.claude/settings.local.json` (gitignored). The team baseline is intentionally minimal; extend if the project grows enough contributors to warrant it.
