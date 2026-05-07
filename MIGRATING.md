# Migrating from v1.x to v2

This guide covers every breaking change between
`github.com/hishamkaram/gismanager` (v1.x) and
`github.com/hishamkaram/gismanager/v2` (v2.0+). The library API
surface is the same shape it had in v1.4 — the v2 cycle was
**structural** (subpackage layout, module-path bump, two deprecated
methods removed, one new option). No behavior changes to
documented contracts; the conversion entry points and the publish
pipeline both produce byte-for-byte identical output.

If you can't migrate yet, the v1.x line stays maintained on the
`release/v1.x` branch (security fixes, CVE patches). v2 is master.

---

## TL;DR — three sed scripts

```bash
# 1. Bump module path on imports.
find . -name '*.go' | xargs sed -i \
  's|"github.com/hishamkaram/gismanager"|"github.com/hishamkaram/gismanager/v2"|g'

# 2. The bumped imports point at the empty root package. Split them
#    into subpackage imports based on which symbol you used:
#      gismanager.ConvertVector / ToCOG / ToPMTiles / WithVector*
#                              → import .../v2/convert
#      gismanager.New / ManagerConfig / Walk / PublishAll / etc.
#                              → import .../v2/publish
#      gismanager.GISError / Err*  → import .../v2/errs

# 3. Apply the type renames.
find . -name '*.go' | xargs sed -i \
  -e 's/\bgismanager\.ManagerConfig\b/publish.Manager/g' \
  -e 's/\bgismanager\.GdalLayer\b/publish.Layer/g' \
  -e 's/\bgismanager\.VectorConvertOption\b/convert.VectorOption/g' \
  -e 's/\bgismanager\.RasterConvertOption\b/convert.RasterOption/g' \
  -e 's/\bgismanager\.\(ConvertVector\|ConvertRaster\|ToCOG\|ReprojectRaster\|Rasterize\|BuildVRT\|DEMProcessing\|ToPMTiles\)\b/convert.\1/g' \
  -e 's/\bgismanager\.With\(Vector\|Raster\|Rasterize\|VRT\|DEM\|PMTiles\)/convert.With\1/g' \
  -e 's/\bgismanager\.\(New\|FromConfig\|WithLogger\|WithGeoserver\|WithDatastore\|WithSource\|GetGISFiles\|DBIsAlive\|DBIsAliveContext\)\b/publish.\1/g' \
  -e 's/\bgismanager\.\(GISError\|Err[A-Z][a-zA-Z]*\)\b/errs.\1/g'
```

This handles the mechanical part. Edge cases below.

---

## Module path

| v1.x | v2 |
|------|----|
| `github.com/hishamkaram/gismanager` | `github.com/hishamkaram/gismanager/v2` |

Tag form: `v1.4.1` (v1 line, on `release/v1.x` branch) vs.
`v2.0.0+` (v2 line, on `master`).

`go get`:

```bash
# v1 stayer
go get github.com/hishamkaram/gismanager@latest  # latest v1.x

# v2 migrator
go get github.com/hishamkaram/gismanager/v2@v2.0.0
```

Both lines coexist in `go.mod`. You can have *one* dependency on each
during a transitional period; just don't mix the two within the same
caller file (Go will resolve them as distinct types).

---

## Subpackage split

The root `gismanager` package is now empty (only `doc.go`). Public
API lives in three subpackages:

| Subpackage | Purpose |
|------------|---------|
| `github.com/hishamkaram/gismanager/v2/publish` | `Manager`, `Layer`, walk + load + publish pipeline |
| `github.com/hishamkaram/gismanager/v2/convert` | Stateless GDAL CLI-equivalents + `ToPMTiles` |
| `github.com/hishamkaram/gismanager/v2/errs` | Typed `*GISError` + `Err*` sentinels |

`internal/slogx` (the default-logger helper) stays internal — v2
callers construct their own `*slog.Logger`.

---

## Type renames

| v1 | v2 | Reason |
|----|----|--------|
| `gismanager.ManagerConfig` | `publish.Manager` | "Config" was a misnomer — the type IS the manager. |
| `gismanager.GdalLayer` | `publish.Layer` | Drop the redundant `Gdal` prefix (clear from package context). |
| `gismanager.VectorConvertOption` | `convert.VectorOption` | Drop the redundant `Convert` prefix in subpackage scope. |
| `gismanager.RasterConvertOption` | `convert.RasterOption` | Same. |

