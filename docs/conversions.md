# Conversions

`gismanager` v1.2 ships a small, stateless conversion subsystem covering
the four most common GDAL/OGR operations:

- **`ConvertVector`** — vector format conversion (the `ogr2ogr` equivalent)
- **`ConvertRaster`** — raster format conversion (the `gdal_translate` equivalent)
- **`ToCOG`** — Cloud-Optimized GeoTIFF generation (a thin wrapper over `ConvertRaster`)
- **`ReprojectRaster`** — raster reprojection (the `gdalwarp` equivalent)

Each is a top-level package function; none requires a `*ManagerConfig`.
Configuration goes through functional options (`WithVector*` or
`WithRaster*` modality prefix to avoid name collisions).

## Vector conversion

### Shapefile → GeoPackage

```go
err := gismanager.ConvertVector(ctx,
    "/vsizip//data/countries.zip", // /vsizip/ for zipped Shapefile bundles
    "/data/countries.gpkg",
    gismanager.WithVectorFormat("GPKG"),
    gismanager.WithVectorOverwrite(),
)
```

### GeoJSON → GeoPackage with reprojection, bbox clip, and attribute filter

```go
err := gismanager.ConvertVector(ctx,
    "/data/world.geojson",
    "/data/africa_3857.gpkg",
    gismanager.WithVectorFormat("GPKG"),
    gismanager.WithVectorOverwrite(),
    gismanager.WithVectorTargetSRS("EPSG:3857"),
    gismanager.WithVectorBoundingBox(-25, -40, 60, 40),
    gismanager.WithVectorWhere("CONTINENT = 'Africa'"),
    gismanager.WithVectorSimplify(100),
    gismanager.WithVectorLayerName("africa"),
)
```

### Vector field selection

```go
err := gismanager.ConvertVector(ctx, src, dst,
    gismanager.WithVectorFormat("FlatGeobuf"),
    gismanager.WithVectorSelectFields("NAME", "POP_EST", "CONTINENT"),
)
```

## Raster conversion

### GeoTIFF → Cloud-Optimized GeoTIFF

```go
err := gismanager.ToCOG(ctx, "/data/scene.tif", "/data/scene.cog.tif")
```

`ToCOG` defaults to `COMPRESS=DEFLATE`, `BLOCKSIZE=512`,
`OVERVIEW_RESAMPLING=NEAREST`. Override any of them by passing your own
`WithRasterCreationOption`:

```go
err := gismanager.ToCOG(ctx, src, dst,
    gismanager.WithRasterCreationOption("COMPRESS", "ZSTD"),
    gismanager.WithRasterCreationOption("PREDICTOR", "2"),
)
```

### GeoTIFF → PNG thumbnail

```go
err := gismanager.ConvertRaster(ctx, "/data/scene.tif", "/data/preview.png",
    gismanager.WithRasterFormat("PNG"),
    gismanager.WithRasterBands(1, 2, 3),
)
```

### Raster reprojection (UTM → Web Mercator)

```go
err := gismanager.ReprojectRaster(ctx,
    "/data/scene_utm.tif",
    "/data/scene_3857.tif",
    "EPSG:32618", "EPSG:3857",
    gismanager.WithRasterFormat("GTiff"),
    gismanager.WithRasterResamplingAlg("bilinear"),
)
```

### Raster reprojection with cookie-cutter clipping

```go
err := gismanager.ReprojectRaster(ctx, src, dst,
    "EPSG:4326", "EPSG:3857",
    gismanager.WithRasterCutline("/data/aoi.geojson", "outline"),
)
```

`WithRasterCutline` adds `-cutline` + `-cl` + `-crop_to_cutline` to the
warp command — pixels outside the polygon become NoData and the output
extent shrinks to the polygon's envelope.

## Vector → raster *(v1.3+)*

### Burn polygons into a Byte mask

