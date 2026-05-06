package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/hishamkaram/gismanager/v2/cmd/internal/cli"
)

// run-level smoke tests for the gismanager CLI. They exercise the
// flag-parsing and config-loading branches without booting any real
// PostGIS / GeoServer — the publish path is covered separately by
// the integration suite. These cases keep cmd/gismanager from
// regressing back to 0.0% coverage as the package grows.

func TestRun_VersionFlagShortCircuits(t *testing.T) {
	err := run(t.Context(), []string{"-version"})
	if !errors.Is(err, cli.ErrVersionRequested) {
		t.Errorf("run -version: want ErrVersionRequested, got %v", err)
	}
}

func TestRun_MissingConfigFlagErrors(t *testing.T) {
	err := run(t.Context(), nil)
	if err == nil {
		t.Fatal("run with no args: want error, got nil")
	}
	if !strings.Contains(err.Error(), "--config") {
		t.Errorf("error should mention --config; got: %v", err)
	}
}

func TestRun_NonexistentConfigErrors(t *testing.T) {
	err := run(t.Context(), []string{
		"-config", "/path/that/should/never/exist/foo.yml",
	})
	if err == nil {
		t.Fatal("nonexistent config: want error, got nil")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("error should mention nonexistence; got: %v", err)
	}
}

// TestRun_MalformedConfigErrors uses the deliberately malformed YAML
// fixture (wrapped in <error> tags). FromConfig surfaces the unmarshal
// failure wrapped in *GISError with ErrConfigInvalid.
func TestRun_MalformedConfigErrors(t *testing.T) {
	err := run(t.Context(), []string{
		"-config", "../../testdata/test_config_err.yml",
	})
	if err == nil {
		t.Fatal("malformed config: want error, got nil")
	}
}
