package gismanager_test

// Runnable godoc examples for the conversion subsystem. Each example
// runs hermetically via the dev container's GDAL build — the source
// fixture is the tracked legacy GeoJSON (no `make fetch-testdata`
// required), and synthetic rasters are generated in /vsimem/ on the
// fly so no host-side raster fixture is needed either.
//
// `// Output:` comments at the bottom of each example are checked by
// `go test`; if the example diverges from its documented output, the
// suite fails. This is the lock-in for "the example as written
// actually works against the current API and current GDAL build."

import (
	"context"
	"fmt"

	"github.com/lukeroth/gdal"

	"github.com/hishamkaram/gismanager"
)

// ExampleConvertVector demonstrates a vector format conversion from
// GeoJSON to GeoPackage in the in-process /vsimem/ VFS — no
// filesystem write, no network. Same shape works for /vsis3/, /vsigs/,
// /vsicurl/ destinations.
func ExampleConvertVector() {
	src := "./testdata/neighborhood_names_gis.geojson"
	dst := "/vsimem/example_convert_vector.gpkg"

	if err := gismanager.ConvertVector(context.Background(), src, dst,
		gismanager.WithVectorFormat("GPKG"),
		gismanager.WithVectorOverwrite(),
	); err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("ok")
	// Output: ok
}

// ExampleToCOG demonstrates Cloud-Optimized GeoTIFF generation. The
// helper pre-fills sane defaults (DEFLATE compression, 512 blocks,
// NEAREST overview resampling); caller-supplied options override.
//
// The example synthesizes a tiny 16x16 input raster in /vsimem/ so
// it runs hermetically without any pre-fetched fixture.
func ExampleToCOG() {
	src := "/vsimem/example_to_cog_src.tif"
	dst := "/vsimem/example_to_cog_dst.tif"

	// Synthesize a minimal source raster.
	driver, _ := gdal.GetDriverByName("GTiff")
	srcDS := driver.Create(src, 16, 16, 1, gdal.Byte, nil)
	srcDS.Close()

	if err := gismanager.ToCOG(context.Background(), src, dst); err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("ok")
	// Output: ok
}

// ExampleReprojectRaster demonstrates raster reprojection (the
// gdalwarp equivalent). Requires both source and target SRS; if a
// cookie-cutter clip is needed pass [WithRasterCutline].
func ExampleReprojectRaster() {
	src := "/vsimem/example_reproject_src.tif"
	dst := "/vsimem/example_reproject_dst.tif"

	// Synthesize a 16x16 raster with a defined geotransform and SRS so
	// the reprojection has something real to act on.
	driver, _ := gdal.GetDriverByName("GTiff")
	srcDS := driver.Create(src, 16, 16, 1, gdal.Byte, nil)
	srcDS.SetGeoTransform([6]float64{0, 1, 0, 16, 0, -1})
	wgs84 := gdal.CreateSpatialReference("")
	_ = wgs84.FromEPSG(4326)
	wkt, _ := wgs84.ToWKT()
	srcDS.SetProjection(wkt)
	srcDS.Close()

	if err := gismanager.ReprojectRaster(context.Background(), src, dst,
		"EPSG:4326", "EPSG:3857",
	); err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("ok")
	// Output: ok
}

// ExampleRasterize demonstrates burning vector features into a
// raster grid (the gdal_rasterize equivalent).
func ExampleRasterize() {
	src := "./testdata/neighborhood_names_gis.geojson"
	dst := "/vsimem/example_rasterize.tif"

	if err := gismanager.Rasterize(context.Background(), src, dst,
		gismanager.WithRasterizeFormat("GTiff"),
		gismanager.WithRasterizeOutputSize(64, 64),
		gismanager.WithRasterizeBurnValues(1),
	); err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("ok")
	// Output: ok
}

// ExampleDEMProcessing demonstrates DEM analysis (the gdaldem
// equivalent). Supported modes: hillshade, slope, aspect,
// color-relief, TRI, TPI, roughness. Color-relief requires
// [WithDEMColorFile]; the others need only the input DEM.
func ExampleDEMProcessing() {
	src := "/vsimem/example_dem_src.tif"
	dst := "/vsimem/example_dem_hillshade.tif"

	// Synthesize a 32x32 elevation raster (constant value is fine —
	// hillshade just needs a defined geotransform).
	driver, _ := gdal.GetDriverByName("GTiff")
	srcDS := driver.Create(src, 32, 32, 1, gdal.Float32, nil)
	srcDS.SetGeoTransform([6]float64{0, 1, 0, 32, 0, -1})
	srcDS.Close()

	if err := gismanager.DEMProcessing(context.Background(), src, dst,
		"hillshade",
	); err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("ok")
	// Output: ok
}

// ExampleBuildVRT demonstrates assembling several rasters into a
// single Virtual Raster (the gdalbuildvrt equivalent). Useful for
// presenting tile pyramids or stacking single-band inputs with
// [WithVRTSeparate].
func ExampleBuildVRT() {
	src1 := "/vsimem/example_buildvrt_a.tif"
	src2 := "/vsimem/example_buildvrt_b.tif"
	dst := "/vsimem/example_buildvrt.vrt"

	driver, _ := gdal.GetDriverByName("GTiff")
	for _, p := range []string{src1, src2} {
		ds := driver.Create(p, 16, 16, 1, gdal.Byte, nil)
		ds.SetGeoTransform([6]float64{0, 1, 0, 16, 0, -1})
		ds.Close()
	}

	if err := gismanager.BuildVRT(context.Background(), dst,
		[]string{src1, src2},
	); err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("ok")
	// Output: ok
}
