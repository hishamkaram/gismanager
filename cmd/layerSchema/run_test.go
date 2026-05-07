package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/hishamkaram/gismanager/v2/cmd/internal/cli"
)

// run-level smoke tests for layerSchema, mirroring the cmd/gismanager
// pattern. JSON-shape locking lives in main_test.go; the run-level
// flag-parsing branches live here so the two concerns stay separable.

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

// TestRun_JSONFlag_ParsesWithoutErrorBeforeWalk verifies that -json
// is recognized at flag-parse time (the actual JSON output is
// covered by the existing TestLayerEntryJSON_* tests in main_test.go).
// We can't exercise the full -json + Walk path without testdata + a
// configured Source, but we can verify the flag exists and combines
// with -version cleanly (tests our own arg-parsing wiring).
func TestRun_JSONFlag_RequiresConfig(t *testing.T) {
	err := run(t.Context(), []string{"-json"})
	if err == nil {
		t.Fatal("run -json without config: want error, got nil")
	}
	if !strings.Contains(err.Error(), "--config") {
		t.Errorf("error should mention --config (RequireFlag check fires before JSON path); got: %v", err)
	}
}
