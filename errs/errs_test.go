package errs

import (
	"errors"
	"fmt"
	"testing"
)

// These tests exercise the typed-error machinery directly, against the
// internal/errs package. The root gismanager package's errors_test.go
// keeps a single alias-chain confirmation test that ensures the v1.x
// public API surface (gismanager.GISError, gismanager.ErrConvertFailed,
// etc.) continues to resolve to these same instances.

func TestGISError_IsAndUnwrap(t *testing.T) {
	cause := errors.New("underlying boom")
	gerr := NewGISError("OpenSource", "/tmp/foo.shp", ErrUnsupportedFormat, cause)

	if !errors.Is(gerr, ErrUnsupportedFormat) {
		t.Errorf("errors.Is(gerr, ErrUnsupportedFormat) = false, want true")
	}
	if errors.Is(gerr, ErrInvalidLayer) {
		t.Errorf("errors.Is(gerr, ErrInvalidLayer) = true, want false")
	}
	if !errors.Is(gerr, cause) {
		t.Errorf("errors.Is(gerr, cause) = false, want true (Unwrap)")
	}
}

func TestGISError_As(t *testing.T) {
	cause := errors.New("underlying boom")
	gerr := NewGISError("PublishGeoserverLayer", "ws/ds/lyr", ErrGeoServerPublish, cause)
	wrapped := fmt.Errorf("outer: %w", gerr)

	var got *GISError
	if !errors.As(wrapped, &got) {
		t.Fatalf("errors.As did not match")
	}
	if got.Op != "PublishGeoserverLayer" {
		t.Errorf("Op = %q, want PublishGeoserverLayer", got.Op)
	}
	if got.Source != "ws/ds/lyr" {
		t.Errorf("Source = %q, want ws/ds/lyr", got.Source)
	}
}

// TestGISError_NestedChain locks in the contract for the nested-*GISError
// chain produced by the publish pipeline when an ensureWorkspace /
// ensureDatastore helper fails: each helper returns a *GISError, which
// PublishGeoserverLayer re-wraps in its own *GISError. Callers must be
// able to:
//
//   - match the publish-failure sentinel at any level;
//   - reach the outer envelope via errors.As;
//   - reach the inner helper-level envelope via Unwrap+errors.As;
//   - reach the underlying upstream cause via errors.As walking the chain.
func TestGISError_NestedChain(t *testing.T) {
	upstream := errors.New("simulated *geoserver.APIError 500")
	inner := NewGISError("ensureWorkspace", "demo-ws", ErrGeoServerPublish, upstream)
	outer := NewGISError("PublishGeoserverLayer", "demo-ws", ErrGeoServerPublish, inner)

	if !errors.Is(outer, ErrGeoServerPublish) {
		t.Errorf("errors.Is(outer, ErrGeoServerPublish) = false; want true")
	}
	if !errors.Is(outer, upstream) {
		t.Errorf("errors.Is(outer, upstream) = false; want true (chain walk)")
	}

	var top *GISError
	if !errors.As(outer, &top) {
		t.Fatalf("errors.As(outer, &top) did not match")
	}
	if top.Op != "PublishGeoserverLayer" {
		t.Errorf("outer Op = %q; want PublishGeoserverLayer", top.Op)
	}

	var helper *GISError
	if !errors.As(top.Unwrap(), &helper) {
		t.Fatalf("errors.As(top.Unwrap(), &helper) did not match a nested *GISError")
	}
	if helper.Op != "ensureWorkspace" {
		t.Errorf("inner Op = %q; want ensureWorkspace", helper.Op)
	}
	if helper.Source != "demo-ws" {
		t.Errorf("inner Source = %q; want demo-ws", helper.Source)
	}
	if !errors.Is(helper.Cause, upstream) {
		t.Errorf("inner Cause = %v; want errors.Is match against upstream", helper.Cause)
	}
}

// TestGISError_JoinedAggregation locks in the v1.4 PublishAll error
// contract: each per-layer failure is appended to a slice, and the
// final return value is errors.Join(slice...). Empty join returns nil;
// populated join supports errors.Is for every constituent sentinel,
// errors.As surfaces the first *GISError, and the multi-Unwrap
// interface enumerates every per-layer failure.
func TestGISError_JoinedAggregation(t *testing.T) {
	a := NewGISError("PublishGeoserverLayer", "ws/ds/A", ErrGeoServerPublish, errors.New("500 Internal Server Error"))
	b := NewGISError("LayerToPostgis", "ds-B", ErrPostGISConnect, errors.New("dial tcp: refused"))
	c := NewGISError("PublishGeoserverLayer", "ws/ds/C", ErrGeoServerPublish, errors.New("conflict"))

	if errors.Join() != nil {
		t.Errorf("errors.Join() with no args = non-nil; want nil")
	}

	joined := errors.Join(a, b, c)
	if joined == nil {
		t.Fatal("errors.Join(a,b,c) = nil; want non-nil")
	}

	if !errors.Is(joined, ErrGeoServerPublish) {
		t.Errorf("errors.Is(joined, ErrGeoServerPublish) = false; want true")
	}
	if !errors.Is(joined, ErrPostGISConnect) {
		t.Errorf("errors.Is(joined, ErrPostGISConnect) = false; want true")
	}
	if errors.Is(joined, ErrUnsupportedFormat) {
		t.Errorf("errors.Is(joined, ErrUnsupportedFormat) = true; want false")
	}

	var first *GISError
	if !errors.As(joined, &first) {
		t.Fatal("errors.As did not extract a *GISError from the joined chain")
	}
	if first.Op != "PublishGeoserverLayer" || first.Source != "ws/ds/A" {
		t.Errorf("errors.As surfaced the wrong leaf: Op=%q Source=%q", first.Op, first.Source)
	}

	var multi interface{ Unwrap() []error }
	if !errors.As(joined, &multi) {
		t.Fatal("errors.As to the multi-Unwrap interface failed")
	}
	if got := len(multi.Unwrap()); got != 3 {
		t.Errorf("Unwrap() returned %d errors; want 3", got)
	}
}

func TestGISError_ErrorFormatsByCombination(t *testing.T) {
	cases := []struct {
		name string
		e    *GISError
		want string
	}{
		{
			name: "sentinel-only",
			e:    NewGISError("OpenSource", "", ErrUnsupportedFormat, nil),
			want: "gismanager: OpenSource: gismanager: unsupported format",
		},
		{
			name: "sentinel-with-cause-no-source",
			e:    NewGISError("Ping", "", ErrPostGISConnect, errors.New("dial tcp: refused")),
			want: "gismanager: Ping: gismanager: postgis connect: dial tcp: refused",
		},
		{
			name: "source-no-cause",
			e:    NewGISError("OpenSource", "/tmp/foo.bin", ErrUnsupportedFormat, nil),
			want: `gismanager: OpenSource "/tmp/foo.bin": gismanager: unsupported format`,
		},
		{
			name: "all-fields",
			e:    NewGISError("Publish", "ws/ds/lyr", ErrGeoServerPublish, errors.New("500")),
			want: `gismanager: Publish "ws/ds/lyr": gismanager: geoserver publish: 500`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.e.Error(); got != tc.want {
				t.Errorf("\n got: %q\nwant: %q", got, tc.want)
			}
		})
	}
}
