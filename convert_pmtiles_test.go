package gismanager

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestToPMTiles_CtxCanceledFastFail(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := ToPMTiles(ctx, "any.mbtiles", "any.pmtiles")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("want context.Canceled fast-fail, got %v", err)
	}
}

func TestToPMTiles_MissingSourceWrapsErr(t *testing.T) {
	err := ToPMTiles(context.Background(),
		"./testdata/this-file-definitely-does-not-exist.mbtiles",
		"/tmp/should-never-be-written.pmtiles")
	if err == nil {
		t.Fatal("want error on missing source, got nil")
	}
	if !errors.Is(err, ErrConvertFailed) {
		t.Errorf("errors.Is(err, ErrConvertFailed) = false; want true. err=%v", err)
	}
	var gerr *GISError
	if !errors.As(err, &gerr) {
		t.Fatalf("errors.As to *GISError: no match")
	}
	if gerr.Op != "ToPMTiles" {
		t.Errorf("Op = %q; want ToPMTiles", gerr.Op)
	}
}

func TestNewPMTilesConfig_DefaultsAndOptions(t *testing.T) {
	cfg := newPMTilesConfig(nil)
	if cfg.logger == nil {
		t.Error("default config: logger should fall back to GetLogger()")
	}
	if !cfg.deduplicate {
		t.Error("default config: deduplicate should default to true")
	}

	var buf bytes.Buffer
	custom := slog.New(slog.NewTextHandler(&buf, nil))
	cfg = newPMTilesConfig([]PMTilesOption{
		WithPMTilesLogger(custom),
		WithPMTilesDeduplicate(false),
	})
	if cfg.logger != custom {
		t.Error("WithPMTilesLogger did not override the logger")
	}
	if cfg.deduplicate {
		t.Error("WithPMTilesDeduplicate(false) did not flip the flag")
	}

	// Nil-logger guard: WithPMTilesLogger(nil) should retain the
	// previously-set logger rather than panic on dereference later.
	cfg = newPMTilesConfig([]PMTilesOption{
		WithPMTilesLogger(custom),
		WithPMTilesLogger(nil),
	})
	if cfg.logger != custom {
		t.Error("WithPMTilesLogger(nil) discarded a real logger")
	}
}

// TestToPMTiles_ErrorMessagesIncludePath documents the contract that
// a stat-failed source yields an error whose Source field carries the
// user-supplied path — useful for triage in batch jobs converting
// many tilesets in one run.
func TestToPMTiles_ErrorMessagesIncludePath(t *testing.T) {
	src := "./testdata/missing-tile-archive-zzz.mbtiles"
	err := ToPMTiles(context.Background(), src, "/tmp/x.pmtiles")
	if err == nil {
		t.Fatal("want error on missing source")
	}
	if !strings.Contains(err.Error(), src) {
		t.Errorf("error message should include source path %q; got: %v", src, err)
	}
}
