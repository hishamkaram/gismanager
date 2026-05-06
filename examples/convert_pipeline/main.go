// Example: a 30-line tile-prep pipeline that reprojects a Natural Earth
// countries GeoJSON to Web Mercator, clips to an Africa bounding box,
// filters by continent, and writes the result as a single-layer
// GeoPackage suitable for tile generation.
//
// Run with:
//
//	go run ./examples/convert_pipeline -in countries.geojson -out africa.gpkg
//
// The fixture is downloaded by `make fetch-testdata`; alternatively
// fetch any 4326 polygon-features GeoJSON yourself.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/hishamkaram/gismanager/convert"
)

func main() {
	in := flag.String("in", "", "source GeoJSON path")
	out := flag.String("out", "", "destination GeoPackage path")
	flag.Parse()
	if *in == "" || *out == "" {
		slog.Error("usage: -in <countries.geojson> -out <africa.gpkg>")
		os.Exit(1)
	}

	if err := convert.ConvertVector(context.Background(), *in, *out,
		convert.WithVectorFormat("GPKG"),
		convert.WithVectorOverwrite(),
		convert.WithVectorTargetSRS("EPSG:3857"),
		convert.WithVectorBoundingBox(-25, -40, 60, 40),
		convert.WithVectorWhere("CONTINENT = 'Africa'"),
		convert.WithVectorLayerName("africa"),
		convert.WithVectorSimplify(100),
	); err != nil {
		slog.Error("convert", "err", err)
		os.Exit(1)
	}

	slog.Info("converted", "src", *in, "dst", *out)
}
