# Architecture

This document describes how `github.com/hishamkaram/gismanager` is laid out, the design tenets, and the trade-offs behind v1.0's shape. Reference material; release notes live in [`../CHANGELOG.md`](../CHANGELOG.md).

## Big picture

gismanager is a small composition of three things:

```
                 ┌─────────────────────────┐
   GIS files ───▶│   GDAL OGR readers      │  (lukeroth/gdal CGo bindings)
   (shp, gpkg,   │  (file ingest + schema  │
    geojson, kml,│   introspection)        │
    zip)         └────────────┬────────────┘
                              │
                              ▼
                 ┌─────────────────────────┐
                 │   OGR PostgreSQL driver │  (CopyLayer over the OGR API)
                 │   → PostGIS table       │
                 └────────────┬────────────┘
                              │
                              ▼
                 ┌─────────────────────────┐
                 │  geoserver/v2 client    │  (REST: workspace, datastore,
                 │  → published feature    │   feature type)
                 │     type                │
                 └─────────────────────────┘
```

The package wires the three together with a `*publish.Manager` that holds the GeoServer endpoint, the PostGIS connection params, and the source-directory path. Everything else is a thin orchestration layer over GDAL + the upstream Go client.

## Package shape

| Path | Purpose |
|---|---|
| `github.com/hishamkaram/gismanager` | Public API — `*publish.Manager`, `*publish.Layer`, `*GISError`, sentinels, public method receivers |
| `github.com/hishamkaram/gismanager/internal/zipx` | stdlib `archive/zip` extractor with zip-slip rejection + per-entry size cap. Used to auto-extract zipped shapefile bundles before the OGR open. Internal — not importable |
| `github.com/hishamkaram/gismanager/cmd/gismanager` | Full publish-pipeline CLI |
| `github.com/hishamkaram/gismanager/cmd/layerSchema` | Read-only schema-printing CLI |
| `github.com/hishamkaram/gismanager/cmd/gisconvert` | v1.2+ conversion CLI (`ogr2ogr` / `gdal_translate` / `gdalwarp` / COG); v1.3 entry points (`Rasterize` / `BuildVRT` / `DEMProcessing`) are exposed via the library only |
| `github.com/hishamkaram/gismanager/examples/convert_pipeline` | 30-line worked example of `ConvertVector` with reproject + clip + filter |

A single Go module at the repo root. No `/v2` semantic-import-versioning suffix because gismanager has no released versions to preserve — v1.0.0 is the first stable tag.

## Public API

The lead type is `*publish.Manager`, constructed via `publish.New(opts ...Option)` (functional options — `WithLogger`, `WithGeoserver`, `WithDatastore`, `WithSource`) or the YAML-driven `publish.FromConfig(path)`. Methods:

```go
// Build a v2 GeoServer client from the configured endpoint + credentials.
func (m *publish.Manager) GetGeoserverCatalog() (*geoserver.Client, error)

// Open a GIS data source via GDAL. ctx is reserved for cancellation.
func (m *publish.Manager) OpenSource(ctx context.Context, path string, access int) (*gdal.DataSource, error)

// Map a path or connection string to its OGR driver.
func (m *publish.Manager) GetDriver(path string) (gdal.OGRDriver, error)

// Copy this GDAL layer into a PostGIS-backed OGR data source.
func (l *publish.Layer) LayerToPostgis(targetSource *gdal.DataSource, m *publish.Manager, overwrite bool) (*publish.Layer, error)

// Publish a PostGIS-backed layer as a GeoServer feature type. Idempotent
// end-to-end (Get + ErrNotFound for workspace, datastore, and feature type).
func (m *publish.Manager) PublishGeoserverLayer(ctx context.Context, layer *publish.Layer) error

// High-level pipeline: walk → load → publish, with per-file source.Destroy().
// Iteration is files-outer / layers-inner.
func (m *publish.Manager) Walk(ctx context.Context) iter.Seq2[WalkItem, error]
func (m *publish.Manager) PublishAll(ctx context.Context) error

// Streaming feature iterator that Destroys each gdal.Feature as iteration
// advances; replaces the deprecated GetFeatures() []*gdal.Feature shape.
func (l *publish.Layer) Features(ctx context.Context) iter.Seq[gdal.Feature]
```

### Conversion subsystem (v1.2+ / v1.3+)

A second, **stateless** surface alongside the publish pipeline. No `*publish.Manager` required — these are top-level package functions that wrap GDAL's command-line workhorses:

