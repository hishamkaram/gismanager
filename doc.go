// Package gismanager is the documentation root for the v2 line of
// github.com/hishamkaram/gismanager. The package itself is empty; all
// public API lives in subpackages:
//
//   - [github.com/hishamkaram/gismanager/v2/publish]: walk a source
//     directory of GIS files (Shapefile / GeoJSON / GeoPackage / KML
//     / GeoParquet), load each layer into PostGIS via the lukeroth/gdal
//     CGo binding, then register the resulting tables as GeoServer
//     feature types via the geoserver/v2 REST client.
//   - [github.com/hishamkaram/gismanager/v2/convert]: stateless GDAL
//     CLI-equivalents (ogr2ogr / gdal_translate / gdalwarp /
//     gdal_rasterize / gdalbuildvrt / gdaldem) plus PMTiles archive
//     generation via protomaps/go-pmtiles.
//   - [github.com/hishamkaram/gismanager/v2/errs]: the typed *GISError
//     envelope and the package-level Err* sentinels both subpackages
//     wrap their failures around.
//
// Three CLI binaries ship under cmd/:
//
//   - cmd/gismanager: walk + load + publish (full pipeline)
//   - cmd/layerSchema: read-only schema introspection (with -json)
//   - cmd/gisconvert: ogr2ogr / gdal_translate-style file-format
//     conversion
//
// v1.x users on the github.com/hishamkaram/gismanager (no /v2 suffix)
// import path stay supported via the release/v1.x branch; see
// MIGRATING.md for the v1 → v2 migration recipe.
package gismanager
