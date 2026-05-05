# Changelog

All notable changes to `github.com/hishamkaram/gismanager` are documented here. Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/); the project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **`ConvertRaster(ctx, src, dst, opts...) error`** — generic raster format conversion (GeoTIFF → COG, GeoTIFF → PNG, band-subset, output-window). Thin wrapper around the C entry point behind `gdal_translate` (`gdal.Translate`). (PR 3 of v1.2)
- **`ToCOG(ctx, src, dst, opts...) error`** — convenience wrapper that pre-fills `WithRasterFormat("COG")` plus sane defaults (`COMPRESS=DEFLATE`, `BLOCKSIZE=512`, `OVERVIEW_RESAMPLING=NEAREST`). Caller-supplied options override defaults — pass `WithRasterCreationOption("COMPRESS", "ZSTD")` to swap codecs. (PR 3 of v1.2)
- **`WithRaster*` option helpers**: `WithRasterLogger`, `WithRasterFormat`, `WithRasterCreationOption`, `WithRasterOutputBounds`, `WithRasterBands`, `WithRasterResamplingAlg`, `WithRasterTargetResolution`, `WithRasterCutline`, `WithRasterRawOptions`. Each maps 1:1 to a `gdal_translate` / `gdalwarp` CLI flag. The cutline option is wired only on the warp-side (PR 4); `gdal_translate` does not support cutlines. (PR 3 of v1.2)
- **`ConvertVector(ctx, src, dst, opts...) error`** — vector format conversion (GeoJSON ↔ GeoPackage ↔ Shapefile ↔ FlatGeobuf ↔ KML) plus reprojection, bbox clip, attribute filter, simplification, field-select, and layer rename. Thin wrapper around the C entry point behind `ogr2ogr` (`gdal.VectorTranslate`). Stateless — does NOT require a `*ManagerConfig`. (PR 2 of v1.2)
- **`WithVector*` option helpers**: `WithVectorLogger`, `WithVectorFormat`, `WithVectorSourceSRS`, `WithVectorTargetSRS`, `WithVectorBoundingBox`, `WithVectorWhere`, `WithVectorSimplify`, `WithVectorSelectFields`, `WithVectorLayerName`, `WithVectorOverwrite`, `WithVectorRawOptions`. Each maps 1:1 to a `ogr2ogr` CLI flag. (PR 2 of v1.2)
- **`ErrConvertFailed`** sentinel for the conversion subsystem. Wrap-recoverable via `errors.As` into `*GISError`; `GISError.Op` disambiguates the conversion entry point. (PR 2 of v1.2)
- **Manifest-driven `make fetch-testdata`.** Real-world geo fixtures (Natural Earth countries, rasterio `RGB.byte.tif`, rio-tiler `cog.tif`, plus more in subsequent PRs) are downloaded into the gitignored sibling `testdata-fetched/` directory on demand, with sha256 verification on each fetch. CI caches the fetched payload keyed on the manifest hash, so cold-runs pay ~3 s of network and warm-runs are zero-network. Tracked: `testdata/manifest.sha256`, `testdata/LICENSES.md`, `testdata/README.md`, `scripts/fetch-testdata.sh`, `testdata-fetched/.gitignore`. Untracked: the binaries themselves. The fetched fixtures live in a sibling directory (not under `testdata/`) so the existing `TestGetGISFiles` unit test and `TestPublishAll_EndToEnd` integration test — both of which walk `./testdata/` — are unaffected. (PR 1 of v1.2)

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