`gismanager.RasterizeOption` / `VRTOption` / `DEMOption` /
`PMTilesOption` were already concise; v2 just adds the `convert.`
prefix.

`gismanager.GeoserverConfig` / `SourceConfig` / `DatastoreConfig` /
`Option` / `WalkItem` / `LayerField` keep their names; only the
package prefix changes (`publish.X`).

`gismanager.GISError` keeps its name; package prefix becomes
`errs.GISError`.

---

## Function/method moves

### Conversion subsystem (8 entry points + ~60 helpers)

```
gismanager.ConvertVector      → convert.ConvertVector
gismanager.ConvertRaster      → convert.ConvertRaster
gismanager.ToCOG              → convert.ToCOG
gismanager.ReprojectRaster    → convert.ReprojectRaster
gismanager.Rasterize          → convert.Rasterize
gismanager.BuildVRT           → convert.BuildVRT
gismanager.DEMProcessing      → convert.DEMProcessing
gismanager.ToPMTiles          → convert.ToPMTiles

gismanager.WithVector*        → convert.WithVector*
gismanager.WithRaster*        → convert.WithRaster*
gismanager.WithRasterize*     → convert.WithRasterize*
gismanager.WithVRT*           → convert.WithVRT*
gismanager.WithDEM*           → convert.WithDEM*
gismanager.WithPMTiles*       → convert.WithPMTiles*
```

### Publish pipeline

```
gismanager.New                → publish.New
gismanager.FromConfig         → publish.FromConfig
gismanager.WithLogger         → publish.WithLogger
gismanager.WithGeoserver      → publish.WithGeoserver
gismanager.WithDatastore      → publish.WithDatastore
gismanager.WithSource         → publish.WithSource
gismanager.GetGISFiles        → publish.GetGISFiles
gismanager.DBIsAlive          → publish.DBIsAlive
gismanager.DBIsAliveContext   → publish.DBIsAliveContext
```

Methods on `publish.Manager` and `publish.Layer` are unchanged
shape-wise — they just live on the renamed types.

### Errors

```
gismanager.GISError           → errs.GISError
gismanager.ErrConfigInvalid   → errs.ErrConfigInvalid
gismanager.ErrUnsupportedFormat → errs.ErrUnsupportedFormat
gismanager.ErrInvalidLayer    → errs.ErrInvalidLayer
gismanager.ErrInvalidDatasource → errs.ErrInvalidDatasource
gismanager.ErrPostGISConnect  → errs.ErrPostGISConnect
gismanager.ErrGeoServerPublish → errs.ErrGeoServerPublish
gismanager.ErrNoSourcesFound  → errs.ErrNoSourcesFound
gismanager.ErrConvertFailed   → errs.ErrConvertFailed
```

`errors.Is` / `errors.As` semantics are preserved across the
boundary — the underlying error instances are the same.

---

## Removed methods

### `(*Layer).GetGeomtryName()`

Typo'd duplicate of `GeometryName()`. Same behavior; v1 kept it for
back-compat. v2 drops it.

```go
// v1
name := layer.GetGeomtryName()

// v2
name := layer.GeometryName()
```

### `(*Layer).GetFeatures()`

Returned `[]*gdal.Feature`; each element owned a C-level handle that
the caller had to remember to `Destroy`. Use the iterator instead:

```go
// v1 — leaks if you forget to Destroy
features := layer.GetFeatures()
for _, f := range features {
    use(f)
    f.Destroy()  // easy to forget
}

// v2 — no Destroy needed; iterator handles it
for f := range layer.Features(ctx) {
    use(f)
}
```

The iterator destroys each feature as the for-range advances **and**
on early `break`. The `ctx` parameter lets you stop iteration
mid-way (the iterator yields nothing after `ctx.Done()`).

### Root `gismanager.GetLogger()`

