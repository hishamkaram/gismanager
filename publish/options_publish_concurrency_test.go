package publish

import "testing"

// Tests for the v2 [WithPublishConcurrency] option.
//
// We can't drive PublishAll's actual concurrency in a unit test (would
// need a live GeoServer + PostGIS); the integration suite covers the
// runtime flow. These cases lock in the option's effect on the manager
// state, which the runtime path then reads.

func TestWithPublishConcurrency_ExplicitValue(t *testing.T) {
	mgr, err := New(WithPublishConcurrency(16))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := mgr.publishConcurrency; got != 16 {
		t.Errorf("publishConcurrency = %d; want 16", got)
	}
}

func TestWithPublishConcurrency_ZeroFallsBackToDefault(t *testing.T) {
	mgr, _ := New(WithPublishConcurrency(0))
	if got := mgr.publishConcurrency; got != 0 {
		t.Errorf("publishConcurrency for zero arg = %d; want 0 (PublishAll then uses defaultPublishConcurrency)", got)
	}
}

func TestWithPublishConcurrency_NegativeFallsBackToDefault(t *testing.T) {
	mgr, _ := New(WithPublishConcurrency(-3))
	if got := mgr.publishConcurrency; got != 0 {
		t.Errorf("publishConcurrency for negative arg = %d; want 0 (treated as default)", got)
	}
}

func TestWithPublishConcurrency_OneEnablesSerial(t *testing.T) {
	mgr, _ := New(WithPublishConcurrency(1))
	if got := mgr.publishConcurrency; got != 1 {
		t.Errorf("publishConcurrency = %d; want 1 (serial publish)", got)
	}
}

func TestNewWithoutOption_LeavesZeroForDefault(t *testing.T) {
	mgr, _ := New()
	if got := mgr.publishConcurrency; got != 0 {
		t.Errorf("publishConcurrency without WithPublishConcurrency = %d; want 0", got)
	}
}
