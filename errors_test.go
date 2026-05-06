package gismanager

import (
	"errors"
	"fmt"
	"testing"
)

func TestGISError_IsAndUnwrap(t *testing.T) {
	cause := errors.New("underlying boom")
	gerr := newGISError("OpenSource", "/tmp/foo.shp", ErrUnsupportedFormat, cause)

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
	gerr := newGISError("PublishGeoserverLayer", "ws/ds/lyr", ErrGeoServerPublish, cause)
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
// chain produced by [PublishGeoserverLayer] when an
// [ensureWorkspace] / [ensureDatastore] helper fails: each helper
// returns a *GISError (Op="ensureWorkspace" / "ensureDatastore",
// Sentinel=ErrGeoServerPublish), which PublishGeoserverLayer re-wraps
// in its own *GISError (Op="PublishGeoserverLayer",
// Sentinel=ErrGeoServerPublish). Callers must be able to:
//
//   - match the publish-failure sentinel at any level;
//   - reach the outer envelope via errors.As (Op shows the public
//     entry point);
//   - reach the inner helper-level envelope via Unwrap+errors.As (Op
//     shows which helper produced the failure for finer-grained
//     triage);
//   - reach the underlying upstream cause (e.g. *geoserver.APIError)
//     via errors.As walking the full chain.
func TestGISError_NestedChain(t *testing.T) {
	upstream := errors.New("simulated *geoserver.APIError 500")
	inner := newGISError("ensureWorkspace", "demo-ws", ErrGeoServerPublish, upstream)
	outer := newGISError("PublishGeoserverLayer", "demo-ws", ErrGeoServerPublish, inner)

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

// TestGISError_JoinedAggregation locks in the v1.4 [PublishAll] error
// contract: each per-layer failure is appended to a slice, and the final
// return value is errors.Join(slice...) — nil when empty, otherwise an
// aggregate that walks through every wrapped GISError. Callers must be
// able to:
//
//   - match any sentinel via errors.Is(joined, sentinel) regardless of
//     which entry in the slice carries it;
//   - extract the first matching *GISError via errors.As (typical
//     "give me one example to log" path);
//   - enumerate every per-layer failure via the unwrap-multiple
//     interface (`interface{ Unwrap() []error }`).
//
// The test constructs a representative joined error directly rather than
// driving a real PublishAll, since PublishAll requires PostGIS +
// GeoServer; the errors.Join + GISError mechanics this guards are
// independent of that infrastructure.
func TestGISError_JoinedAggregation(t *testing.T) {
	a := newGISError("PublishGeoserverLayer", "ws/ds/A", ErrGeoServerPublish, errors.New("500 Internal Server Error"))
	b := newGISError("LayerToPostgis", "ds-B", ErrPostGISConnect, errors.New("dial tcp: refused"))
	c := newGISError("PublishGeoserverLayer", "ws/ds/C", ErrGeoServerPublish, errors.New("conflict"))

	// Empty join is nil — locks in the "no-failures returns nil"
	// branch of PublishAll.
	if errors.Join() != nil {
		t.Errorf("errors.Join() with no args = non-nil; want nil")
	}

	joined := errors.Join(a, b, c)
	if joined == nil {
		t.Fatal("errors.Join(a,b,c) = nil; want non-nil")
	}

	if !errors.Is(joined, ErrGeoServerPublish) {
		t.Errorf("errors.Is(joined, ErrGeoServerPublish) = false; want true (a or c carries it)")
	}
	if !errors.Is(joined, ErrPostGISConnect) {
		t.Errorf("errors.Is(joined, ErrPostGISConnect) = false; want true (b carries it)")
	}
	if errors.Is(joined, ErrUnsupportedFormat) {
		t.Errorf("errors.Is(joined, ErrUnsupportedFormat) = true; want false")
	}

	// errors.As surfaces the FIRST *GISError in the joined chain (Go's
	// stdlib walks left-to-right). For PublishAll this is always
	// useful: callers usually just want one example to log.
	var first *GISError
	if !errors.As(joined, &first) {
		t.Fatal("errors.As did not extract a *GISError from the joined chain")
	}
	if first.Op != "PublishGeoserverLayer" || first.Source != "ws/ds/A" {
		t.Errorf("errors.As surfaced the wrong leaf: Op=%q Source=%q", first.Op, first.Source)
	}

	// Enumerate via the unwrap-multiple interface — the documented
	// path for "show me every failure" callers (e.g. CLI summaries
	// or batch dashboards).
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
			e:    newGISError("OpenSource", "", ErrUnsupportedFormat, nil),
			want: "gismanager: OpenSource: gismanager: unsupported format",
		},
		{
			name: "sentinel-with-cause-no-source",
			e:    newGISError("Ping", "", ErrPostGISConnect, errors.New("dial tcp: refused")),
			want: "gismanager: Ping: gismanager: postgis connect: dial tcp: refused",
		},
		{
			name: "source-no-cause",
			e:    newGISError("OpenSource", "/tmp/foo.bin", ErrUnsupportedFormat, nil),
			want: `gismanager: OpenSource "/tmp/foo.bin": gismanager: unsupported format`,
		},
		{
			name: "all-fields",
			e:    newGISError("Publish", "ws/ds/lyr", ErrGeoServerPublish, errors.New("500")),
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
