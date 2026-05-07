# Conversions

`gismanager` v1.2 ships a small, stateless conversion subsystem covering
the four most common GDAL/OGR operations:

- **`ConvertVector`** — vector format conversion (the `ogr2ogr` equivalent)
- **`ConvertRaster`** — raster format conversion (the `gdal_translate` equivalent)
- **`ToCOG`** — Cloud-Optimized GeoTIFF generation (a thin wrapper over `ConvertRaster`)
- **`ReprojectRaster`** — raster reprojection (the `gdalwarp` equivalent)

Each is a top-level package function; none requires a `*publish.Manager`.
Configuration goes through functional options (`WithVector*` or
`WithRaster*` modality prefix to avoid name collisions).

## Vector conversion

### Shapefile → GeoPackage

```go
err := convert.ConvertVector(ctx,
    "/vsizip//data/countries.zip", // /vsizip/ for zipped Shapefile bundles
    "/data/countries.gpkg",
    convert.WithVectorFormat("GPKG"),
    convert.WithVectorOverwrite(),
)
```

### GeoJSON → GeoPackage with reprojection, bbox clip, and attribute filter

```go
err := convert.ConvertVector(ctx,
    "/data/world.geojson",
    "/data/africa_3857.gpkg",
    convert.WithVectorFormat("GPKG"),
    convert.WithVectorOverwrite(),
    convert.WithVectorTargetSRS("EPSG:3857"),
    convert.WithVectorBoundingBox(-25, -40, 60, 40),
    convert.WithVectorWhere("CONTINENT = 'Africa'"),
    convert.WithVectorSimplify(100),
    convert.WithVectorLayerName("africa"),
)
```

### Vector field selection

```go
err := convert.ConvertVector(ctx, src, dst,
    convert.WithVectorFormat("FlatGeobuf"),
    convert.WithVectorSelectFields("NAME", "POP_EST", "CONTINENT"),
)
```

### GeoParquet *(v1.4+)*

`gismanager` v1.4 added GeoParquet support to the conversion subsystem
and the publish-pipeline OGR dispatch. The `.parquet` extension routes
to the GDAL `Parquet` driver in [`OpenSource`](../publish/manager.go) and
`GetDriver`, and `ConvertVector` accepts `Parquet` as a target format.

```go
// Convert a Shapefile bundle into GeoParquet (cloud-native interchange).
err := convert.ConvertVector(ctx,
    "/vsizip//data/countries.zip",
    "/data/countries.parquet",
    convert.WithVectorFormat("Parquet"),
    convert.WithVectorOverwrite(),
)

// Read a GeoParquet file via the publish pipeline (manager-driven).
mgr, _ := publish.New(/* ... */)
ds, _ := mgr.OpenSource(ctx, "/data/cities.parquet", 0)
defer ds.Destroy()
```

**Image variant.** GeoParquet support requires the `Parquet` GDAL OGR
driver, which is bundled in the `ubuntu-full` GDAL image variant but
**not** in `ubuntu-small`. v1.4 changed the project's `Dockerfile` base
from `ubuntu-small` to `ubuntu-full` so both the dev image and the
published runtime image carry the driver out of the box.

The trade-off is image size — the dev/runtime images grew from ~2 GB to
~4 GB. Operators who don't need GeoParquet can pin `GDAL_BASE_DIGEST`
in the `Dockerfile` back to the `ubuntu-small` manifest list digest for
a lighter image. The publish-pipeline driver dispatch and conversion
options stay unchanged either way; only `.parquet` paths fail at the
GDAL boundary on `ubuntu-small` (with a "driver not registered" error).

**Why GeoParquet.** It is the dominant 2026 interchange format for
cloud-native geospatial data — STAC catalogs, Apache Iceberg geospatial
tables, and lakehouse pipelines all use it. Adding driver dispatch +
the `ubuntu-full` base lets gismanager publish from a Parquet source
the same way it publishes from a Shapefile or GeoPackage today.

## Raster conversion

### GeoTIFF → Cloud-Optimized GeoTIFF

```go
err := convert.ToCOG(ctx, "/data/scene.tif", "/data/scene.cog.tif")
```

`ToCOG` defaults to `COMPRESS=DEFLATE`, `BLOCKSIZE=512`,
`OVERVIEW_RESAMPLING=NEAREST`. Override any of them by passing your own
`WithRasterCreationOption`:

