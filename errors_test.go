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