```go
// v1.2 — format conversion family
func ConvertVector(ctx, src, dst string, opts ...VectorConvertOption) error          // ogr2ogr
func ConvertRaster(ctx, src, dst string, opts ...RasterConvertOption) error          // gdal_translate
func ToCOG(ctx, src, dst string, opts ...RasterConvertOption) error                  // gdal_translate -of COG
func ReprojectRaster(ctx, src, dst, srcSRS, dstSRS string, opts ...RasterConvertOption) error  // gdalwarp

// v1.3 — analysis / mosaic / vector→raster
func Rasterize(ctx, vectorSrc, rasterDst string, opts ...RasterizeOption) error      // gdal_rasterize
func BuildVRT(ctx, dst string, srcs []string, opts ...VRTOption) error               // gdalbuildvrt
func DEMProcessing(ctx, src, dst, mode string, opts ...DEMOption) error              // gdaldem
```

Options use a modality prefix (`WithVector*` / `WithRaster*` / `WithRasterize*` / `WithVRT*` / `WithDEM*`) to avoid name collisions in the same package. Each helper renders to a single CLI flag — the `build<Op>Args` renderers are unit-tested separately so the mapping is locked in without needing CGo. Errors are wrapped with `ErrConvertFailed`; the `GISError.Op` field disambiguates the entry point.

Driver names supplied via the `With<Op>Format` helpers are pre-validated against the running GDAL build (`gdal.GetDriverByName`) before the C call — an unknown driver surfaces as a clean `ErrConvertFailed` envelope rather than the upstream silent fail-with-stderr-warning behavior. Added in v1.3.

All conversion entry points pass paths transparently to GDAL, so any `/vsi*/` prefix (`/vsis3/`, `/vsicurl/`, `/vsimem/`, `/vsizip/`, `/vsigs/`, `/vsiaz/`) works without special handling. Full reference: [`conversions.md`](conversions.md).

All methods that can fail return errors wrapping a sentinel from [`../errs/errs.go`](../errs/errs.go) — see the README's [Errors](../README.md#errors) section for the matching idiom.

### Resource lifecycle

`*gdal.DataSource` (returned by [`OpenSource`](../publish/manager.go)) and `gdal.Feature` (yielded by [`Features`](../publish/layer.go)) own CGo-side handles that must be released via `.Destroy()`. The binding does not have a `Close()` method — `Destroy()` is the canonical release primitive in `lukeroth/gdal`.

The library's high-level helpers handle this automatically:

- `Walk` / `PublishAll` `defer source.Destroy()` after each file before moving on, so per-file CGo handles never leak even on early `break` of the for-range loop.
- `Features` destroys each `gdal.Feature` as iteration advances, and the deferred destroy still fires when the for-range loop exits early.

**Direct callers** of `OpenSource` are responsible for their own `defer source.Destroy()`. The README's "Library use" section shows the pattern.

The deprecated `GetFeatures()` and `DBIsAlive()` are kept for back-compat with v1.0 callers but flagged with `// Deprecated:`. New code should use `Features(ctx)` and `DBIsAliveContext(ctx, ...)`.

## Context-first

Every exported method that does I/O takes `ctx context.Context` as its first argument. There are no `*Context` twin methods — that pattern was an upstream `geoserver` v1 source-compat idiom that v2 dropped, and gismanager mirrors v2's discipline. Callers without a context pass `context.Background()` at the call site.

Today the OGR / GDAL bindings don't accept context (no upstream cancellation surface), so gismanager threads ctx through its own boundary and uses it for the geoserver/v2 calls. When upstream grows ctx propagation, swapping it in is a non-breaking change.

## Errors

`errs` package defines 7 sentinels (`ErrConfigInvalid`, `ErrUnsupportedFormat`, `ErrInvalidLayer`, `ErrInvalidDatasource`, `ErrPostGISConnect`, `ErrGeoServerPublish`, `ErrNoSourcesFound`) and a typed `*GISError` with `Op`, `Source`, `Sentinel`, `Cause` fields. `Unwrap` returns Cause; `Is` matches Sentinel.

Why both layers: callers can branch by category (`errors.Is(err, ErrPostGISConnect)`) for control flow, *and* extract the underlying `*geoserver.APIError` (or driver-level error) via `errors.As` for diagnostic logging. The same error satisfies both at once.

The wrapped `*geoserver.APIError` from the upstream v2 client carries HTTP status + body preview, so a 409 Conflict from a duplicate datastore looks like:

```
gismanager: PublishGeoserverLayer "myws/mystore": gismanager: geoserver publish: geoserver: Datastores.Create POST http://...: 409 Conflict: Store 'mystore' already exists in workspace 'myws'
```

## Logging

`*slog.Logger` directly. No wrapper. The default `slogx.Default()` returns `slog.New(slog.NewTextHandler(os.Stderr, nil))`; production callers should construct their own `*slog.Logger` (any handler — JSON, lumberjack rotation, otel) and pass it on `publish.Manager.logger`.

