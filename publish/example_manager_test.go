package publish_test

// Runnable godoc examples for the publish-pipeline surface. The
// PublishAll example demonstrates wiring without making real network
// calls (no `// Output:` line, so `go test` only compile-checks it
// rather than running it end-to-end). The OpenSource example uses
// the tracked legacy GeoPackage fixture and prints layer count for a
// `// Output:`-checked round-trip.

import (
	"context"
	"fmt"

	"github.com/hishamkaram/gismanager/publish"
)

// ExampleManager_PublishAll demonstrates the high-level publish
// flow: walk a source directory, load every supported GIS file into
// the configured PostGIS datastore, then publish each as a GeoServer
// feature type.
//
// Programmatic construction via [New] + functional options is
// preferred over [FromConfig] when no YAML file is involved, since
// each option gets compile-time-checked and the manager's logger can
// be threaded in via [WithLogger]. FromConfig + ${VAR} env-var
// interpolation is the right path for ops contexts that store config
// in YAML and secrets in the environment.
//
// The example deliberately does NOT include an `// Output:` comment
// — godoc tests would otherwise try to make a real GeoServer / PostGIS
// connection. The integration suite (//go:build integration) covers
// the runtime contract end-to-end.
func ExampleManager_PublishAll() {
	mgr, err := publish.New(
		publish.WithGeoserver(publish.GeoserverConfig{
			ServerURL:     "http://localhost:8080/geoserver",
			Username:      "admin",
			Password:      "geoserver",
			WorkspaceName: "demo",
		}),
		publish.WithDatastore(publish.DatastoreConfig{
			Host:   "localhost",
			Port:   5432,
			DBName: "gis",
			DBUser: "postgres",
			DBPass: "postgres",
			Name:   "gismanager_demo",
		}),
		publish.WithSource(publish.SourceConfig{Path: "./gis-data"}),
	)
	if err != nil {
		fmt.Println("construct manager:", err)
		return
	}

	if err := mgr.PublishAll(context.Background()); err != nil {
		fmt.Println("publish:", err)
		return
	}
	fmt.Println("published")
}

// ExampleManager_OpenSource shows the read-only inspection
// surface: open a GIS file via the configured GDAL OGR driver and
// list layer count. No PostGIS or GeoServer is touched.
func ExampleManager_OpenSource() {
	mgr, err := publish.New(
		publish.WithSource(publish.SourceConfig{Path: "../testdata"}),
	)
	if err != nil {
		fmt.Println("construct manager:", err)
		return
	}

	ds, err := mgr.OpenSource(context.Background(),
		"../testdata/sample.gpkg", 0)
	if err != nil {
		fmt.Println("open:", err)
		return
	}
	defer ds.Destroy()

	fmt.Println("layers:", ds.LayerCount())
	// Output: layers: 1
}
