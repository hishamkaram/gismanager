# Changelog

All notable changes to `github.com/hishamkaram/gismanager` are documented here. Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/); the project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [2.0.0] — 2026-05-07

### Context

The v2 line ships gismanager as a subpackage-organized library
(`github.com/hishamkaram/gismanager/v2`) with the **publish pipeline**,
**conversion subsystem**, and the typed-error machinery each living in
its own subpackage. Six focused PRs (#53–#58) sequenced for review
delivered the layout; the v1.x line (release branch `release/v1.x`)
stays maintained for users who can't migrate yet. See
[`MIGRATING.md`](./MIGRATING.md) for the v1 → v2 import-rewrite recipe.

The v2 cycle is intentionally **structural** rather than feature-driven:
the library API surface is the same shape gismanager has had since v1.4
plus a few targeted cleanups and one new option. No new external runtime
dependencies were added; the dep surface stays the small set documented
in `CLAUDE.md`.

### Module path

- **`github.com/hishamkaram/gismanager` → `github.com/hishamkaram/gismanager/v2`** (PR #56)

  Every external import requires a path bump; type/function names are
  otherwise either unchanged or renamed per the table in MIGRATING.md.
  The release/v1.x branch keeps the old import path supported for
  patch releases (security fixes, etc.).

### Added

- **`github.com/hishamkaram/gismanager/v2/publish`** (PR #55) —
  the publish pipeline subpackage. Hosts `Manager` (was `ManagerConfig`),
  `Layer` (was `GdalLayer`), `New`, `FromConfig`, `WithLogger` /
  `WithGeoserver` / `WithDatastore` / `WithSource`, `(*Manager).Walk`,
  `(*Manager).PublishAll`, `(*Manager).OpenSource` / `GetDriver` /
  `Validate`, `(*Layer).LayerToPostgis` / `PublishGeoserverLayer` /
  `GetLayerSchema` / `Features` / `GeometryName`, plus the configuration
  struct types (`GeoserverConfig`, `DatastoreConfig`, `SourceConfig`,
  `LayerField`, `WalkItem`, `Option`).
- **`github.com/hishamkaram/gismanager/v2/convert`** (PR #54) —
  the stateless conversion subsystem subpackage. Hosts `ConvertVector`,
  `ConvertRaster`, `ToCOG`, `ReprojectRaster`, `Rasterize`, `BuildVRT`,
  `DEMProcessing`, `ToPMTiles`, plus 60-ish `WithVector*` / `WithRaster*`
  / `WithRasterize*` / `WithVRT*` / `WithDEM*` / `WithPMTiles*` option
  helpers. Option types renamed: `VectorConvertOption` → `VectorOption`,
  `RasterConvertOption` → `RasterOption` (the redundant `Convert`
  prefix made sense at the root but is noise inside `convert/`).
- **`github.com/hishamkaram/gismanager/v2/errs`** (PR #56) —
  promoted from `internal/errs` to a public subpackage. Hosts the
  `*GISError` envelope and the `ErrConfigInvalid` / `ErrUnsupportedFormat`
  / `ErrInvalidLayer` / `ErrInvalidDatasource` / `ErrPostGISConnect` /
  `ErrGeoServerPublish` / `ErrNoSourcesFound` / `ErrConvertFailed`
  sentinels. v2 callers do `errors.Is(err, errs.ErrConvertFailed)`
  rather than the v1 `errors.Is(err, gismanager.ErrConvertFailed)`.
- **`publish.WithPublishConcurrency(n int)`** (PR #58) — caps how
  many GeoServer feature-type creations `(*Manager).PublishAll`
  dispatches in parallel. Zero/negative → fall back to package
  default (currently 8); explicit positive → cap; `n=1` forces
  strictly serial publish (useful for diagnostic runs against
  finicky GeoServer instances). Walk + LayerToPostgis stay serial
  regardless — only the HTTP-only publish step parallelizes, since
  the lukeroth/gdal CGo handles aren't reentrancy-safe across
  goroutines.
- **`MIGRATING.md`** — new top-level migration guide covering the
  module-path bump, every type/function rename, the dropped
  v1-deprecated methods, the new option, and concrete `sed`-style
  migration recipes for typical caller patterns.

### Changed

- **`ManagerConfig` → `Manager`** in `publish/` (PR #55). "Config" was
  a misnomer — the type IS the manager, not its config. Method sets
  (`Validate`, `OpenSource`, `GetDriver`, `Walk`, `PublishAll`,
  `GetGeoserverCatalog`, `NewLayer`) flow with the type.
- **`GdalLayer` → `Layer`** in `publish/` (PR #55). Drop the redundant
  `Gdal` prefix — clear from package context. Method sets
  (`LayerToPostgis`, `PublishGeoserverLayer`, `GetLayerSchema`,
  `Features`, `GeometryName`) flow with the type.
- **`*ConvertOption` → `*Option`** in `convert/` (PR #54). `VectorOption`
  / `RasterOption` (`RasterizeOption` / `VRTOption` / `DEMOption` /
  `PMTilesOption` were already concise, unchanged).
- **Internal extraction of error machinery + default logger** to
  `internal/errs` (Phase 1, PR #53) then promotion of `errs` to
  public (Phase 4, PR #56). `internal/slogx` stays internal in
  v2; v2 callers construct their own `*slog.Logger` and pass it
  via `publish.WithLogger`.

### Removed

- **`gismanager.GISError` / `gismanager.Err*` sentinels** (root) —
  replaced by `errs.GISError` / `errs.Err*`. Pre-v1.4 callers that
  matched on string error messages keep working (errors.Is preserves
  identity across the boundary), but type-name compile-time references
  must update.
- **`gismanager.ManagerConfig` / `gismanager.GdalLayer` / `gismanager.New`
  / `gismanager.FromConfig` / `gismanager.WithLogger` / `gismanager.WithGeoserver`
  / `gismanager.WithDatastore` / `gismanager.WithSource` / `gismanager.GetGISFiles`
  / `gismanager.DBIsAlive` / `gismanager.DBIsAliveContext`** (root) —
  replaced by the `publish.X` equivalents.
- **`gismanager.ConvertVector` / `gismanager.ConvertRaster` / `gismanager.ToCOG`
  / `gismanager.ReprojectRaster` / `gismanager.Rasterize` / `gismanager.BuildVRT`
  / `gismanager.DEMProcessing` / `gismanager.ToPMTiles`** + ~60 root-level
  `WithX*` helpers — replaced by the `convert.X` equivalents.
- **`gismanager.GetLogger()`** (root) — dropped from public API.
  v2 callers construct their own `*slog.Logger` (any handler — text,
  JSON, OTel-bridged) and pass it via `publish.WithLogger`. The
  internal default at `internal/slogx.Default` is package-private.
- **`(*Layer).GetGeomtryName()`** (PR #57) — typo'd duplicate of
  `GeometryName()`; behaviorally identical but with the typo
  preserved for v1 back-compat. v2 drops it.
- **`(*Layer).GetFeatures()`** (PR #57) — returned a slice of
  `*gdal.Feature` whose elements each owned a C-level handle the
  caller had to remember to `Destroy`; the slice form made leaks
  too easy. v2 callers use the `(*Layer).Features(ctx)` iterator
  instead, which destroys each feature as iteration advances and
  on early break.

### Tests

- `TestFeatures_Iterator` (publish) — replaces `TestGetFeatures`
  with the iterator-form contract (PR #57).
- `TestWithPublishConcurrency_*` (publish, 5 cases) — locks in
  the option's effect on the manager state across explicit-positive,
  zero-fallback, negative-fallback, n=1-serial, and unset paths
  (PR #58).
- The full integration suite (GeoServer 2.27.4 LTS + 2.28.0 stable,
  PostGIS 16) ran green at every Phase boundary.

### Operations

- **`release/v1.x` maintenance branch** (PR #52) — branched from
  `v1.4.1`. Patch releases on the v1.x line (security fixes,
  CVE patches) cut from this branch. CI workflows (`ci.yml`,
  `integration.yml`, `security.yml`) trigger on `release/v*` in
  addition to `master`.
- **Trivy SARIF aggregator race fix** (PR #56 follow-up) — both
  Trivy job uploads now share a single `category: trivy` to side-
  step the Code Scanning aggregator's "missing config" race that
  fired at workflow start (~2s) when an expected SARIF category
  hadn't yet uploaded.

### Migration

See [`MIGRATING.md`](./MIGRATING.md). The short version:

```bash
# 1. Bump every import.
find . -name '*.go' | xargs sed -i \
  's|"github.com/hishamkaram/gismanager"|"github.com/hishamkaram/gismanager/v2/publish"|g'
# (then split: convert/ for ConvertVector etc., errs/ for sentinels)

# 2. Rename Manager / Layer types.
find . -name '*.go' | xargs sed -i \
  's/\bgismanager\.ManagerConfig\b/publish.Manager/g; s/\bgismanager\.GdalLayer\b/publish.Layer/g'

# 3. Replace dropped methods.
#    GetGeomtryName → GeometryName
#    GetFeatures (slice) → Features(ctx) iterator (use range over the result)
```

## [1.4.1] — 2026-05-06

### Context

Infrastructure-only patch: fixes the release pipeline's binary-extract step that hung past the 60-min monitor window during the v1.4.0 cut. The image stage was unaffected (multi-arch + cosign-signed at ghcr.io published cleanly); only the raw amd64 binary tarball, checksums, and SBOM were missing from the v1.4.0 GitHub Release page. The library and CLI surfaces are byte-for-byte unchanged.

### Fixed

- **`release.yml` binary-extract step.** Replaced `docker buildx build --output type=local,dest=./dist-build .` (extracts the entire ~4 GB build-stage filesystem; hung the runner) with a two-step pattern: `docker/build-push-action@v6` with `platforms: linux/amd64` + `load: true` to hydrate the local Docker daemon, then `docker create` + `docker cp /usr/local/bin/<binary>` to extract just the three binaries (~10s). Validated end-to-end via the `v1.4.1-rc.1` tag — full release run completed in ~40 min vs. v1.4.0's >60 min hang. (PR #50)

## [1.4.0] — 2026-05-06

### Context

Polish, ecosystem fit, and operational maturity on top of v1.3.0. Seventeen focused PRs (#32–#48) sequenced for review-ability rather than dependency. Where v1.3 grew the conversion subsystem to feature-complete vs. the GDAL CLI surface, v1.4 turns the project from "library with a workable CLI" into "production-shippable v1.4-class": signed releases, security scans, runtime config validation, GeoParquet ingest, PMTiles export, OpenTelemetry-ready logging, and a parallel publish pipeline.

The v1.4 envelope was driven by three audits run before the cycle started — architecture, tests/CI/ops, and a 2026-ecosystem competitive scan — that produced a top-10 plan plus a backlog. All ten top-plan items shipped; six follow-on backlog items shipped on top of that. The remaining backlog (`runtime.AddCleanup` migration, integration `t.Parallel`, log-key OTel rename audit, STAC sidecar, Helm chart, etc.) is deferred to v1.5+.

No breaking changes to public API surface. Two behavioral enrichments (PublishAll now aggregates per-layer errors instead of returning nil; PublishAll dispatches the GeoServer publish step into a worker pool) are documented as strict information-additions — pre-v1.4 callers continue to work unchanged.

### Added

- **`(*ManagerConfig).Validate()`** — public method that aggregates every required-field check on the publish-pipeline contract (Geoserver / Datastore / Source) and returns a single `*GISError` wrapping `ErrConfigInvalid`. Callers using only `OpenSource` or the conversion functions can skip Validate; FromConfig calls it automatically. (PR #35)
- **`${VAR}` env-var interpolation in [FromConfig]** — every operator-supplied string field on `ManagerConfig` is run through `os.ExpandEnv` after YAML decode and before validation, so YAML files can reference secrets via env vars instead of inlining them. Datastore.Port (uint) is intentionally not expanded. (PR #35)
- **`cmd/internal/cli` helper package** — shared scaffolding for the three CLI binaries (gismanager / layerSchema / gisconvert): `SignalContext` (Ctrl-C / SIGTERM propagation), `PrintVersion` (stable `<binary> version=v.. commit=.. built=..` envelope, populated via `-ldflags` with runtime/debug.ReadBuildInfo fallback), `RequireFlag` (uniform missing-flag error envelope). (PR #34)
- **`-version` flag on every CLI binary**, populated at build time via the new `make build-cli` Makefile target which injects `git describe --tags`, `git rev-parse --short HEAD`, and `date -u`. (PR #34)
- **`-json` output mode in `cmd/layerSchema`** — emits a single top-level JSON array of `{path, name, fields[]}` objects (compact, suitable for `jq` / Terraform / Ansible). Empty-walk yields `[]` rather than `null`. (PR #44)
- **`ToPMTiles(ctx, src, dst, opts...) error`** — converts an existing MBTiles archive to a PMTiles v3 archive on disk, via [protomaps/go-pmtiles](https://github.com/protomaps/go-pmtiles). Stdlib `*log.Logger` of pmtiles.Convert is bridged through `slog.NewLogLogger` so progress lines flow through the manager's configured `*slog.Logger` handler. Two functional options: `WithPMTilesLogger`, `WithPMTilesDeduplicate`. Direct raster → PMTiles is deferred to v1.5; the v1.4 path is `ConvertRaster(... MBTILES) → ToPMTiles(...)`. (PR #42)
- **GeoParquet read+write support** — Dockerfile base swapped from `ghcr.io/osgeo/gdal:ubuntu-small-3.12.4` to `ubuntu-full-3.12.4` (digest-pinned: `sha256:5828162cffed3af3...`) bringing the Apache `Parquet` OGR driver into both the dev image and the published runtime image. New `parquetDriver = "Parquet"` constant; `.parquet` added to `supportedEXT` and routed in `GetDriver`. Trade-off: image size grew ~2× (~2 GB → ~4 GB); operators who don't need Parquet can re-pin `GDAL_BASE_DIGEST` back to ubuntu-small. (PR #40)
- **`errors.Join` aggregation in `(*ManagerConfig).PublishAll`** — every per-layer failure (walk error, PostGIS load error, GeoServer publish error) is collected and returned as `errors.Join(slice...)` from the final return. Pre-v1.4 callers that did `if err := mgr.PublishAll(ctx); err != nil` keep working — the change strictly enriches the error envelope. The previous "silently nil despite per-layer failures" path was an acknowledged bug (see the v1.3 doc comment "Future versions may aggregate per-layer errors"). (PR #43)
- **Bounded-concurrency GeoServer publish in `PublishAll`** — the publish step (HTTP-only; GeoServer queries PostGIS itself) now dispatches into a worker pool of 8 goroutines. Walk + LayerToPostgis stay serial because both touch the shared `*gdal.DataSource` and the underlying CGo handle is not reentrancy-safe across goroutines. Workspace + datastore are pre-warmed once before the pool fires to avoid the create-race that surfaces as 409 Conflict on workspace and 500 "already exists" on datastore (a GeoServer wire-format quirk). Stdlib `sync.WaitGroup` + buffered-channel semaphore — no new runtime dep. Default concurrency hardcoded; expose as functional option in v1.5 if there's demand to tune. (PR #48)
- **`make ci` umbrella + `make benchmark`** — `make ci` runs `lint + vet + test-unit + vuln` in sequence (~3 min) for fast preflight; `make benchmark` wires `go test -bench=. -benchmem -run='^$' ./...` so future Benchmark* functions are exercisable without remembering the magic invocation. (PR #45)
- **Tag-triggered release workflow** (`.github/workflows/release.yml`) — produces multi-arch Docker images (`linux/amd64` + `linux/arm64`) at `ghcr.io/<owner>/<repo>:<vX.Y.Z>` and `:latest`, cosign-signed keyless via the GitHub Actions OIDC identity; Linux amd64 binary tarball with `gismanager` + `layerSchema` + `gisconvert` (each carrying version metadata via `-ldflags`); SHA-256 checksums + cosign sign-blob; SBOM via `anchore/sbom-action` (syft, SPDX-JSON); auto-created GitHub Release with the matching CHANGELOG stanza as body. Triggers ONLY on `v*.*.*` tag push — Claude / CI never tag autonomously. New `RELEASING.md` documents the cut runbook + yank procedure. (PR #36)
- **Trivy security workflow** (`.github/workflows/security.yml`) — scans the runtime Docker image and the repository filesystem for HIGH/CRITICAL CVEs; findings flow into the GitHub Security tab as SARIF (one category per scan: `trivy-image`, `trivy-fs`). Triggers on push, PR, weekly Sunday 04:23 UTC, and `workflow_dispatch`. Does NOT fail the build on findings — the right pressure model for upstream CVEs is a Security-tab notification triggering an image rebase. (PR #46)
- **Coverage gate (`>= 65%`)** in `ci.yml` — fails the build if `go tool cover -func=coverage.out` total drops below 65%. Baseline at ship time: 69.3%. Raise the threshold whenever real coverage improves. (PR #37)
- **Integration log artifact upload on failure** — `integration.yml` now captures `docker compose logs` to `compose-logs-<geoserver-version>.txt` and uploads via `actions/upload-artifact@v4` (14-day retention). 200-line tail still prints inline for at-a-glance triage. (PR #37)
- **`.dockerignore`** — excludes `.git/`, `.github/`, `.devcontainer/`, `.claude/`, `testdata-fetched/`, compose YAMLs, `bin/` / `dist/`, editor / OS leftovers from the build context. Smaller context = faster context send + smaller buildkit cache layers. (PR #37)
- **Image digest pins** — `Dockerfile`'s `GDAL_BASE_DIGEST` and `docker/Dockerfile`'s `TOMCAT_BASE_DIGEST` ARGs pin manifest list digests; `FROM` lines reference them via `image@${digest}` instead of bare tag, immunizing the build from upstream re-tags. Multi-arch resolution still works (the pinned digests are the manifest list digests for `linux/amd64` + `linux/arm64`). (PR #37)
- **Apt-mirror connectivity probe** in `Dockerfile` — `curl --connect-timeout 5` checks `azure.archive.ubuntu.com` reachability before the apt-source rewrite. Local builds off the Azure network now automatically fall back to the default `archive.ubuntu.com` sources rather than hanging on a dead Azure-mirror connection. CI builds unaffected. (PR #40)
- **Seven runnable godoc `Example*` functions** in `example_convert_test.go` (`ExampleConvertVector` / `ExampleToCOG` / `ExampleReprojectRaster` / `ExampleRasterize` / `ExampleDEMProcessing` / `ExampleBuildVRT`, all checked via `// Output:`) and `example_manager_test.go` (`ExampleManagerConfig_PublishAll`, `ExampleManagerConfig_OpenSource`). Hermetic — every conversion example uses `/vsimem/` for synthetic sources and destinations; raster examples synthesize a tiny GeoTIFF in `/vsimem/` on the fly. pkg.go.dev now renders working code samples for every conversion entry point. (PR #38)
- **OpenTelemetry observability recipe** — new `examples/otel_pipeline/` (separate Go submodule, isolating the OTel SDK + OTLP exporter dependencies from the main module's intentionally small dep surface) wires `otelslog.NewLogger` + `otlploghttp.New` and runs a publish flow with trace correlation. New `docs/observability.md` covers plain slog, the OTel bridge, a Kubernetes deployment recipe, the current log-key conventions vs OTel semantic conventions, and performance notes (BatchProcessor vs SimpleProcessor). README gains an "Observability" section. (PR #41)

### Changed

- **`PublishAll`'s error semantics enriched** — see "Added" above (`errors.Join` aggregation). Strict information addition; pre-v1.4 callers' `err != nil` checks continue to work.
- **`PublishAll`'s concurrency model** — see "Added" above (bounded worker pool for the publish step). Publish-side ordering is now non-deterministic, but ordering was never part of the contract.
- **`ensureWorkspace` / `ensureDatastore`** — return `*GISError` directly with `Op="ensureWorkspace"` / `"ensureDatastore"` and `Sentinel=ErrGeoServerPublish` instead of bare `fmt.Errorf` with a stringly-typed prefix. The chain becomes `*GISError(Op=PublishGeoserverLayer) → *GISError(Op=helper) → *geoserver.APIError`, all walkable by `errors.Is` / `errors.As`. Public-facing `Op` stays `"PublishGeoserverLayer"`. (PR #33)
- **`ensureWorkspace` tolerates 409 Conflict** — concurrent callers can race the Get-not-found → Create pattern; the loser receives `ErrConflict` which the helper now treats as success. (PR #48)
- **Dockerfile runtime stage now ships gisconvert** — was previously missing despite the gisconvert binary shipping in v1.2. The ldflags now inject version metadata (`-X github.com/hishamkaram/gismanager/cmd/internal/cli.{Version,Commit,Date}`) into all three runtime binaries. (PR #36)

### Fixed

- **Conversion error-path Close hardening** — every conversion entry point (`ConvertVector`, `ConvertRaster`, `ReprojectRaster`, `Rasterize`, `BuildVRT`, `DEMProcessing`) now uses `defer out.Close()` after the success check instead of unconditional `out.Close()` at the success tail. Not a fix for an existing leak (the lukeroth/gdal binding returns `Dataset{}` with nil C handle on error, verified against the binding source) — the change earns its keep on **panic-safety** + **idiomaticity** (lint-clean per staticcheck SA5001) + **forward-proofing** future post-call code from silently dropping the close. (PR #32)

### Tests

- New `validate_test.go`: 7 tests / 11 sub-cases covering happy-path validation, every required-field omission, aggregated multi-failure envelope, env-var interpolation (set + unset), and `FromConfig` end-to-end with both `test_config_envvars.yml` and `test_config_missing.yml`. (PR #35)
- New `cmd/internal/cli/cli_test.go`: 3 tests covering `PrintVersion` (set + fallback paths), `RequireFlag` (table-driven), and `SignalContext` (SIGTERM cancellation via `syscall.Kill` on the test process). (PR #34)
- New `convert_vector_geoparquet_integration_test.go`: full GeoJSON ⇄ GeoParquet round-trip via `ConvertVector` with `WithVectorFormat("Parquet")` then `OpenSource` on the `.parquet` result. Asserts feature-count parity. (PR #40)
- New `convert_pmtiles_test.go`: 4 unit tests covering ctx-canceled fast-fail, missing-source error wrapping, config defaults + nil-logger guard, and error-message-contains-source-path. (PR #42)
- New `errors_test.go::TestGISError_NestedChain`: locks in the v1.4 nested-`*GISError` contract. (PR #33)
- New `errors_test.go::TestGISError_JoinedAggregation`: locks in the `errors.Join` chain semantics for PublishAll's aggregated return. (PR #43)
- New `vars_test.go`: 4 tests / 22 sub-cases covering `pgRegex` matches/non-matches, `supportedEXT` exact-set guard, and driver-name constants non-empty. (PR #47)
- New `cmd/gismanager/main_test.go` and `cmd/layerSchema/run_test.go`: 4 tests each, covering version flag short-circuit, missing-config flag, nonexistent-config-file, malformed-config-yaml. (PR #47)
- New `cmd/layerSchema/main_test.go::TestLayerEntryJSON_*`: locks in the on-the-wire JSON shape and the empty-array vs nil-slice distinction. (PR #44)

### Known limitations

- **`cmd/gisconvert -metadata-only` deferred** — implementation needs a vector/raster code-path split (the `gdal.Dataset` type returned by `gdal.OpenEx` exposes `LayerByIndex` but not `LayerCount`; only `gdal.DataSource` from `driver.Open` has the latter). Slated for v1.5+.
- **Direct concurrent `PublishGeoserverLayer` not goroutine-safe** — within `PublishAll`, the worker pool is safe because workspace + datastore are pre-warmed before goroutines fire. Callers invoking `PublishGeoserverLayer` directly across goroutines for the same workspace/datastore can still race the Create paths; only `ensureWorkspace`'s 409 tolerance covers part of this. The 500 "already exists" wire-format quirk on `ensureDatastore` is not yet handled. Slated for v1.5+ if there's demand.
- **`runtime.AddCleanup` migration deferred** — Go 1.25 has the new finalizer API but the GDAL handle lifecycle is well-served by explicit `Destroy()` calls today. Mechanical refactor for v1.5+.
- **OTel log-key rename audit deferred** — library log calls use Go-conventional keys (`path`, `src`, `workspace`, `err`) rather than OTel semantic conventions (`file.path`, `geoserver.workspace`, `db.name`). `docs/observability.md` documents the gap and shows an OTel collector processor recipe for renaming at ingest time. Library-side rename is non-breaking (log keys aren't public API) and is deferred to v1.5+.
- **Vector → PMTiles** is not yet supported (raster + tippecanoe-emitted MBTiles only). Vector-MBTiles emission via tippecanoe-style is out of scope for v1.4.
- **GeoServer 3.0 / Tomcat 11** unsupported — the v2-line GeoServer client + the Tomcat 9 base in `docker/Dockerfile` target the `javax.*` servlet namespace. GeoServer 3.0's move to `jakarta.*` is tracked separately for a future v1.x point release.
- **Direct raster → PMTiles** (skipping the intermediate MBTiles) deferred to v1.5+. The v1.4 path is `ConvertRaster(MBTILES) → ToPMTiles`.
- **`pgRegex` permits any single whitespace** before `PG:` (newline, tab, space) — `vars_test.go` documents this so a future "tighten to literal-space only" PR can flip the expectation deliberately.

## [1.3.0] — 2026-05-05

### Context

Three new conversion operations + one binding-gap closer on top of v1.2.0. Four focused PRs (#27, #28, #29, #30) sequenced by dependency. All changes are additive — no breaking changes, no deprecations.

The v1.2 series was about format conversion (file → file in a different shape). v1.3 broadens the conversion subsystem to cover the rest of GDAL's command-line workhorses: vector→raster (`gdal_rasterize`), multi-raster mosaic (`gdalbuildvrt`), and DEM analysis (`gdaldem`). Plus we close the silent-failure-on-unknown-driver gap documented in v1.2's known limitations.

### Added

- **`Rasterize(ctx, vectorSrc, rasterDst, opts...) error`** — vector → raster (the `gdal_rasterize` equivalent). Burn constant values via `WithRasterizeBurnValues` or per-feature attribute values via `WithRasterizeAttribute`; control output via `WithRasterizeFormat` / `WithRasterizeOutputType` / `WithRasterizeTargetResolution` / `WithRasterizeOutputSize` / `WithRasterizeOutputBounds` / `WithRasterizeLayer` / `WithRasterizeWhere` / `WithRasterizeCreationOption` / `WithRasterizeRawOptions`. Errors wrap `ErrConvertFailed` with `Op="Rasterize"`. (PR #27)
- **`BuildVRT(ctx, dst, srcs, opts...) error`** — multi-raster mosaic into a Virtual Raster (the `gdalbuildvrt` equivalent). Useful for assembling tile pyramids from many GeoTIFFs, or stacking single-band inputs into RGBA via `WithVRTSeparate`. Options: `WithVRTLogger` / `WithVRTResolution` (highest|lowest|average|user) / `WithVRTUserResolution` / `WithVRTSeparate` / `WithVRTAddAlpha` / `WithVRTResamplingAlg` / `WithVRTSrcNoData` / `WithVRTNoData` / `WithVRTHideNoData` / `WithVRTBands` / `WithVRTAllowProjectionDifference` / `WithVRTRawOptions`. Errors wrap `ErrConvertFailed` with `Op="BuildVRT"`. (PR #28)
- **`DEMProcessing(ctx, src, dst, mode, opts...) error`** — DEM raster analysis (the `gdaldem` equivalent). Supported modes: `hillshade`, `slope`, `aspect`, `color-relief` (requires `WithDEMColorFile`), `TRI`, `TPI`, `roughness`. Options: `WithDEMLogger` / `WithDEMFormat` / `WithDEMColorFile` / `WithDEMZFactor` / `WithDEMScale` / `WithDEMAzimuth` / `WithDEMAltitude` / `WithDEMCombined` / `WithDEMMultidirectional` / `WithDEMAlgorithm` (Horn|ZevenbergenThorne) / `WithDEMCreationOption` / `WithDEMOutputType` / `WithDEMRawOptions`. Empty mode and color-relief-without-color-file are pre-validated. Errors wrap `ErrConvertFailed` with `Op="DEMProcessing"`. (PR #29)
- **Driver pre-validation** for every conversion entry point that takes a format option (`ConvertVector` / `ConvertRaster` / `ToCOG` / `ReprojectRaster` / `Rasterize` / `DEMProcessing`). `WithVectorFormat` / `WithRasterFormat` / `WithRasterizeFormat` / `WithDEMFormat` values are now validated against the running GDAL driver registry via `gdal.GetDriverByName` before the C call. Unknown drivers surface as a clean `ErrConvertFailed` instead of GDAL's silent fail-with-stderr-warning behavior. Closes the silent-failure gap documented in v1.2's CHANGELOG known limitations. (PR #30)

### Changed

- **None.** All v1.3 additions are surface-additive — existing v1.x code paths are byte-for-byte unchanged.

### Tests

- 12-case table-driven unit test for `buildRasterizeArgs` option → `gdal_rasterize` arg mapping. (PR #27)
- 10-case table-driven unit test for `buildVRTArgs` option → `gdalbuildvrt` arg mapping. (PR #28)
- 9-case table-driven unit test for `buildDEMArgs` option → `gdaldem` arg mapping. (PR #29)
- ctx-canceled fast-fail and source-open error wrapping for all three new entry points.
- Empty-mode + color-relief-without-color-file pre-validation rejection for `DEMProcessing`. (PR #29)
- Empty-sources rejection for `BuildVRT`. (PR #28)
- 6-case unit test for the driver-registry validator. (PR #30)
- 4 integration-validating tests confirming each entry point's `Op` field on the error envelope when given a bogus format. (PR #30)
- Integration: Africa countries → 256×256 Byte GeoTIFF mask via `Rasterize` burn-value 1 with `-where` filter. (PR #27)
- Integration: country `POP_EST` attribute → 360×180 Float32 raster via `Rasterize`. (PR #27)
- Integration: two GeoTIFFs → mosaic VRT (default + `-separate` modes). (PR #28)
- Integration: `RGB.byte.tif` → hillshade / slope / TRI via `DEMProcessing`. (PR #29)

### Known limitations

- **No progress callbacks.** Same as v1.2 — `lukeroth/gdal`'s utility wrappers don't thread `pfnProgress` through the C functions. Long conversions are opaque from the Go side. Tracked for v1.4+ (needs an upstream binding patch).
- **No mid-conversion cancellation.** Same as v1.2 — `ctx` is honored at the function boundary, not inside the synchronous CGo call.
- **Other silent-failure modes still possible.** Driver-name pre-validation closes the most common case, but invalid CRS strings, malformed `-spat` extents, and similar option-side errors can still slip past `cerr=0`. Use the dev container's `ogr2ogr` / `gdal_translate` / `gdalwarp` / `gdal_rasterize` CLIs to validate inputs ahead of programmatic use if you need a hard guarantee.
- **No GeoParquet driver in the dev image.** The `ghcr.io/osgeo/gdal:ubuntu-small-3.12.4` base image doesn't include Apache Arrow / Parquet (`ubuntu-small` excludes the parquet driver). Swap to `ubuntu-full` if you need GeoParquet support; revisit in v1.4+.

## [1.2.0] — 2026-05-05

### Context

GDAL data-conversion subsystem on top of v1.1.0. Six focused PRs (#17, #19, #20, #21, #22, #23) sequenced by dependency. All changes are additive — no breaking changes, no deprecations. The conversion surface is **stateless**: top-level package functions, no `*ManagerConfig` required, mirroring the design choice that conversion is independent of GeoServer/PostGIS state.

### Added

- **`ConvertVector(ctx, src, dst, opts...) error`** — vector format conversion (GeoJSON ↔ GeoPackage ↔ Shapefile ↔ FlatGeobuf ↔ KML) plus reprojection, bbox clip, attribute filter, simplification, field-select, and layer rename. Thin wrapper around the C entry point behind `ogr2ogr` (`gdal.VectorTranslate`). (PR #19)
- **`ConvertRaster(ctx, src, dst, opts...) error`** — generic raster format conversion (GeoTIFF → COG, GeoTIFF → PNG, band-subset, output-window). Thin wrapper around the C entry point behind `gdal_translate` (`gdal.Translate`). (PR #20)
- **`ToCOG(ctx, src, dst, opts...) error`** — Cloud-Optimized GeoTIFF convenience wrapper. Pre-fills `WithRasterFormat("COG")` plus sane defaults (`COMPRESS=DEFLATE`, `BLOCKSIZE=512`, `OVERVIEW_RESAMPLING=NEAREST`); caller-supplied options override the defaults. (PR #20)
- **`ReprojectRaster(ctx, src, dst, srcSRS, dstSRS, opts...) error`** — raster reprojection (the `gdalwarp` equivalent). Cookie-cutter clipping via `WithRasterCutline` emits `-cutline` + `-cl` + `-crop_to_cutline`. (PR #21)
- **`WithVector*` option helpers** (11 helpers): `WithVectorLogger`, `WithVectorFormat`, `WithVectorSourceSRS`, `WithVectorTargetSRS`, `WithVectorBoundingBox`, `WithVectorWhere`, `WithVectorSimplify`, `WithVectorSelectFields`, `WithVectorLayerName`, `WithVectorOverwrite`, `WithVectorRawOptions`. Each maps 1:1 to an `ogr2ogr` CLI flag. (PR #19)
- **`WithRaster*` option helpers** (9 helpers): `WithRasterLogger`, `WithRasterFormat`, `WithRasterCreationOption`, `WithRasterOutputBounds`, `WithRasterBands`, `WithRasterResamplingAlg`, `WithRasterTargetResolution`, `WithRasterCutline`, `WithRasterRawOptions`. Each maps 1:1 to a `gdal_translate` / `gdalwarp` CLI flag. (PR #20, PR #21)
- **`ErrConvertFailed`** sentinel for the conversion subsystem. Wrap-recoverable via `errors.As` into `*GISError`; `GISError.Op` disambiguates the conversion entry point (`"ConvertVector"` / `"ConvertRaster"` / `"ReprojectRaster"`). (PR #19)
- **`cmd/gisconvert`** — CLI counterpart binary. Stdlib `flag` only — no new runtime deps. Vector mode covers all `WithVector*` options; raster mode covers all `WithRaster*` options including a `-cog` shortcut and reprojection trigger via `-s-srs` + `-t-srs`. (PR #23)
- **Manifest-driven `make fetch-testdata`.** Real-world geo fixtures (Natural Earth countries Shapefile + GeoJSON, rasterio `RGB.byte.tif`, rio-tiler `cog.tif`) are downloaded on demand into a gitignored sibling `testdata-fetched/` directory with sha256 verification. CI caches the fetched payload keyed on the manifest hash. (PR #17)
- **`docs/conversions.md`** — full conversion reference with vector + raster + reprojection examples and the cloud-I/O VFS matrix (`/vsis3/`, `/vsicurl/`, `/vsimem/`, `/vsizip/`, `/vsigs/`, `/vsiaz/`). (PR #22)
- **README "Conversion" section** — top-level discoverability pointing at `docs/conversions.md`. (PR #22)
- **`examples/convert_pipeline/`** — 30-line program exercising `ConvertVector` with reprojection + bbox clip + WHERE filter + simplification, validating the new API surface from a caller's perspective.

### Changed

- **None.** All v1.2 additions are surface-additive — existing v1.1.x code paths are byte-for-byte unchanged.

### Tests

- 13-case table-driven unit test for `buildVectorTranslateArgs` option → `ogr2ogr` arg mapping. (PR #19)
- 11-case table-driven unit test for `buildTranslateArgs` option → `gdal_translate` arg mapping. (PR #20)
- 7-case table-driven unit test for `buildWarpArgs` option → `gdalwarp` arg mapping. (PR #21)
- ctx-canceled fast-fail and source-open error wrapping for all four conversion entry points.
- `/vsimem/` destination unit test for `ConvertVector` — exercises the cloud-aware I/O without network or filesystem write. (PR #22)
- Integration: Shapefile (via `/vsizip/`) → GeoPackage with feature-count parity. (PR #19)
- Integration: GeoJSON → GeoPackage with reproject (4326 → 3857) + bbox clip + WHERE filter + EPSG:3857 SRS verification. (PR #19)
- Integration: GeoTIFF → COG with COG driver + overview-count assertion. (PR #20)
- Integration: GeoTIFF → PNG with band selection. (PR #20)
- Integration: `RGB.byte.tif` (EPSG:32618) → EPSG:3857 reprojection with bilinear resampling. (PR #21)
- 9-case unit suite for `cmd/gisconvert` flag → option mapping (`parseBBox`, `parseBands`, `parseRes`, repeated `-co`, missing-arg rejection, raster-reproject SRS validation). (PR #23)

### Known limitations

- **No progress callbacks.** `lukeroth/gdal`'s utility wrappers don't thread `pfnProgress` through `VectorTranslate`/`Warp`/`Translate`. Long conversions are opaque from the Go side. Tracked for v1.3 (needs an upstream binding patch).
- **No mid-conversion cancellation.** `ctx` is honored at the function boundary (before `OpenEx`); the underlying CGo call is synchronous and uninterruptible.
- **Silent failure on unknown drivers.** When `WithVectorFormat` / `WithRasterFormat` names a driver GDAL doesn't have, the C-level option parsing fails with stderr noise but `cerr=0`, so the Go wrapper returns `nil`. Pre-validate driver names on the caller side for a hard guarantee.
- **Bare `.zip` Shapefile bundles do not auto-prefix.** Pass `/vsizip/<path>` explicitly, or pre-extract via `GetGISFiles`.
- **No GeoParquet, no PMTiles.** GeoParquet driver in GDAL 3.8+ but ecosystem is still settling — revisit in v1.3. PMTiles needs Tippecanoe (separate dependency, out of GDAL scope).

## [1.1.0] — 2026-05-04

### Context

Maintainability + structure improvements on top of v1.0.0. Five focused PRs (#11 → #15) sequenced by dependency. All changes are additive on the public surface; three soft deprecations (`GetFeatures`, `DBIsAlive`, `GetGeomtryName`) — bodies kept verbatim, callers warned via `// Deprecated:` to migrate at their own pace. Removal is a v2 break.

### Added

- **Functional-options constructor.** `gismanager.New(opts ...Option) (*ManagerConfig, error)` + `WithLogger` / `WithGeoserver` / `WithDatastore` / `WithSource` helpers. Programmatic callers can now build a manager without a YAML file and inject a custom `*slog.Logger`. `FromConfig(yamlPath)` keeps working — internally delegates to `New` after the YAML decode. `WithLogger(nil)` falls back to the default `GetLogger()` so a zero-arg `New()` is usable. (PR #11)
- **`(*ManagerConfig).NewLayer(*gdal.Layer) *GdalLayer`.** Stamps the manager's logger onto the wrapper so per-manager logger configuration reaches helper methods like `GetLayerSchema`, `GetFeatures`, etc. The zero-value `GdalLayer{Layer: l}` form is still supported — methods that need a logger fall back to `GetLogger()` when the field is nil. (PR #12)
- **`(*ManagerConfig).Walk(ctx) iter.Seq2[WalkItem, error]`.** Streams `(file, layer)` pairs over every supported GIS file under `Source.Path`. Files-outer / layers-inner ordering. Per-file failures yield a non-nil `error` so callers can `continue`; the iterator never aborts the whole walk on one bad file. Bounds memory to one file's worth of OGR state by `defer source.Destroy()`-ing each `*gdal.DataSource` before moving on. Honors `ctx.Err()` between files and between layers. (PR #13)
- **`(*ManagerConfig).PublishAll(ctx) error`.** Convenience wrapping `Walk + LayerToPostgis + PublishGeoserverLayer` — the body of `cmd/gismanager`'s main loop, now reusable from any caller. Opens the PostGIS datastore once at the top and re-uses it across every file. Per-layer failures are logged via the manager's logger; only setup failures surface as a returned error. (PR #13)
- **`(*GdalLayer).Features(ctx) iter.Seq[gdal.Feature]`.** Streaming iterator that destroys each `gdal.Feature` as iteration advances and on early `break` (no caller-side cleanup contract). Honors `ctx.Err()` between feature reads. (PR #14)
- **`DBIsAliveContext(ctx, dbType, conn)`.** Ctx-aware variant of `DBIsAlive`. The original `DBIsAlive` now delegates to this with `context.Background()`. (PR #14)
- **`(*GdalLayer).GeometryName()`** — properly-spelled sibling of the deprecated `GetGeomtryName()`. Identical behaviour. (PR #15)

### Changed

- **CLIs simplified.** `cmd/gismanager` is now ~10 lines (calls `manager.PublishAll(ctx)`); `cmd/layerSchema` is ~20 lines (iterates `manager.Walk(ctx)`). Both lost the duplicated walk-files → open-source → iterate-layers loop they reimplemented. (PR #13)
- **`Walk` / `PublishAll` `defer source.Destroy()` automatically.** Per-file CGo handles are released between iterations; long-running pipelines no longer leak `*gdal.DataSource` handles. Direct callers of `OpenSource` remain responsible for their own `defer source.Destroy()` — documented in README + `docs/architecture.md`. (PRs #13 + #14)

### Deprecated

- **`(*GdalLayer).GetFeatures()`.** Returns `[]*gdal.Feature` whose handles must be `Destroy()`-ed by the caller, easy to forget. Use `Features(ctx)` instead.
- **`DBIsAlive(dbType, conn)`.** Hard-codes `context.Background()`. Use `DBIsAliveContext` instead.
- **`(*GdalLayer).GetGeomtryName()`.** Typo (missing 'e' in "Geometry"). Use `GeometryName()`.

All three keep their bodies for v1.x back-compat; removal is a v2 break.

### Fixed

- **KML files were silently dropped from directory walks** because `supportedEXT` had `"kml"` (no leading dot); `filepath.Ext()` always returns the dot, so the entry never matched. Bug present since the project's first commit (2018). Fixed: entry is now `".kml"`. A `testdata/sample.kml` fixture was added so directory-walk tests exercise the path. Two new guard tests (`TestSupportedEXT_AllHaveLeadingDot`, `TestSupportedEXT_IncludesKML`) prevent regression. (PR #15)

### Removed

- **Unused unexported constants `openFileGDBDriver` and `esriJSONDriver`** in `vars.go`. They were never referenced from `GetDriver`'s switch. Unexported, no external API impact. (PR #15)

### Tests

- New table-driven `driver_table_test.go` for `GetDriver` (~22 rows): every supported extension (lower / upper / mixed case), every `pgRegex`-matching PostgreSQL connection-string variant, unsupported categories (raster, csv, xml, no-extension, empty path, double extension), and the `.gdb`-listed-but-unrouted edge case.
- New `walk_test.go`: Walk yields layers from testdata, early-break safe, `context.Cancel` honored, missing source surfaces error item, KML fixture appears in output.
- New `lifecycle_test.go`: `Features` zero-value safety + ctx.Cancel handling, `DBIsAliveContext` fail-fast under canceled ctx, `DBIsAlive` back-compat.
- New `layer_logger_test.go`: `NewLayer` stamps logger, custom-logger emission lands in caller-controlled buffer, `getGISFiles` forwards manager logger to `preprocessFile`.
- New `options_test.go`: every `WithX`, `WithLogger(nil)` fallback, parity-with-FromConfig, last-write-wins ordering, nil-`Option` tolerated.

### Known limitations (carried from v1.0.0)

- **GDAL CGo dep** — runtime image is ~500 MB, dynamically linked against `libgdal.so.38`.
- **No streaming publish.** Layers are materialized into PostGIS before publishing.
- **`*gdal.DataSource` / `gdal.OGRDriver` / `*geoserver.Client` / `gdal.Feature` still leak on the public surface.** Callers must import `lukeroth/gdal` and `geoserver/v2`. Hiding these behind project-owned types is a v2 facade-pattern break.

## [1.0.0] — 2026-05-04

### Context

First-ever stable release. Revives a 7-year-stale repo whose build had been broken on Go 1.13+ since 2022 (issue #1). The codebase has been rewritten end-to-end on top of the stabilized [`hishamkaram/geoserver/v2`](https://github.com/hishamkaram/geoserver) client. Everything below was delivered across 5 sequenced PRs (#2 → #6) on `master`.

### Added

- **Docker-only dev environment.** Multi-stage `Dockerfile` based on `ghcr.io/osgeo/gdal:ubuntu-small-3.12.4` with three targets: `dev` (Go 1.25.9 + golangci-lint v2.12.1 + govulncheck + goimports + libgdal headers), `build` (compile binaries from source), `runtime` (binaries + libgdal at ~500 MB). `docker-compose.yml` for the dev shell. `.devcontainer/devcontainer.json` for VS Code. No GDAL on the host.
- **Go modules.** `go.mod` (Go 1.25 floor) + `go.sum`. Replaces `dep`'s `Gopkg.toml` / `Gopkg.lock` (abandoned 2020).
- **`internal/zipx` package.** Stdlib-only zip extractor with zip-slip rejection (CVE-2018-1002200 class) and a 2 GiB per-entry size cap. Replaces the deprecated `mholt/archiver` dependency.
- **`*GISError` typed error + 7 sentinels.** `ErrConfigInvalid`, `ErrUnsupportedFormat`, `ErrInvalidLayer`, `ErrInvalidDatasource`, `ErrPostGISConnect`, `ErrGeoServerPublish`, `ErrNoSourcesFound`. Match by sentinel via `errors.Is`; recover wrapped `*geoserver.APIError` via `errors.As`. The `*GISError.Error()` format is `gismanager: <Op> <Source>: <sentinel>: <cause>`.
- **`*slog.Logger` throughout.** Default `gismanager.GetLogger()` returns a stderr text-handler logger. Internal call sites use structured key/value logging.
- **Idempotent publish flow.** `PublishGeoserverLayer` checks workspace, datastore, and feature-type existence via `Get` + `errors.Is(err, geoserver.ErrNotFound)` before creating; running the same publish twice is a no-op rather than a 409 Conflict.
- **GitHub Actions CI.** `Lint` (golangci-lint v2.12.1), `Unit tests (Go 1.25)`, `govulncheck`, `Analyze (Go)` (CodeQL), all running in container `ghcr.io/osgeo/gdal:ubuntu-small-3.12.4`.
- **Integration test stack.** `docker-compose.test.yml` boots GeoServer (parameterized 2.27.4 LTS or 2.28.0 stable) + PostGIS 16 + a `test-runner` service on the same network. `publish_integration_test.go` (`//go:build integration`) covers end-to-end publish, idempotency, unsupported-format sentinel surfacing, and the nil-datasource / nil-layer error paths from the pre-revival suite.
- **Integration matrix workflow.** `.github/workflows/integration.yml` runs the integration suite against both supported GeoServer versions on every PR; both legs must pass before merge.
- **Project docs.** README rewritten product-first; new `docs/architecture.md`, `docs/version-compat.md`. Issue templates, PR template, CODEOWNERS, Dependabot config.
- **Project Claude Code config** under `.claude/`: `go-reviewer`, `gdal-reviewer`, `integration-runner` agents; `gismanager-quirks` and `docker-only-dev` skills; `/integration-test` and `/lint-fix` slash commands. `CLAUDE.md` at the repo root documents the conventions.

### Changed

- **`OpenSource` signature** — was `(path, access) (*gdal.DataSource, bool)` (the redundant `ok bool` v1.0 idiom); now `(ctx, path, access) (*gdal.DataSource, error)`. Errors surface `ErrUnsupportedFormat` / `ErrInvalidDatasource`.
- **`PublishGeoserverLayer` signature** — was `(*GdalLayer) (bool, error)`; now `(ctx, *GdalLayer) error`. Idempotent end-to-end.
- **`GetGeoserverCatalog` signature** — was `() *GeoServer` (v1 monolithic client); now `() (*geoserver.Client, error)` returning a `geoserver/v2` sub-client client.
- **Datastore connection shape** — replaced the v1 `gsconfig.DatastoreConnection` with v2's typed `datastores.PostGIS`.
- **Logging migration** — `logrus` → `*slog.Logger` direct. `GetLogger` now returns `*slog.Logger`.
- **CLIs no longer panic.** `cmd/gismanager` and `cmd/layerSchema` now have a `func main() { if err := run(ctx); err != nil { slog.Error(...); os.Exit(1) } }` shape. Missing config / load failures produce one-line structured errors instead of goroutine stack dumps.
- **`make fmt`** — actually works (`goimports` is now installed in the dev image; was missing from the v1.0.0 RC).

### Removed

- **`mholt/archiver` dependency.** The upstream package was deprecated in 2024. gismanager only used `archiver.Zip.Open` for shapefile-zip preprocessing; `internal/zipx` replaces it.
- **`logrus` dependency.** Replaced with stdlib `*slog.Logger`.
- **`Gopkg.toml`, `Gopkg.lock`.** The `dep` dependency manager was abandoned in 2020.
- **`.travis.yml`.** Travis CI hasn't been viable for OSS since 2021.
- **`gopkg.in/yaml.v2`.** Bumped to `gopkg.in/yaml.v3`.
- **`v1` `hishamkaram/geoserver` client import.** Migrated to `hishamkaram/geoserver/v2` v2.0.0+.

### Fixed

- **Issue #1** (Compile Issues, Jan 2022): the `lukeroth/gdal` API drift (`Layer.Feature(int)` → `Layer.Feature(int64)`) and the `mholt/archiver.Zip.Open` value→pointer receiver change. `go get -v github.com/hishamkaram/gismanager` works on Go 1.25 again.
- **Library-code panics removed.** Library code (root + `internal/`) contains zero `panic(` calls; cmd/ binaries also panic-free as of v1.0.0.

### Security

- **govulncheck clean** as of v1.0.0. The Go 1.25.3 stdlib CVEs flagged during PR 3 (GO-2026-4869 / 4865 / 4603 etc.) are all fixed by the pinned 1.25.9 toolchain.
- **Zip-slip protection** added to `internal/zipx` via the canonical `filepath.Rel` defense pattern (also recognized by CodeQL's `go/zipslip` query).

### Known limitations

- **GDAL CGo dep** — the runtime image is ~500 MB and dynamically linked against `libgdal.so.38`. Pure-Go alternatives (`twpayne/go-shapefile`, `paulmach/orb`) cover fewer formats than GDAL; the trade-off is documented in [`docs/architecture.md`](docs/architecture.md).
- **No streaming publish.** Each layer is materialized into PostGIS before publishing; very large datasets need to fit memory at OGR-copy time.
- **No functional-options constructor yet.** `gismanager.FromConfig(yamlPath)` is the only public constructor; a `gismanager.New(opts ...Option)` form will land in v1.1.x.
