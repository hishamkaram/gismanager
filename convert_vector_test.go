package gismanager

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBuildVectorTranslateArgs is a table-driven unit test that locks in
// the option -> ogr2ogr-arg mapping. CGo isn't exercised here — the goal
// is to catch any drift between the WithVector* helpers and the actual
// GDALVectorTranslate flag set without needing the GeoServer/PostGIS
// stack.
func TestBuildVectorTranslateArgs(t *testing.T) {
	cases := []struct {
		name string
		opts []VectorConvertOption
		want []string
	}{
		{
			name: "no options yields empty args",
			opts: nil,
			want: nil,
		},
		{
			name: "format only",
			opts: []VectorConvertOption{WithVectorFormat("GPKG")},
			want: []string{"-f", "GPKG"},
		},
		{
			name: "overwrite",
			opts: []VectorConvertOption{WithVectorOverwrite()},
			want: []string{"-overwrite"},
		},
		{
			name: "source + target SRS",
			opts: []VectorConvertOption{
				WithVectorSourceSRS("EPSG:4326"),
				WithVectorTargetSRS("EPSG:3857"),
			},
			want: []string{"-s_srs", "EPSG:4326", "-t_srs", "EPSG:3857"},
		},
		{
			name: "bbox emits four float args after -spat",
			opts: []VectorConvertOption{
				WithVectorBoundingBox(-10.5, -20, 30.25, 45.75),
			},
			want: []string{"-spat", "-10.5", "-20", "30.25", "45.75"},
		},
		{
			name: "where clause",
			opts: []VectorConvertOption{
				WithVectorWhere("CONTINENT = 'Africa'"),
			},
			want: []string{"-where", "CONTINENT = 'Africa'"},
		},
		{
			name: "simplify with non-zero tolerance",
			opts: []VectorConvertOption{WithVectorSimplify(0.001)},
			want: []string{"-simplify", "0.001"},
		},
		{
			name: "simplify with zero tolerance is dropped",
			opts: []VectorConvertOption{WithVectorSimplify(0)},
			want: nil,
		},
		{
			name: "select fields",
			opts: []VectorConvertOption{
				WithVectorSelectFields("NAME", "POP_EST", "CONTINENT"),
			},
			want: []string{"-select", "NAME,POP_EST,CONTINENT"},
		},
		{
			name: "layer name",
			opts: []VectorConvertOption{WithVectorLayerName("countries_3857")},
			want: []string{"-nln", "countries_3857"},
		},
		{
			name: "raw options append at end",
			opts: []VectorConvertOption{
				WithVectorFormat("GPKG"),
				WithVectorRawOptions("-lco", "FID=fid", "-skipfailures"),
			},
			want: []string{"-f", "GPKG", "-lco", "FID=fid", "-skipfailures"},
		},
		{
			name: "full pipeline: reproject + bbox + where + simplify + select + rename",
			opts: []VectorConvertOption{
				WithVectorFormat("GPKG"),
				WithVectorOverwrite(),
				WithVectorTargetSRS("EPSG:3857"),
				WithVectorBoundingBox(-20, 0, 60, 40),
				WithVectorWhere("CONTINENT = 'Africa'"),
				WithVectorSimplify(100),
				WithVectorSelectFields("NAME", "POP_EST"),
				WithVectorLayerName("africa_3857"),
			},
			want: []string{
				"-f", "GPKG",
				"-overwrite",
				"-t_srs", "EPSG:3857",
				"-spat", "-20", "0", "60", "40",
				"-where", "CONTINENT = 'Africa'",
				"-simplify", "100",
				"-select", "NAME,POP_EST",
				"-nln", "africa_3857",
			},
		},
		{
			name: "nil options are tolerated",
			opts: []VectorConvertOption{
				nil,
				WithVectorFormat("GPKG"),
				nil,
			},
			want: []string{"-f", "GPKG"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := newVectorConvertConfig(tc.opts)
			got := buildVectorTranslateArgs(cfg)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestWithVectorLogger_NilFallsBackToDefault confirms passing nil through
// WithVectorLogger doesn't leave cfg.logger nil — the helper falls back
// to GetLogger() so downstream callers can always emit structured logs.
func TestWithVectorLogger_NilFallsBackToDefault(t *testing.T) {
	cfg := newVectorConvertConfig([]VectorConvertOption{WithVectorLogger(nil)})
	assert.NotNil(t, cfg.logger, "logger must never be nil after option apply")
}

// TestWithVectorLogger_CustomHandlerCaptured exercises the custom-logger
// path: a JSON handler routed at a *bytes.Buffer captures structured
// output, so callers can verify ConvertVector emits the expected records.
func TestWithVectorLogger_CustomHandlerCaptured(t *testing.T) {
	var buf bytes.Buffer
	custom := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	cfg := newVectorConvertConfig([]VectorConvertOption{WithVectorLogger(custom)})
	cfg.logger.Debug("test", "key", "value")
	assert.Contains(t, buf.String(), `"key":"value"`)
}

// TestConvertVector_CtxCancelledFailsFast verifies the ctx-honor at the
// function boundary: a pre-canceled context surfaces as ctx.Err() before
// any GDAL call, so callers can use a context to abort batches between
// conversions.
func TestConvertVector_CtxCancelledFailsFast(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := ConvertVector(ctx, "ignored.geojson", "ignored.gpkg")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled),
		"expected context.Canceled, got %v", err)
}

// TestConvertVector_OpenError_WrapsErrConvertFailed locks in the error
// envelope: a missing source surfaces as *GISError wrapping
// ErrConvertFailed, so callers can branch on the sentinel without
// scraping strings.
func TestConvertVector_OpenError_WrapsErrConvertFailed(t *testing.T) {
	err := ConvertVector(context.Background(),
		"./testdata/__definitely_does_not_exist__.geojson",
		"/tmp/gismanager_should_not_be_created.gpkg")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrConvertFailed),
		"expected ErrConvertFailed, got %v", err)

	var gerr *GISError
	assert.True(t, errors.As(err, &gerr), "expected *GISError envelope")
	assert.Equal(t, "ConvertVector", gerr.Op)
}
