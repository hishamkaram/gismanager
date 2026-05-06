package slogx

import (
	"testing"
)

func TestDefault(t *testing.T) {
	l := Default()
	if l == nil {
		t.Fatal("Default() returned nil")
	}
	// Default()'s signature returns *slog.Logger; the public API
	// (gismanager.GetLogger) advertises that exact shape via the
	// thin wrapper. No further type-assertion is needed at the
	// test layer — a return-type drift would surface as a compile
	// error in the wrapper.
}

// TestDefault_DistinctInstances guards against an accidental "return
// the same global *slog.Logger" implementation. Each call constructs
// a fresh handler — callers attach context (e.g. WithAttrs) without
// affecting other call sites.
func TestDefault_DistinctInstances(t *testing.T) {
	a := Default()
	b := Default()
	if a == b {
		t.Errorf("Default() should return distinct instances each call; got the same pointer")
	}
}
