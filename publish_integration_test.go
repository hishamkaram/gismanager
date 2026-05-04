//go:build integration

package gismanager_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/workspaces"

	"github.com/hishamkaram/gismanager"
)

// integrationConfig builds a *gismanager.ManagerConfig from the environment.
// Defaults match docker-compose.test.yml's test-runner service so the test
// works in-cluster (compose internal hostnames) and is overridable for
// out-of-cluster use (export the env vars before running go test).
func integrationConfig(t *testing.T, workspaceName string) *gismanager.ManagerConfig {
	t.Helper()

	cfgPath := filepath.Join(t.TempDir(), "integration_config.yml")
	port, err := strconv.Atoi(envOr("GISMANAGER_TEST_POSTGIS_PORT", "5432"))
	if err != nil {
		t.Fatalf("parse postgis port: %v", err)
	}
	yaml := fmt.Sprintf(`
geoserver:
  url: %s
  username: %s
  password: %s
  workspace: %s
datastore:
  host: %s
  port: %d
  database: %s
  username: %s
  password: %s
  name: gismanager_test_ds
source:
  path: ./testdata
`,
		envOr("GISMANAGER_TEST_GEOSERVER_URL", "http://localhost:8080/geoserver"),
		envOr("GISMANAGER_TEST_GEOSERVER_USER", "admin"),
		envOr("GISMANAGER_TEST_GEOSERVER_PASSWORD", "geoserver"),
		workspaceName,
		envOr("GISMANAGER_TEST_POSTGIS_HOST", "localhost"),
		port,
		envOr("GISMANAGER_TEST_POSTGIS_DB", "gis"),
		envOr("GISMANAGER_TEST_POSTGIS_USER", "golang"),
		envOr("GISMANAGER_TEST_POSTGIS_PASSWORD", "golang"),
	)
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
		t.Fatalf("write tmp config: %v", err)
	}

	mgr, err := gismanager.FromConfig(cfgPath)
	if err != nil {
		t.Fatalf("FromConfig: %v", err)
	}
	return mgr
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// uniqueWorkspaceName returns a workspace name unique to the running test
// so concurrent or sequential test runs don't interfere with each other on
// a shared GeoServer instance.
func uniqueWorkspaceName(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("gismgr_it_%d", time.Now().UnixNano())
}

// cleanup deletes the workspace recursively. Called via t.Cleanup so tests
// leave a clean GeoServer behind even on failure.
func cleanup(t *testing.T, mgr *gismanager.ManagerConfig, ws string) {
	t.Helper()
	c, err := mgr.GetGeoserverCatalog()
	if err != nil {
		t.Logf("cleanup: build geoserver client: %v", err)
		return
	}
	if err := c.Workspaces.Delete(context.Background(), ws, workspaces.DeleteOptions{Recurse: true}); err != nil {
		// ErrNotFound is fine — nothing to clean up.
		if !errors.Is(err, geoserver.ErrNotFound) {
			t.Logf("cleanup: delete workspace %q: %v", ws, err)
		}
	}
}

// TestPublishGeoJSON_EndToEnd_Integration exercises the full pipeline:
// open a GeoJSON via GDAL, copy it into PostGIS via OGR's PostgreSQL
// driver, then publish the resulting table as a GeoServer feature type.
// Verifies via the v2 client that the workspace, datastore, and feature
// type all exist after the publish completes.
func TestPublishGeoJSON_EndToEnd_Integration(t *testing.T) {
	ctx := context.Background()
	ws := uniqueWorkspaceName(t)
	mgr := integrationConfig(t, ws)
	t.Cleanup(func() { cleanup(t, mgr, ws) })

	// Pick a small GeoJSON fixture so the test stays fast.
	srcPath := "./testdata/neighborhood_names_gis.geojson"
	src, err := mgr.OpenSource(ctx, srcPath, 0)
	if err != nil {
		t.Fatalf("OpenSource %s: %v", srcPath, err)
	}
	target, err := mgr.OpenSource(ctx, mgr.Datastore.BuildConnectionString(), 1)
	if err != nil {
		t.Fatalf("OpenSource postgis: %v", err)
	}

	if src.LayerCount() == 0 {
		t.Fatalf("source %s has no layers", srcPath)
	}
	layer := src.LayerByIndex(0)
	gLayer := gismanager.GdalLayer{Layer: &layer}

	newLayer, err := gLayer.LayerToPostgis(target, mgr, true)
	if err != nil {
		t.Fatalf("LayerToPostgis: %v", err)
	}
	if newLayer == nil || newLayer.Layer == nil {
		t.Fatal("LayerToPostgis returned nil layer")
	}

	if err := mgr.PublishGeoserverLayer(ctx, newLayer); err != nil {
		t.Fatalf("PublishGeoserverLayer: %v", err)
	}

	// Verify via the v2 client: workspace + datastore + featuretype all exist.
	c, err := mgr.GetGeoserverCatalog()
	if err != nil {
		t.Fatalf("GetGeoserverCatalog: %v", err)
	}
	if _, err := c.Workspaces.Get(ctx, ws); err != nil {
		t.Errorf("workspace %q not found after publish: %v", ws, err)
	}
	if _, err := c.Datastores.InWorkspace(ws).Get(ctx, mgr.Datastore.Name); err != nil {
		t.Errorf("datastore %q not found after publish: %v", mgr.Datastore.Name, err)
	}
	layerName := newLayer.Name()
	if _, err := c.FeatureTypes.InWorkspace(ws).InDatastore(mgr.Datastore.Name).Get(ctx, layerName); err != nil {
		t.Errorf("feature type %q not found after publish: %v", layerName, err)
	}
}