```go
err := convert.ToCOG(ctx, src, dst,
    convert.WithRasterCreationOption("COMPRESS", "ZSTD"),
    convert.WithRasterCreationOption("PREDICTOR", "2"),
)
```

### GeoTIFF → PNG thumbnail

```go
err := convert.ConvertRaster(ctx, "/data/scene.tif", "/data/preview.png",
    convert.WithRasterFormat("PNG"),
    convert.WithRasterBands(1, 2, 3),
)
```

### Raster reprojection (UTM → Web Mercator)

```go
err := convert.ReprojectRaster(ctx,
    "/data/scene_utm.tif",
    "/data/scene_3857.tif",
    "EPSG:32618", "EPSG:3857",
    convert.WithRasterFormat("GTiff"),
    convert.WithRasterResamplingAlg("bilinear"),
)
```

### Raster reprojection with cookie-cutter clipping

```go
err := convert.ReprojectRaster(ctx, src, dst,
    "EPSG:4326", "EPSG:3857",
    convert.WithRasterCutline("/data/aoi.geojson", "outline"),
)
```

`WithRasterCutline` adds `-cutline` + `-cl` + `-crop_to_cutline` to the
warp command — pixels outside the polygon become NoData and the output
extent shrinks to the polygon's envelope.

## Vector → raster *(v1.3+)*

### Burn polygons into a Byte mask

```go
err := convert.Rasterize(ctx, "countries.geojson", "africa_mask.tif",
    convert.WithRasterizeFormat("GTiff"),
    convert.WithRasterizeOutputType("Byte"),
    convert.WithRasterizeBurnValues(1.0),
    convert.WithRasterizeWhere("CONTINENT = 'Africa'"),
    convert.WithRasterizeOutputBounds(-25, -40, 60, 40),
    convert.WithRasterizeOutputSize(256, 256),
)
```

### Burn an attribute into a continuous Float32 field

```go
err := convert.Rasterize(ctx, "countries.geojson", "pop.tif",
    convert.WithRasterizeFormat("GTiff"),
    convert.WithRasterizeOutputType("Float32"),
    convert.WithRasterizeAttribute("POP_EST"),
    convert.WithRasterizeOutputSize(360, 180),
)
```

`WithRasterizeBurnValues` and `WithRasterizeAttribute` are mutually
exclusive — use one or the other.

## Multi-raster mosaic *(v1.3+)*

### Combine many GeoTIFFs into a Virtual Raster

```go
err := convert.BuildVRT(ctx, "mosaic.vrt",
    []string{"tile1.tif", "tile2.tif", "tile3.tif"},
    convert.WithVRTResolution("highest"),
    convert.WithVRTAddAlpha(),
)
```

### Stack single-band inputs into RGBA

```go
err := convert.BuildVRT(ctx, "rgba.vrt",
    []string{"red.tif", "green.tif", "blue.tif", "alpha.tif"},
    convert.WithVRTSeparate(),
)
```

`WithVRTSeparate` emits one output band per input dataset; without it,
inputs are tiled into a single (multi-band) mosaic at the inputs'
shared band count.

## DEM analysis *(v1.3+)*

### Hillshade

```go
err := convert.DEMProcessing(ctx, "dem.tif", "dem.hs.tif", "hillshade",
    convert.WithDEMAzimuth(315),
    convert.WithDEMAltitude(45),
    convert.WithDEMMultidirectional(),
)
```

### Slope

```go
err := convert.DEMProcessing(ctx, "dem.tif", "dem.slope.tif", "slope",
    convert.WithDEMAlgorithm("ZevenbergenThorne"),
)
```

### Color-relief (requires a color file)

```go
err := convert.DEMProcessing(ctx, "dem.tif", "dem.color.tif", "color-relief",
    convert.WithDEMColorFile("./elevation_palette.txt"),
)
```

Other supported modes: `aspect`, `TRI` (Terrain Ruggedness Index),
`TPI` (Topographic Position Index), `roughness`. Color-file format
documented at https://gdal.org/programs/gdaldem.html#color-relief.

## PMTiles archive *(v1.4+)*

