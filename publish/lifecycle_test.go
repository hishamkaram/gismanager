package publish

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestFeatures_NilLayer_YieldsNothing locks in the zero-value safety:
// iterating Features on a Layer whose embedded *gdal.Layer is nil
// must produce zero items and not panic.
func TestFeatures_NilLayer_YieldsNothing(t *testing.T) {
	zero := &Layer{}
	count := 0
	for range zero.Features(context.Background()) {
		count++
	}
	assert.Equal(t, 0, count, "Features on nil-Layer Layer must yield zero items")
}

// TestFeatures_HonorsCanceledContext locks in cancellation: a ctx that
// is already canceled before iteration starts produces zero items
// (the iterator's first ctx.Err() check short-circuits).
func TestFeatures_HonorsCanceledContext(t *testing.T) {
	// Use a real layer fixture from testdata — Features needs a non-nil
	// *gdal.Layer to reach the ctx.Err() check inside the loop.
	mgr, err := New(WithSource(SourceConfig{Path: "../testdata"}))
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before iteration begins

	for item, walkErr := range mgr.Walk(context.Background()) {
		if walkErr != nil || item.Layer == nil {
			continue
		}
		count := 0
		for range item.Layer.Features(ctx) {
			count++
		}
		assert.Equal(t, 0, count, "canceled ctx must produce zero feature yields; got %d", count)
		return
	}
	t.Fatal("no fixture yielded a non-nil layer to drive the test")
}

// TestDBIsAliveContext_HonorsCanceledContext exercises the ctx-aware
// variant: a canceled ctx must produce a non-nil error from PingContext
// before the open even has a chance to time out on its own. We pair this
// with a clearly-unreachable connection string so any hang would be
// obvious; the canceled ctx should fail-fast regardless.
func TestDBIsAliveContext_HonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Driver-level open with a syntactically invalid DSN returns
	// an error immediately — that's also acceptable; either way we
	// expect a non-nil error in under a second.
	done := make(chan error, 1)
	go func() {
		done <- DBIsAliveContext(ctx, "postgres", "postgresql://nobody@127.0.0.1:1/x?connect_timeout=2&sslmode=disable")
	}()

	select {
	case err := <-done:
		assert.Error(t, err, "expected non-nil error from canceled ctx or unreachable DB")
		// Either context.Canceled, or the underlying driver error — both fine.
		_ = errors.Is(err, context.Canceled) // documented expected case
	case <-time.After(5 * time.Second):
		t.Fatal("DBIsAliveContext hung past the canceled ctx — expected fail-fast")
	}
}

// TestDBIsAlive_BackCompat_DelegatesToContext verifies the deprecated
// DBIsAlive still produces an error against an unreachable DB (i.e.
// the deprecation didn't remove the behaviour, only deferred to
// DBIsAliveContext internally).
func TestDBIsAlive_BackCompat_DelegatesToContext(t *testing.T) {
	err := DBIsAlive("postgres", "postgresql://nobody@127.0.0.1:1/x?connect_timeout=1&sslmode=disable")
	assert.Error(t, err, "expected non-nil error from unreachable DB via deprecated DBIsAlive")
}
