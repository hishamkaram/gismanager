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