[PMTiles](https://docs.protomaps.com/pmtiles/) is a single-file
range-readable tile archive — ideal for serverless tile distribution
from S3, HTTP, or any blob storage that supports HTTP range requests.
v1.4 added [`ToPMTiles`](../convert/pmtiles.go), a thin wrapper over
[`protomaps/go-pmtiles`](https://github.com/protomaps/go-pmtiles)
that converts an existing **MBTiles** archive to PMTiles v3.

Two-stage pipeline starting from a raster source:

```go
// 1. raster -> MBTiles (via the GDAL MBTiles driver)
err := convert.ConvertRaster(ctx,
    "/data/scene.tif",
    "/tmp/scene.mbtiles",
    convert.WithRasterFormat("MBTILES"),
)

// 2. MBTiles -> PMTiles
err = convert.ToPMTiles(ctx,
    "/tmp/scene.mbtiles",
    "/data/scene.pmtiles",
)
```

Vector inputs work the same way — produce MBTiles via tippecanoe (or
any other tool that emits MBTiles), then run `ToPMTiles` to repackage.

**Direct raster → PMTiles** (skipping the intermediate MBTiles) is
deferred to v1.5; for v1.4 the two-step path is the supported route.

## Cloud I/O via GDAL Virtual File Systems

All seven conversion entry points pass paths straight through to GDAL,
so any of GDAL's virtual file system prefixes work transparently:

| Prefix | Source | Notes |
|---|---|---|
| `/vsis3/` | Amazon S3 | Set `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` env vars (or use the EC2/ECS instance role) |
| `/vsigs/` | Google Cloud Storage | Set `GS_ACCESS_KEY_ID` + `GS_SECRET_ACCESS_KEY` or use ADC |
| `/vsiaz/` | Azure Blob Storage | Set `AZURE_STORAGE_ACCOUNT` + `AZURE_STORAGE_ACCESS_KEY` |
| `/vsicurl/` | Any HTTP/HTTPS URL | Read-only; supports range requests for COG / FlatGeobuf |
| `/vsizip/` | A `.zip` archive | Required for zipped Shapefile bundles — bare `.zip` paths do NOT auto-prefix |
| `/vsimem/` | In-process memory | Useful for tests and intermediate steps without disk I/O |

### Example: read a remote COG from S3 and reproject locally

```go
err := convert.ReprojectRaster(ctx,
    "/vsis3/satellite-archive/2024/scene.tif",
    "/data/scene.local.tif",
    "EPSG:32618", "EPSG:3857",
)
```

### Example: convert a remote GeoJSON over HTTP

```go
err := convert.ConvertVector(ctx,
    "/vsicurl/https://example.com/data/admin.geojson",
    "/data/admin.gpkg",
    convert.WithVectorFormat("GPKG"),
    convert.WithVectorOverwrite(),
)
```

### Example: in-memory pipeline with `/vsimem/`

```go
convert.ConvertVector(ctx,
    "/vsimem/in.geojson",   // populated upstream via gdal.VSIFileFromMemBuffer
    "/vsimem/out.gpkg",
    convert.WithVectorFormat("GPKG"),
)
```

## Error handling

Every conversion entry point wraps GDAL's error in a `*GISError`. Branch
on the sentinel to detect category:

```go
err := convert.ConvertVector(ctx, src, dst, opts...)
if errors.Is(err, errs.ErrConvertFailed) {
    var gerr *errs.GISError
    if errors.As(err, &gerr) {
        log.Printf("conversion %s failed: %v", gerr.Op, gerr.Cause)
    }
}
```

`gerr.Op` is one of `"ConvertVector"`, `"ConvertRaster"`,
`"ReprojectRaster"`, `"Rasterize"`, `"BuildVRT"`, or `"DEMProcessing"`
(`ToCOG` delegates to `ConvertRaster` so it surfaces as
`"ConvertRaster"`).

## Known gaps

- **Progress callbacks** — `lukeroth/gdal`'s utility wrappers don't
  thread `pfnProgress` through. Long conversions are opaque from the Go
  side. Tracked for v1.4+ (needs an upstream patch).
- **Cancellation mid-conversion** — `ctx` is honored at the function
  boundary (before `OpenEx`), not inside the synchronous CGo call. A
  conversion that's already running cannot be aborted.
- **Other silent-failure modes** — driver-name pre-validation (v1.3)
  closes the most common case, but invalid CRS strings, malformed
  bounding boxes, etc. can still slip past `cerr=0`. Use the dev
  container's CLIs to validate inputs ahead of time if needed.
- **No GeoParquet** in the dev image (`ubuntu-small` excludes Apache
  Arrow / Parquet). Swap to `ubuntu-full` if you need it; revisit in
  v1.4+.
