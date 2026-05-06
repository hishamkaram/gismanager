package publish

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew_NoOptions_ReturnsUsableManagerWithDefaultLogger(t *testing.T) {
	mgr, err := New()
	assert.NoError(t, err)
	assert.NotNil(t, mgr)
	assert.NotNil(t, mgr.logger, "default logger must be set when no WithLogger option provided")
	assert.Equal(t, GeoserverConfig{}, mgr.Geoserver)
	assert.Equal(t, DatastoreConfig{}, mgr.Datastore)
	assert.Equal(t, SourceConfig{}, mgr.Source)
}

func TestNew_WithGeoserver_SetsConfig(t *testing.T) {
	cfg := GeoserverConfig{
		WorkspaceName: "ws",
		ServerURL:     "http://example/geoserver",
		Username:      "u",
		Password:      "p",
	}
	mgr, err := New(WithGeoserver(cfg))
	assert.NoError(t, err)
	assert.Equal(t, cfg, mgr.Geoserver)
}

func TestNew_WithDatastore_SetsConfig(t *testing.T) {
	cfg := DatastoreConfig{
		Host: "h", Port: 5432, DBName: "db", DBUser: "u", DBPass: "p", Name: "ds",
	}
	mgr, err := New(WithDatastore(cfg))
	assert.NoError(t, err)
	assert.Equal(t, cfg, mgr.Datastore)
}

func TestNew_WithSource_SetsConfig(t *testing.T) {
	cfg := SourceConfig{Path: "/some/path"}
	mgr, err := New(WithSource(cfg))
	assert.NoError(t, err)
	assert.Equal(t, cfg, mgr.Source)
}

func TestNew_WithLogger_AcceptsCustomLogger(t *testing.T) {
	var buf bytes.Buffer
	custom := slog.New(slog.NewJSONHandler(&buf, nil))
	mgr, err := New(WithLogger(custom))
	assert.NoError(t, err)
	assert.Same(t, custom, mgr.logger, "custom logger must be installed verbatim")
}

func TestNew_WithLoggerNil_FallsBackToDefault(t *testing.T) {
	mgr, err := New(WithLogger(nil))
	assert.NoError(t, err)
	assert.NotNil(t, mgr.logger, "WithLogger(nil) must fall back to slogx.Default()")
}

func TestNew_NilOptionIgnored(t *testing.T) {
	// A nil Option must not panic — useful for callers that conditionally
	// build slices like `opts = append(opts, maybeOption(...))` where
	// maybeOption may return nil.
	var skipped Option
	mgr, err := New(skipped, WithSource(SourceConfig{Path: "x"}))
	assert.NoError(t, err)
	assert.Equal(t, "x", mgr.Source.Path)
}

func TestFromConfig_ParityWithNew(t *testing.T) {
	// Loading the same YAML through FromConfig and through New + manual
	// options should yield deep-equal Geoserver / Datastore / Source
	// values. Logger pointers will differ — both are slogx.Default() instances
	// which are not deduped — so compare the configurable fields only.
	viaFromConfig, err := FromConfig("../testdata/test_config.yml")
	assert.NoError(t, err)

	viaNew, err := New(
		WithGeoserver(GeoserverConfig{
			WorkspaceName: "golang",
			ServerURL:     "http://localhost:8080/geoserver",
			Username:      "admin",
			Password:      "geoserver",
		}),
		WithDatastore(DatastoreConfig{
			Host: "postgis", Port: 5432,
			DBName: "gis", DBUser: "golang", DBPass: "golang", Name: "gismanager_data",
		}),
		WithSource(SourceConfig{Path: "../testdata"}),
	)
	assert.NoError(t, err)

	assert.Equal(t, viaFromConfig.Geoserver, viaNew.Geoserver)
	assert.Equal(t, viaFromConfig.Datastore, viaNew.Datastore)
	assert.Equal(t, viaFromConfig.Source, viaNew.Source)
}

func TestNew_OptionsAppliedInOrder(t *testing.T) {
	// Later WithSource overrides earlier — establishes that options are
	// applied left-to-right and behaviour is the standard "last write wins".
	mgr, err := New(
		WithSource(SourceConfig{Path: "first"}),
		WithSource(SourceConfig{Path: "second"}),
	)
	assert.NoError(t, err)
	assert.Equal(t, "second", mgr.Source.Path)
}
