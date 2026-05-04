# Project memory for `github.com/hishamkaram/gismanager`

This file is auto-loaded by Claude Code at the start of every session in this repository. Keep it concise (~150 lines, hard cap 200) and focused on **what cannot be derived from the code itself**.

## Project identity

- **Module**: `github.com/hishamkaram/gismanager`
- **Purpose**: Go library + CLIs that walk a directory of GIS data files (shapefile / GeoJSON / GeoPackage / KML), load them into PostGIS via GDAL bindings, then publish the resulting tables as GeoServer feature types. The product layer that sits on top of the lower-level [`hishamkaram/geoserver`](https://github.com/hishamkaram/geoserver) Go client.
- **Maintainer / display name**: **Hesham Karm** (note: no trailing 'a'; first name is "Hesham" not "Hisham"). Use this exact spelling in LICENSE, README authorship, AUTHORS files, commit signatures, and any user-facing credits. The legacy GitHub handle `hishamkaram` and the historical email are intentional — do not "correct" them.

## Build & dev — Docker only

**This project does NOT install GDAL on the host machine.** All build, test, integration, and CI work runs inside the multi-stage Dockerfile rooted at the repo's `Dockerfile`, based on `ghcr.io/osgeo/gdal:ubuntu-small-3.12.4`.

- `make dev` opens an interactive bash inside the dev container.
- Every other `make` target shells into the same container non-interactively (`docker compose run --rm -T dev <cmd>`).
- VS Code users: `.devcontainer/devcontainer.json` lets the editor open the source tree inside the container so `gopls` resolves the GDAL CGo headers.
- CI runs the same Docker image as the build container.
- **Never** add `apt-get install gdal-dev` (or equivalent) to host docs, READMEs, or scripts. If a contributor lacks Docker, the entry point is "install Docker", not "install GDAL".

## Versioning regime

- **First stable tag is `v1.0.0`.** Module path stays `github.com/hishamkaram/gismanager` (no `/v2` suffix — gismanager has no released versions to preserve, so the post-revival API is the v1.0 baseline).
- The repo before this revival was last tagged in 2018, never released, and broken on Go ≥ 1.13 (issue #1). The post-revival v1.0.0 is the first-ever release; previous source is reachable only by pre-revival commit SHAs.
- **Don't auto-tag releases.** Tagging is an explicit user action — never run `git tag` or push tags from a Claude session without explicit user authorization.

## Test split

- `*_unit_test.go` (no build tag) — **fast, hermetic, httptest-based.** Run by default via `make test-unit`.
- `*_integration_test.go` with `//go:build integration` — **integration tests against a real GeoServer + PostGIS stack.** Never invoke without `--tags=integration` AND a live compose stack via `make compose-up`.
- The `make test-unit` and `make test-integration` targets are the canonical entry points; CI mirrors them.
- **Both unit and integration tests are mandatory on every PR.** Integration runs against GeoServer 2.27.4 LTS and 2.28.0 stable matching the upstream `hishamkaram/geoserver` client's matrix; both legs must pass for the PR to merge.

## GeoServer client integration

gismanager publishes layers via [`github.com/hishamkaram/geoserver/v2`](https://github.com/hishamkaram/geoserver) — the **v2 line** at the repo root. Key idioms:

- Constructor: `geoserver.New(serverURL, geoserver.WithBasicAuth(user, pass), geoserver.WithLogger(slogLogger))`.
- No `Exists` methods. Test for presence via `Get` + `errors.Is(err, geoserver.ErrNotFound)`.
- Hierarchical scoping: `c.Datastores.InWorkspace(ws).Create(ctx, ...)`, `c.FeatureTypes.InWorkspace(ws).InDatastore(ds).Create(ctx, ft)`.
- Errors surface as `*geoserver.APIError`; sentinel matching via `errors.Is`.
- Every method takes `ctx context.Context` as the first argument. gismanager mirrors this — every public method here also takes `ctx` first.

## Context handling (mandatory for new exports)

gismanager is **context-first**: every exported method on `*Manager` takes `ctx context.Context` as its first argument. No `*Context` twins, no `context.Background()` delegators inside the library. If a caller has no context, they pass `context.Background()` at the call site.

## Typed errors

- Sentinel set in `errors.go`: `ErrNoDriver`, `ErrUnsupportedFormat`, `ErrInvalidLayer`, `ErrPostGISConnect`, `ErrGeoServerPublish`, `ErrNoSourcesFound`.
- Typed `*GISError` with `Op`, `Source`, `Cause` fields, `errors.Is`/`As` semantics. Wraps `*geoserver.APIError` and underlying GDAL / PostGIS errors with `%w`.
- **Never compare error strings.** `errors.Is(err, ErrXxx)` is the only correct test.

## Logging

- `*slog.Logger` directly. No wrapper, no `logrus`. Configure via `WithLogger(l *slog.Logger)`. Default is `slog.New(slog.DiscardHandler)` (silent).
- Internal call sites use structured logging — `logger.Debug(msg, args...)` with key/value pairs, not printf-style.
- Library logs Debug for HTTP/GDAL details, Warn for retry-exhausted or unexpected wire shapes, Error for protocol violations or deserialization failures. No Info chatter.

## Concurrency

- `*Manager` is **immutable after `New(...)` returns.** All struct fields are private or pointers to sub-clients set once at construction and never reassigned. Concurrent use across goroutines is safe by design.
- Don't introduce shared mutable state. If you need per-call state, allocate it inside the method.
- The CGo GDAL bindings have their own thread-safety constraints — review `lukeroth/gdal` upstream docs before parallelizing GDAL operations.

## GeoServer + PostGIS version matrix

- **Supported: GeoServer 2.27 LTS + 2.28 stable + PostGIS 16.** Integration tests run against both via the CI matrix.
- **GDAL 3.12.4** pinned in the `Dockerfile`. Bumping requires re-running the integration suite end-to-end.
- Don't add code paths gated on other GeoServer / PostGIS / GDAL versions without adding a corresponding CI matrix entry.

## Build & lint surfaces

- `make` is the canonical entry point. CI workflow names match Make targets.
- Don't bypass `make` with raw `go test`/`golangci-lint` invocations (they'd run on the host without GDAL).
- `.golangci.yml` enables: errcheck, govet, staticcheck, ineffassign, unused, bodyclose, errorlint, noctx, copyloopvar, revive, gocritic, misspell, unconvert, gosec.
- Don't add `//nolint:` comments outside the existing exemptions.

## Conventions and don'ts

- **Never commit directly to `master`.** Always create a feature branch, push it, open a PR, wait for CI to go green, then squash-merge.
- **Never add Claude (or any AI assistant) as a git co-author.** Do not append `Co-Authored-By: Claude ...` trailers. Commit messages are authored by the user only.
- **Never commit planning markdowns** — design docs, revival plans, research notes belong in `~/.claude/plans/`, not in this repo.
- **No panics in library code.** gismanager library code (root + `internal/`) must contain zero `panic(` calls. The two CLIs (`cmd/gismanager`, `cmd/layerSchema`) may panic at the entry point only when a config error is fatal.
- **No new runtime dependencies** without prior discussion. The dependency surface is intentionally small: `lukeroth/gdal`, `lib/pq`, `hishamkaram/geoserver/v2`, `gopkg.in/yaml.v3`, stdlib.
- **No `mholt/archiver`** — deprecated upstream; use `internal/zipx` (stdlib `archive/zip`).
- **Don't auto-tag releases** and don't merge a PR with red or pending CI — both are explicit user actions.

## Index of project Claude config

Subagents (delegated workers, own context window) — *populated in PR 6:*

- `go-reviewer` — Go-idiom review of changed files (adapted from the geoserver repo).
- `integration-runner` — boots the gismanager compose stack, runs integration suite, dumps logs on failure.
- `gdal-reviewer` — CGo + GDAL-binding review (catches Feature/int64 drift, leaked OGR handles, etc.).

Skills (loadable knowledge / procedures) — *populated in PR 6:*

- `/gismanager-quirks` — catalog of GDAL-binding quirks the codebase works around.
- `/docker-only-dev` — reference for the "shell every Go command into Docker" pattern.

Slash commands (callable recipes) — *populated in PR 6:*

- `/integration-test` — boot compose stack and run integration suite.
- `/lint-fix` — golangci-lint with autofix + gofmt + goimports, all in-container.
