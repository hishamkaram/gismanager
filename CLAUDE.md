# Project memory for `github.com/hishamkaram/gismanager`

This file is auto-loaded by Claude Code at the start of every session in this repository. Keep it concise (~150 lines, hard cap 200) and focused on **what cannot be derived from the code itself**.

## Project identity

- **Module**: `github.com/hishamkaram/gismanager`
- **Purpose**: Go library + CLIs covering two GIS workflows. (1) **Publish pipeline** (`cmd/gismanager`): walk a directory of GIS data files (shapefile / GeoJSON / GeoPackage / KML), load them into PostGIS via GDAL bindings, then publish the resulting tables as GeoServer feature types — the product layer over [`hishamkaram/geoserver`](https://github.com/hishamkaram/geoserver). (2) **Conversion subsystem** (v1.2+, `cmd/gisconvert`): `ConvertVector` / `ConvertRaster` / `ToCOG` / `ReprojectRaster` — stateless wrappers over `gdal.VectorTranslate` / `gdal.Translate` / `gdal.Warp` exposing the `ogr2ogr` / `gdal_translate` / `gdalwarp` surface as a Go API plus a CLI.
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

- Sentinels live in `errors.go`: `ErrConfigInvalid`, `ErrUnsupportedFormat`, `ErrInvalidLayer`, `ErrInvalidDatasource`, `ErrPostGISConnect`, `ErrGeoServerPublish`, `ErrNoSourcesFound`.
- Typed `*GISError` with `Op`, `Source`, `Sentinel`, `Cause` fields and `errors.Is`/`As` semantics. `Unwrap` returns the underlying cause; `Is` matches the sentinel. The `*geoserver.APIError` from the v2 client is recoverable via `errors.As` against the wrapped cause.
- **Never compare error strings.** `errors.Is(err, ErrXxx)` is the only correct test.
- All public methods that can fail return wrapped `*GISError` (covered by tests in `errors_test.go`).

## Logging

- `*slog.Logger` directly (since PR 4). No wrapper, no `logrus`. The default `GetLogger()` returns a stderr text-handler logger; production callers should construct their own `*slog.Logger` with whatever handler they want and pass it on `ManagerConfig.logger` (functional-options constructor lands later).
- Internal call sites use structured logging — `logger.Error("operation", "key", value, "err", err)` with key/value pairs, not printf-style.
- Library logs Debug for transient details, Warn for retry-exhausted or unexpected wire shapes, Error for failed external calls (PostGIS, GeoServer, file read). No Info chatter from the library; `cmd/*` may emit Info on successful publish events.

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

Subagents (delegated workers, own context window):

- `go-reviewer` — Go-idiom review of changed files: errors.Is/As over string-matching, ctx-first method shape, slog discipline, no library-code panics, the `*GISError` sentinel-wrapping contract.
- `integration-runner` — boots the gismanager compose-test stack, runs the integration suite, dumps logs on failure.
- `gdal-reviewer` — CGo + `lukeroth/gdal`-binding review: integer-width drift (`Layer.Feature(int64)`), OGR handle ownership / Destroy semantics, soname pinning, OGR driver-name constants.

Skills (loadable knowledge / procedures):

- `/gismanager-quirks` — catalog of GDAL-binding + GeoServer wire-format quirks the codebase works around (auto-loads on relevant edits).
- `/docker-only-dev` — reference for the "shell every Go command into Docker" pattern (auto-loads on build/test/run questions).

Slash commands (callable recipes):

- `/integration-test [version]` — boot compose-test stack and run integration suite (default 2.28.0; pass `2.27` for the LTS leg).
- `/lint-fix` — golangci-lint with autofix + gofmt + goimports, all in-container.