Dropped. v2 callers construct their own `*slog.Logger` (any handler
— text, JSON, OTel-bridged via the
[otelslog bridge](https://pkg.go.dev/go.opentelemetry.io/contrib/bridges/otelslog))
and pass it via `publish.WithLogger`.

```go
// v1
mgr, _ := gismanager.New(gismanager.WithLogger(gismanager.GetLogger()))

// v2
import "log/slog"
import "os"

mgr, _ := publish.New(publish.WithLogger(
    slog.New(slog.NewJSONHandler(os.Stderr, nil)),
))

// Or pass nil to opt into the package default:
mgr, _ := publish.New(publish.WithLogger(nil))
```

`publish.WithLogger(nil)` is equivalent to v1's
`gismanager.WithLogger(gismanager.GetLogger())` — the package
internally falls back to `internal/slogx.Default()`.

---

## New in v2

### `publish.WithPublishConcurrency(n int)`

Caps how many GeoServer feature-type creations
`(*Manager).PublishAll` dispatches in parallel. Walk +
`LayerToPostgis` stay serial regardless (CGo handle thread-safety).

```go
mgr, _ := publish.New(
    publish.WithPublishConcurrency(16),  // default is 8
    // ... other options ...
)
```

| Arg | Effect |
|-----|--------|
| `n <= 0` | Fall back to package default (currently 8). |
| `n > 0` | Cap parallel publish at `n`. |
| `n == 1` | Strictly serial publish. |

---

## CLI binaries

The three CLI binaries (`gismanager`, `layerSchema`, `gisconvert`)
are unchanged in their CLI surface — same flags, same outputs.
Only the package paths inside their source updated to `/v2/...`.

If you build them yourself from source (rather than pulling the
v2.0.0 GitHub Release tarball or the `ghcr.io/hishamkaram/gismanager:v2.0.0`
Docker image), the `make build-cli` target now injects v2 ldflags
automatically.

---

## Common patterns

### Pattern: cross-package use (publish + convert)

In v1, both lived at the root, so there was no cross-package import:

```go
// v1
import "github.com/hishamkaram/gismanager"

mgr, _ := gismanager.New(/* ... */)
_ = gismanager.ConvertVector(ctx, src, dst, gismanager.WithVectorFormat("GPKG"))
```

In v2, they're separate:

```go
// v2
import (
    "github.com/hishamkaram/gismanager/v2/convert"
    "github.com/hishamkaram/gismanager/v2/publish"
)

mgr, _ := publish.New(/* ... */)
_ = convert.ConvertVector(ctx, src, dst, convert.WithVectorFormat("GPKG"))
```

### Pattern: error matching

`errors.Is` semantics are preserved — the v2 sentinels are the same
underlying error instances the v1 sentinels aliased to.

```go
// v1
if errors.Is(err, gismanager.ErrGeoServerPublish) { ... }

// v2
if errors.Is(err, errs.ErrGeoServerPublish) { ... }
```

`errors.As` to the typed envelope:

```go
// v1
var ge *gismanager.GISError
if errors.As(err, &ge) { fmt.Println(ge.Op, ge.Source) }

// v2
var ge *errs.GISError
if errors.As(err, &ge) { fmt.Println(ge.Op, ge.Source) }
```

The `Op`, `Source`, `Sentinel`, `Cause` fields are unchanged.

### Pattern: PublishAll error aggregation

The error contract from v1.4 (`errors.Join` of per-layer failures)
is preserved in v2:

```go
err := mgr.PublishAll(ctx)
if err != nil {
    if errors.Is(err, errs.ErrGeoServerPublish) {
        // at least one layer failed at the publish step
    }
    // Enumerate every per-layer failure:
    var u interface{ Unwrap() []error }
    if errors.As(err, &u) {
        for _, e := range u.Unwrap() {
            log.Printf("per-layer failure: %v", e)
        }
    }
}
```

---

## What didn't change

- The CLI flags, environment variables, config YAML schema.
- The Docker image entry points (`ghcr.io/hishamkaram/gismanager:v2.0.0`
  has the same `gismanager` / `layerSchema` / `gisconvert` binaries at
  the same `/usr/local/bin/` paths).
- The minimum supported Go version (1.25.0).
- The pinned GDAL base (`ghcr.io/osgeo/gdal:ubuntu-full-3.12.4`,
  digest-pinned).
- The supported GeoServer versions (2.27.4 LTS + 2.28.0 stable).
- The dependency surface: `lukeroth/gdal`, `lib/pq`,
  `hishamkaram/geoserver/v2`, `gopkg.in/yaml.v3`, `protomaps/go-pmtiles`,
  stdlib. No new runtime deps.

---

## Where to ask

- v2 issues: open against `master`.
- v1.x patch requests: open against `release/v1.x` (the maintenance
  branch documented in `RELEASING.md`).
