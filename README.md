[![Go Reference](https://pkg.go.dev/badge/github.com/hishamkaram/gismanager.svg)](https://pkg.go.dev/github.com/hishamkaram/gismanager)
[![Go Report Card](https://goreportcard.com/badge/github.com/hishamkaram/gismanager)](https://goreportcard.com/report/github.com/hishamkaram/gismanager)
[![GitHub license](https://img.shields.io/github/license/hishamkaram/gismanager.svg)](https://github.com/hishamkaram/gismanager/blob/master/LICENSE)
[![GitHub issues](https://img.shields.io/github/issues/hishamkaram/gismanager.svg)](https://github.com/hishamkaram/gismanager/issues)

# GIS Manager

**A Go tool that publishes GIS vector data to GeoServer via PostGIS.** Drop a directory of shapefiles, GeoJSON, GeoPackage, or KML files into a config, run one command, and gismanager loads each file into a PostGIS database and registers the resulting tables as GeoServer feature types — workspace creation, datastore registration, and layer publishing all idempotent.

It exists because the path from "I have a shapefile" to "I have a published GeoServer layer" otherwise involves three different tools (ogr2ogr, psql, and the GeoServer admin UI or REST API). gismanager is one CLI plus one config.

## Contents

- [What this tool does](#what-this-tool-does)
- [Install](#install)
- [Quick start](#quick-start)
- [Worked example: publish a directory of GIS files](#worked-example-publish-a-directory-of-gis-files)
- [Conversion](#conversion)
- [Errors](#errors)
- [Configuration](#configuration)
- [Library use](#library-use)
- [Version compatibility](#version-compatibility)
- [Contributing](#contributing)
- [License](#license)

## What this tool does

Three CLI binaries plus a Go library covering two GIS workflows:

- **`gismanager`** — full publish pipeline. Walks `source.path`, picks every supported GIS file, copies its layers into PostGIS via OGR's PostgreSQL driver, then publishes each PostGIS table as a GeoServer feature type. Idempotent: workspace, datastore, and feature-type creation all check existence first via the `geoserver/v2` client's `Get` + `errors.Is(err, geoserver.ErrNotFound)` idiom.
- **`layerSchema`** — read-only schema inspector. Walks `source.path`, opens each GIS file via GDAL, and prints the geometry column + attribute fields for every layer it finds. No PostGIS, no GeoServer.
- **`gisconvert`** *(v1.2+)* — data-conversion CLI: vector format conversion (`ogr2ogr` equivalent), raster format conversion (`gdal_translate` equivalent), Cloud-Optimized GeoTIFF generation, and raster reprojection (`gdalwarp` equivalent). See [Conversion](#conversion).
- **`package gismanager`** — both flows as a Go library:
  - **Publish**: construct a `*ManagerConfig` via `gismanager.New(opts ...Option)` (functional-options) or the YAML-driven `gismanager.FromConfig(yamlPath)`, then call `Walk` / `PublishAll` or the lower-level `OpenSource` / `LayerToPostgis` / `PublishGeoserverLayer` primitives.
  - **Convert** *(v1.2+)*: stateless `ConvertVector` / `ConvertRaster` / `ToCOG` / `ReprojectRaster` package functions. No `*ManagerConfig` required.

### Supported source formats

The two surfaces have different reach.

#### Publish pipeline (`cmd/gismanager`, `Walk`, `PublishAll`)

Driven by an explicit extension allowlist in [`vars.go::supportedEXT`](vars.go) — only these extensions are picked up by directory walks:

| Extension | OGR driver |
|---|---|
| `.shp`, `.zip` (zipped shapefile bundle) | ESRI Shapefile |
| `.geojson`, `.json` | GeoJSON |
| `.gpkg` | GeoPackage |
| `.kml` | KML |

Zipped shapefile bundles are auto-extracted into a temp directory before the OGR open — see [`internal/zipx`](internal/zipx) for the stdlib `archive/zip`-based extractor (zip-slip rejection, 2 GiB per-entry cap). Adding a new extension to the publish pipeline means adding a row to `supportedEXT` and a switch case in [`manager.go::GetDriver`](manager.go).

#### Conversion entry points *(v1.2+, `cmd/gisconvert`, `ConvertVector` / `ConvertRaster` / `ToCOG` / `ReprojectRaster`)*

No allowlist — the conversion functions delegate straight to GDAL's full driver registry via `gdal.OpenEx`. Whatever the underlying GDAL build can read or write, the conversion entry points accept. In the project's dev image (`ghcr.io/osgeo/gdal:ubuntu-small-3.12.4`) that includes the obvious workhorses:

- **Vector** (OGR): Shapefile, GeoPackage, GeoJSON, FlatGeobuf, KML, GML, GPX, MapInfo TAB/MIF, CSV (with WKT/lon-lat geometry), DGN, S-57, PostgreSQL/PostGIS, plus the cloud-native paths via VFS (`/vsis3/`, `/vsicurl/`, `/vsizip/`, `/vsimem/`, …).
- **Raster** (GDAL): GeoTIFF, Cloud-Optimized GeoTIFF, JP2/OpenJPEG, PNG, JPEG, NITF, HFA (Erdas .img), VRT, plus the same VFS paths.

For an authoritative list against any GDAL build, run `ogrinfo --formats` (vector) and `gdalinfo --formats` (raster) inside the dev container — driver compile-time inclusion varies across GDAL builds, and the dev image is the supported baseline.

## Install

**This project does NOT install GDAL on the host machine.** All build, test, and run work happens inside the Docker dev image (`ghcr.io/osgeo/gdal:ubuntu-small-3.12.4` base + Go 1.25.9 + tooling). If you don't have Docker, [install Docker first](https://docs.docker.com/get-docker/).

```bash
git clone https://github.com/hishamkaram/gismanager
cd gismanager
make dev          # opens an interactive bash inside the container
# OR run targets non-interactively:
make build        # go build ./...
make test-unit    # unit tests (no live GeoServer)
make image        # produce a runtime image: gismanager:local
```

The runtime image (`make image`) is a multi-stage build that ships only the binaries + libgdal at `~500 MB`. It's what you'd publish to a registry for production-ish use.

## Quick start

The shortest path to a published layer assumes you have a GeoServer + PostGIS already running somewhere reachable. If you don't, the project's own integration test stack is a working example — `make compose-test-up` boots GeoServer 2.28.0 + PostGIS 16 in two containers.

1. Write a config file. Example `my-config.yaml`:

   ```yaml
   geoserver:
     url: http://localhost:8080/geoserver
     username: admin
     password: geoserver
     workspace: my_workspace
   datastore:
     host: localhost
     port: 5432
     database: gis
     username: golang
     password: golang
     name: my_postgis_store
   source:
     path: ./testdata
   ```

2. Run the CLI from the runtime image:

   ```bash
   docker run --rm --network host \
     -v "$PWD/my-config.yaml:/cfg.yaml:ro" \
     -v "$PWD/testdata:/testdata:ro" \
     gismanager:local --config /cfg.yaml
   ```

   gismanager scans `/testdata`, loads each supported file into PostGIS, and publishes the resulting tables as feature types in the `my_workspace` workspace.

## Worked example: publish a directory of GIS files

The repo's own `testdata/` directory has a GeoJSON file and a GeoPackage. Booting the integration stack and publishing them end-to-end:

```bash
# Boot GeoServer 2.28.0 + PostGIS 16 in containers.
make compose-test-up

# Run the CLI inside the test-runner container so it can see the
# in-network GeoServer + PostGIS by service name.
docker compose -f docker-compose.test.yml run --rm test-runner \
  bash -c 'go run ./cmd/gismanager --config testdata/test_config.yml'

# Verify via the GeoServer REST API.
curl -fsS -u admin:geoserver \
  http://localhost:8080/geoserver/rest/workspaces/golang/datastores/gismanager_data/featuretypes.json | jq

# Tear down.
make compose-test-down
```

Same idea programmatically (the integration suite is a worked example — see [`publish_integration_test.go`](publish_integration_test.go) `TestPublishGeoJSON_EndToEnd_Integration`).

## Conversion

Beyond the publish pipeline, gismanager ships a stateless conversion subsystem that mirrors GDAL's command-line workhorses:

| Function | Wraps | Use case | Since |
|---|---|---|---|
| `ConvertVector` | `ogr2ogr` | vector format conversion + reproject + bbox clip + attribute filter + simplify | v1.2 |
| `ConvertRaster` | `gdal_translate` | raster format conversion + band subset + output window | v1.2 |
| `ToCOG` | `gdal_translate -of COG` | Cloud-Optimized GeoTIFF with sane defaults pre-applied | v1.2 |
| `ReprojectRaster` | `gdalwarp` | raster reprojection + cookie-cutter clip via cutline | v1.2 |
| `Rasterize` | `gdal_rasterize` | vector → raster: burn polygons into a mask, or attributes into a continuous field | v1.3 |
| `BuildVRT` | `gdalbuildvrt` | mosaic many GeoTIFFs into a Virtual Raster (tile pyramid prep, RGBA stacking) | v1.3 |
| `DEMProcessing` | `gdaldem` | DEM analysis: hillshade, slope, aspect, color-relief, TRI, TPI, roughness | v1.3 |

All seven are top-level package functions — no `*ManagerConfig` required — and accept `/vsi*/`-prefixed paths transparently (`/vsis3/`, `/vsicurl/`, `/vsimem/`, `/vsizip/`, `/vsigs/`, `/vsiaz/`). Driver names supplied via the `With*Format` helpers are pre-validated against the running GDAL build (since v1.3) — an unknown driver surfaces as a clean `ErrConvertFailed` instead of GDAL's silent fail-with-stderr-warning behavior.

```go
// Vector: 4326 GeoJSON → 3857 GeoPackage, clipped to Africa, simplified.
err := gismanager.ConvertVector(ctx, "world.geojson", "africa.gpkg",
    gismanager.WithVectorFormat("GPKG"),
    gismanager.WithVectorOverwrite(),
    gismanager.WithVectorTargetSRS("EPSG:3857"),
    gismanager.WithVectorBoundingBox(-25, -40, 60, 40),
    gismanager.WithVectorWhere("CONTINENT = 'Africa'"),
    gismanager.WithVectorSimplify(100),
)

// Raster: GeoTIFF → COG with the canonical defaults.
err = gismanager.ToCOG(ctx, "scene.tif", "scene.cog.tif")

// Raster reprojection: UTM → Web Mercator.
err = gismanager.ReprojectRaster(ctx, "utm.tif", "wm.tif",
    "EPSG:32618", "EPSG:3857",
    gismanager.WithRasterResamplingAlg("bilinear"),
)
```

Full reference, more examples, and the cloud-I/O matrix: [`docs/conversions.md`](docs/conversions.md).

## Errors

Every error gismanager returns is a `*GISError` wrapping a sentinel from [`errors.go`](errors.go). Match by sentinel:

```go
import "github.com/hishamkaram/gismanager"

err := mgr.PublishGeoserverLayer(ctx, layer)
switch {
case errors.Is(err, gismanager.ErrPostGISConnect):
    // PostGIS unreachable; bring it up and retry.
case errors.Is(err, gismanager.ErrGeoServerPublish):
    // Workspace/datastore/feature-type creation failed; the wrapped
    // *geoserver.APIError carries the HTTP status (404, 409, 500…).
    var apiErr *geoserver.APIError
    if errors.As(err, &apiErr) {
        log.Printf("status=%d body=%s", apiErr.StatusCode, apiErr.Body)
    }
case errors.Is(err, gismanager.ErrUnsupportedFormat):
    // The path's extension isn't one of the dispatched OGR drivers.
}
```

Sentinels: `ErrConfigInvalid`, `ErrUnsupportedFormat`, `ErrInvalidLayer`, `ErrInvalidDatasource`, `ErrPostGISConnect`, `ErrGeoServerPublish`, `ErrNoSourcesFound`. Never compare error strings — `errors.Is(err, sentinel)` is the only correct test.

## Observability

The library emits structured logs via `*slog.Logger` throughout the publish
pipeline and the conversion subsystem. Pass a custom logger via
`WithLogger` to control formatting, filtering, and routing.

For production deploys with trace correlation (every library log gets the
active span's `trace_id` / `span_id`), use the
[OpenTelemetry slog bridge](https://pkg.go.dev/go.opentelemetry.io/contrib/bridges/otelslog).
A runnable end-to-end example lives at
[`examples/otel_pipeline/`](./examples/otel_pipeline) (its own Go submodule
so the OTel SDK doesn't bloat gismanager's runtime dep surface). See
[`docs/observability.md`](./docs/observability.md) for the architectural
pattern and a Kubernetes deployment recipe.

## Configuration

YAML schema (the same struct gets used by `gismanager.FromConfig`):

```yaml
geoserver:
  url: <string>          # GeoServer base URL, e.g. http://geoserver:8080/geoserver
  username: <string>     # admin user
  password: <string>
  workspace: <string>    # workspace gismanager publishes into; created on first use

datastore:               # connection params for the PostGIS database
  host: <string>
  port: <uint>
  database: <string>     # database name (created externally; gismanager does not provision databases)
  username: <string>
  password: <string>
  name: <string>         # GeoServer datastore name (must be unique within the workspace)

source:
  path: <string>         # directory or single file containing GIS data
```

A working config used by the integration suite lives at [`testdata/test_config.yml`](testdata/test_config.yml).

## Library use

### Publish pipeline

```go
import (
    "context"
    "github.com/hishamkaram/gismanager"
)

func main() {
    mgr, err := gismanager.FromConfig("my-config.yaml")
    if err != nil { /* ... */ }

    ctx := context.Background()

    // Convenience: walk + load + publish in one call.
    if err := mgr.PublishAll(ctx); err != nil { /* ... */ }

    // Or stream-iterate manually for finer control:
    for item, err := range mgr.Walk(ctx) {
        if err != nil { continue }
        // item.Layer is logger-stamped via NewLayer; use it freely.
        _ = item.Layer
    }

    // Or call the low-level primitives directly (note the defer to
    // release CGo handles — Walk / PublishAll do this automatically):
    target, err := mgr.OpenSource(ctx, mgr.Datastore.BuildConnectionString(), 1)
    if err != nil { /* ... */ }
    defer target.Destroy()

    src, err := mgr.OpenSource(ctx, "/path/to/file.geojson", 0)
    if err != nil { /* ... */ }
    defer src.Destroy()

    for i := 0; i < src.LayerCount(); i++ {
        layer := src.LayerByIndex(i)
        wrapped := mgr.NewLayer(&layer)
        newLayer, err := wrapped.LayerToPostgis(target, mgr, true)
        if err != nil || newLayer == nil { continue }
        _ = mgr.PublishGeoserverLayer(ctx, newLayer)
    }
}
```

> **Resource lifecycle:** `*gdal.DataSource` returned by `OpenSource` owns a CGo-side handle that must be released via `.Destroy()` to avoid leaks. `Walk` and `PublishAll` do this automatically (`defer source.Destroy()` per file). Direct callers of `OpenSource` are responsible for calling `Destroy()` themselves — the binding has no `Close()` method; `Destroy()` is the canonical release primitive in `lukeroth/gdal`.

### Conversion *(v1.2+)*

The conversion entry points are top-level package functions — no `*ManagerConfig` required, no GeoServer/PostGIS state held.

```go
// Reproject + clip + filter + simplify in one call.
err := gismanager.ConvertVector(ctx, "world.geojson", "africa.gpkg",
    gismanager.WithVectorFormat("GPKG"),
    gismanager.WithVectorOverwrite(),
    gismanager.WithVectorTargetSRS("EPSG:3857"),
    gismanager.WithVectorBoundingBox(-25, -40, 60, 40),
    gismanager.WithVectorWhere("CONTINENT = 'Africa'"),
    gismanager.WithVectorSimplify(100),
)

// Cloud-Optimized GeoTIFF with sane defaults.
err = gismanager.ToCOG(ctx, "scene.tif", "scene.cog.tif")

// UTM → Web Mercator with bilinear resampling.
err = gismanager.ReprojectRaster(ctx, "utm.tif", "wm.tif",
    "EPSG:32618", "EPSG:3857",
    gismanager.WithRasterResamplingAlg("bilinear"),
)

// v1.3+: vector → raster (burn an attribute into a continuous Float32 field).
err = gismanager.Rasterize(ctx, "countries.geojson", "pop.tif",
    gismanager.WithRasterizeFormat("GTiff"),
    gismanager.WithRasterizeOutputType("Float32"),
    gismanager.WithRasterizeAttribute("POP_EST"),
    gismanager.WithRasterizeOutputSize(360, 180),
)

// v1.3+: stack many GeoTIFFs into a single VRT for downstream Warp/Translate.
err = gismanager.BuildVRT(ctx, "mosaic.vrt",
    []string{"tile1.tif", "tile2.tif", "tile3.tif"},
    gismanager.WithVRTResolution("highest"),
)

// v1.3+: hillshade from a DEM.
err = gismanager.DEMProcessing(ctx, "dem.tif", "dem.hs.tif", "hillshade",
    gismanager.WithDEMAzimuth(315),
    gismanager.WithDEMAltitude(45),
)
```

A complete 30-line program lives in [`examples/convert_pipeline/main.go`](examples/convert_pipeline/main.go); full reference and the cloud-I/O VFS matrix in [`docs/conversions.md`](docs/conversions.md).

## Version compatibility

gismanager v1.0.x targets:

- **Go 1.25+** — required by the transitive `geoserver/v2` dependency.
- **GeoServer 2.27 LTS + 2.28 stable** — both validated by the matrix integration leg in CI on every PR.
- **PostGIS 16-3.4** — the integration stack pins this; older PostGIS (≥ 2.5) should work but isn't gated in CI.
- **GDAL 3.12.x** — pinned via the `ghcr.io/osgeo/gdal:ubuntu-small-3.12.4` base image. Bumping requires re-running the integration suite.

GeoServer 3.0 (Tomcat 11 / Jakarta EE / ImageN) is parked for a future v1.x point release after the upstream migration settles.

Full matrix: [`docs/version-compat.md`](docs/version-compat.md).

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md). Highlights:

- All work happens inside the Docker dev image — no host GDAL install.
- Branch from `master`; squash-merge with `--delete-branch`. Never push directly to `master`.
- Both unit and integration tests are mandatory on every PR. The matrix CI runs against GeoServer 2.27.4 + 2.28.0; both legs must pass.
- Conventional Commits.

## License

[MIT](LICENSE) © Hesham Karm.