Internal call sites use structured logging — `logger.Error("publish feature type", "workspace", ws, "datastore", ds, "layer", name, "err", err)`. The library emits Debug/Warn/Error only; `cmd/*` may emit Info on successful publish events.

## Concurrency

`*publish.Manager` is intended to be read-only after construction. The `logger` field is `*slog.Logger` (concurrency-safe by stdlib contract). The other fields are configuration values that never change.

The geoserver/v2 client constructed by `GetGeoserverCatalog` is itself concurrency-safe (the upstream contract — verified in that project's CI under `-race`). gismanager itself doesn't introduce shared mutable state.

The CGo GDAL bindings have their own thread-safety constraints — review `lukeroth/gdal` upstream docs before parallelizing GDAL operations across goroutines. The current synchronous flow (open → copy → publish, one layer at a time) is well within the bindings' safe envelope.

## Docker-only build

CGo + GDAL means the build needs system GDAL headers and a C toolchain. Rather than ask every contributor to install GDAL on their host, gismanager pins to `ghcr.io/osgeo/gdal:ubuntu-small-3.12.4` and shells every Go invocation into a dev container.

The dev image (built via `docker compose build dev`) installs Go 1.25.9 + golangci-lint v2.12.1 + govulncheck + goimports on top of the OSGeo image. The `Makefile` shells every target through `docker compose run --rm -T dev <command>`.

A `.devcontainer/devcontainer.json` lets VS Code (and other devcontainer-aware editors) open the source tree inside the container; that's how `gopls` resolves the GDAL CGo headers (`#include <gdal.h>`) without any host install.

**Critical Dockerfile note:** do NOT `apt-get install libgdal-dev` on top of the OSGeo image. Ubuntu 24's apt ships GDAL 3.6 (`libgdal.so.34`); installing it alongside the OSGeo-bundled GDAL 3.12 (`libgdal.so.38`) produces binaries linked against the older soname, which fail to load at runtime. The Dockerfile's apt step deliberately omits `libgdal-dev`; the OSGeo image already provides headers at `/usr/include/gdal_*.h` + `gdal.pc`.

## Test split

Two layers, distinguished by file name + build tag:

| Layer | Files | Build tag | What it does |
|---|---|---|---|
| Unit | `*_test.go` (no `_integration_` infix) | none | Self-contained: GDAL + testdata fixtures, no network. `make test-unit` runs them in <2s. |
| Integration | `*_integration_test.go` | `//go:build integration` | Real GeoServer + PostGIS via `docker-compose.test.yml`. Exercises end-to-end publish flow plus the nil-check error paths that need a live PostGIS to reach (DBIsAlive runs first). `make test-integration` boots the stack first. |

Both layers are mandatory on every PR. CI runs unit on Go 1.25 and integration on **GeoServer 2.27.4 LTS + 2.28.0 stable** — all three legs (Lint + Unit + Integration matrix) must go green.

## v2 geoserver client integration

gismanager publishes via [`github.com/hishamkaram/geoserver/v2`](https://github.com/hishamkaram/geoserver). Mapping of gismanager calls to v2 surface:

| gismanager step | v2 client call |
|---|---|
| Build client | `geoserver.New(url, geoserver.WithBasicAuth(user, pass))` |
| Workspace exists? | `c.Workspaces.Get(ctx, name)` + `errors.Is(err, geoserver.ErrNotFound)` |
| Create workspace | `c.Workspaces.Create(ctx, &workspaces.Workspace{Name: name})` |
| Datastore exists? | `c.Datastores.InWorkspace(ws).Get(ctx, name)` + ErrNotFound |
| Create PostGIS datastore | `c.Datastores.InWorkspace(ws).Create(ctx, datastores.PostGIS{...})` |
| Feature type exists? | `c.FeatureTypes.InWorkspace(ws).InDatastore(ds).Get(ctx, name)` + ErrNotFound |
| Publish feature type | `c.FeatureTypes.InWorkspace(ws).InDatastore(ds).Create(ctx, &featuretypes.FeatureType{...})` |

The Get + ErrNotFound idiom is what makes the publish flow idempotent end-to-end — see [`../publish/publish_integration_test.go`](../publish/publish_integration_test.go) `TestPublishGeoJSON_Idempotent_Integration`.

## Cross-references

- [`../README.md`](../README.md) — install, quick start, worked example, errors, config schema
- [`../CHANGELOG.md`](../CHANGELOG.md) — release notes
- [`../CONTRIBUTING.md`](../CONTRIBUTING.md) — Docker-only dev workflow, PR conventions
- [`version-compat.md`](version-compat.md) — Go × GeoServer × GDAL × PostGIS matrix
- [`../publish/publish_integration_test.go`](../publish/publish_integration_test.go) — worked end-to-end example as a test