// TestPublishGeoJSON_Idempotent_Integration calls PublishGeoserverLayer
// twice and asserts the second call succeeds (no ErrConflict). Locks in
// the Get+ErrNotFound idempotency contract for workspace + datastore +
// feature type.
func TestPublishGeoJSON_Idempotent_Integration(t *testing.T) {
	ctx := context.Background()
	ws := uniqueWorkspaceName(t)
	mgr := integrationConfig(t, ws)
	t.Cleanup(func() { cleanup(t, mgr, ws) })

	src, err := mgr.OpenSource(ctx, "./testdata/neighborhood_names_gis.geojson", 0)
	if err != nil {
		t.Fatalf("OpenSource: %v", err)
	}
	target, err := mgr.OpenSource(ctx, mgr.Datastore.BuildConnectionString(), 1)
	if err != nil {
		t.Fatalf("OpenSource postgis: %v", err)
	}
	layer := src.LayerByIndex(0)
	gLayer := gismanager.GdalLayer{Layer: &layer}

	newLayer, err := gLayer.LayerToPostgis(target, mgr, true)
	if err != nil {
		t.Fatalf("LayerToPostgis: %v", err)
	}

	for i := 0; i < 2; i++ {
		if err := mgr.PublishGeoserverLayer(ctx, newLayer); err != nil {
			t.Fatalf("publish iteration %d: %v", i+1, err)
		}
	}
}

// TestUnsupportedFormat_Integration confirms that the sentinel surfaces
// against a real driver-lookup attempt rather than only a unit-style mock.
// Belt-and-braces — also covered in unit tests.
func TestUnsupportedFormat_Integration(t *testing.T) {
	ctx := context.Background()
	ws := uniqueWorkspaceName(t)
	mgr := integrationConfig(t, ws)
	// No t.Cleanup — no workspace was ever created.

	_, err := mgr.OpenSource(ctx, "./testdata/sample_dummy.xml", 0)
	if !errors.Is(err, gismanager.ErrUnsupportedFormat) {
		t.Fatalf("expected ErrUnsupportedFormat, got %v", err)
	}
}

// TestLayerToPostgis_NilDatasource_Integration locks in the
// ErrInvalidDatasource branch of LayerToPostgis. It needs the integration
// build tag because DBIsAlive runs first and would fail with
// ErrPostGISConnect when no PostGIS is reachable, masking the
// nil-datasource check this test is asserting. Direct port of the same
// case from the pre-revival ManagerLayerSuite.TestLayerOperations.
func TestLayerToPostgis_NilDatasource_Integration(t *testing.T) {
	ctx := context.Background()
	mgr := integrationConfig(t, uniqueWorkspaceName(t))
	src, err := mgr.OpenSource(ctx, "./testdata/neighborhood_names_gis.geojson", 0)
	if err != nil {
		t.Fatalf("OpenSource: %v", err)
	}
	layer := src.LayerByIndex(0)
	gLayer := gismanager.GdalLayer{Layer: &layer}

	got, postgisErr := gLayer.LayerToPostgis(nil, mgr, true)
	if got != nil {
		t.Errorf("got non-nil layer with nil targetSource: %#v", got)
	}
	if !errors.Is(postgisErr, gismanager.ErrInvalidDatasource) {
		t.Fatalf("expected ErrInvalidDatasource, got %v", postgisErr)
	}
}

// TestLayerToPostgis_NilLayer_Integration locks in the ErrInvalidLayer
// branch of LayerToPostgis. Direct port of the same case from the
// pre-revival ManagerLayerSuite.TestLayerOperations.
func TestLayerToPostgis_NilLayer_Integration(t *testing.T) {
	ctx := context.Background()
	mgr := integrationConfig(t, uniqueWorkspaceName(t))
	target, err := mgr.OpenSource(ctx, mgr.Datastore.BuildConnectionString(), 1)
	if err != nil {
		t.Fatalf("OpenSource postgis: %v", err)
	}

	gLayer := gismanager.GdalLayer{} // nil embedded *gdal.Layer

	got, postgisErr := gLayer.LayerToPostgis(target, mgr, true)
	if got != nil {
		t.Errorf("got non-nil layer with nil GdalLayer.Layer: %#v", got)
	}
	if !errors.Is(postgisErr, gismanager.ErrInvalidLayer) {
		t.Fatalf("expected ErrInvalidLayer, got %v", postgisErr)
	}
}