```go
err := gismanager.Rasterize(ctx, "countries.geojson", "africa_mask.tif",
    gismanager.WithRasterizeFormat("GTiff"),
    gismanager.WithRasterizeOutputType("Byte"),
    gismanager.WithRasterizeBurnValues(1.0),
    gismanager.WithRasterizeWhere("CONTINENT = 'Africa'"),
    gismanager.WithRasterizeOutputBounds(-25, -40, 60, 40),
    gismanager.WithRasterizeOutputSize(256, 256),
)
```

### Burn an attribute into a continuous Float32 field

```go
err := gismanager.Rasterize(ctx, "countries.geojson", "pop.tif",
    gismanager.WithRasterizeFormat("GTiff"),
    gismanager.WithRasterizeOutputType("Float32"),
    gismanager.WithRasterizeAttribute("POP_EST"),
    gismanager.WithRasterizeOutputSize(360, 180),
)
```

`WithRasterizeBurnValues` and `WithRasterizeAttribute` are mutually
exclusive — use one or the other.

## Multi-raster mosaic *(v1.3+)*

### Combine many GeoTIFFs into a Virtual Raster

```go
err := gismanager.BuildVRT(ctx, "mosaic.vrt",
    []string{"tile1.tif", "tile2.tif", "tile3.tif"},
    gismanager.WithVRTResolution("highest"),
    gismanager.WithVRTAddAlpha(),
)
```

### Stack single-band inputs into RGBA

```go
err := gismanager.BuildVRT(ctx, "rgba.vrt",
    []string{"red.tif", "green.tif", "blue.tif", "alpha.tif"},
    gismanager.WithVRTSeparate(),
)
```

`WithVRTSeparate` emits one output band per input dataset; without it,
inputs are tiled into a single (multi-band) mosaic at the inputs'
shared band count.

## DEM analysis *(v1.3+)*

### Hillshade

```go
err := gismanager.DEMProcessing(ctx, "dem.tif", "dem.hs.tif", "hillshade",
    gismanager.WithDEMAzimuth(315),
    gismanager.WithDEMAltitude(45),
    gismanager.WithDEMMultidirectional(),
)
```

### Slope

```go
err := gismanager.DEMProcessing(ctx, "dem.tif", "dem.slope.tif", "slope",
    gismanager.WithDEMAlgorithm("ZevenbergenThorne"),
)
```

### Color-relief (requires a color file)

```go
err := gismanager.DEMProcessing(ctx, "dem.tif", "dem.color.tif", "color-relief",
    gismanager.WithDEMColorFile("./elevation_palette.txt"),
)
```

Other supported modes: `aspect`, `TRI` (Terrain Ruggedness Index),
`TPI` (Topographic Position Index), `roughness`. Color-file format
documented at https://gdal.org/programs/gdaldem.html#color-relief.

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
err := gismanager.ReprojectRaster(ctx,
    "/vsis3/satellite-archive/2024/scene.tif",
    "/data/scene.local.tif",
    "EPSG:32618", "EPSG:3857",
)
```

### Example: convert a remote GeoJSON over HTTP

```go
err := gismanager.ConvertVector(ctx,
    "/vsicurl/https://example.com/data/admin.geojson",
    "/data/admin.gpkg",
    gismanager.WithVectorFormat("GPKG"),
    gismanager.WithVectorOverwrite(),
)
```

### Example: in-memory pipeline with `/vsimem/`

```go
gismanager.ConvertVector(ctx,
    "/vsimem/in.geojson",   // populated upstream via gdal.VSIFileFromMemBuffer
    "/vsimem/out.gpkg",
    gismanager.WithVectorFormat("GPKG"),
)
```

## Error handling

Every conversion entry point wraps GDAL's error in a `*GISError`. Branch
on the sentinel to detect category:

```go
err := gismanager.ConvertVector(ctx, src, dst, opts...)
if errors.Is(err, gismanager.ErrConvertFailed) {
    var gerr *gismanager.GISError
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
