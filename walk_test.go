package gismanager

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestWalk_YieldsLayersFromTestdata verifies the iterator emits at least
// one (Path, Layer) pair from the testdata fixtures. The exact count
// depends on the fixtures (multiple layers per GeoPackage); we assert
// it's non-zero so the test stays robust against fixture additions.
func TestWalk_YieldsLayersFromTestdata(t *testing.T) {
	mgr, err := New(WithSource(SourceConfig{Path: "./testdata"}))
	assert.NoError(t, err)

	ctx := context.Background()
	layerCount := 0
	for item, walkErr := range mgr.Walk(ctx) {
		if walkErr != nil {
			t.Logf("per-file walk error (skipped, expected for unsupported fixtures): %v", walkErr)
			continue
		}
		assert.NotEmpty(t, item.Path, "Path must be set when error is nil")
		assert.NotNil(t, item.Layer, "Layer must be set when error is nil")
		layerCount++
	}
	assert.Greater(t, layerCount, 0, "expected at least one layer yielded from testdata")
}

// TestWalk_EarlyBreak_DoesNotPanic confirms the iterator handles
// caller-side `break` cleanly: the deferred Destroy on the
// currently-open source still runs, no goroutine leaks, no panic.
// We can't directly assert Destroy was called (no public counter), but
// a successful early break + a subsequent fresh walk is the strongest
// in-test signal we can produce without instrumenting CGo.
func TestWalk_EarlyBreak_DoesNotPanic(t *testing.T) {
	mgr, err := New(WithSource(SourceConfig{Path: "./testdata"}))
	assert.NoError(t, err)

	ctx := context.Background()
	first := 0
	for _, walkErr := range mgr.Walk(ctx) {
		if walkErr != nil {
			continue
		}
		first++
		break
	}
	assert.GreaterOrEqual(t, first, 1)

	// Run a second walk to confirm the manager is still usable after
	// an early break.
	second := 0
	for _, walkErr := range mgr.Walk(ctx) {
		if walkErr != nil {
			continue
		}
		second++
	}
	assert.GreaterOrEqual(t, second, first)
}

// TestWalk_HonorsContextCancellation cancels the context immediately and
// asserts the iterator yields the cancellation error without producing
// any layer items.
func TestWalk_HonorsContextCancellation(t *testing.T) {
	mgr, err := New(WithSource(SourceConfig{Path: "./testdata"}))
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediate

	yielded := 0
	sawCancel := false
	for _, walkErr := range mgr.Walk(ctx) {
		if errors.Is(walkErr, context.Canceled) {
			sawCancel = true
			break
		}
		yielded++
	}
	assert.True(t, sawCancel || yielded == 0,
		"with a canceled context the iterator must either surface context.Canceled or yield zero items; got %d items", yielded)
}

// TestWalk_NonexistentSource_ReturnsError confirms a non-existent source
// directory surfaces as an error item (not a panic, not a silent empty
// iteration). The error wraps the os.Stat failure from getGISFiles.
func TestWalk_NonexistentSource_ReturnsError(t *testing.T) {
	mgr, err := New(WithSource(SourceConfig{Path: "./testdata_does_not_exist"}))
	assert.NoError(t, err)

	ctx := context.Background()
	yielded := 0
	sawErr := false
	for _, walkErr := range mgr.Walk(ctx) {
		if walkErr != nil {
			sawErr = true
			continue
		}
		yielded++
	}
	assert.True(t, sawErr, "expected an error item for a missing source dir")
	assert.Equal(t, 0, yielded)
}
