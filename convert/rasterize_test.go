package convert

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hishamkaram/gismanager/internal/errs"
)

func TestBuildRasterizeArgs(t *testing.T) {
	cases := []struct {
		name string
		opts []RasterizeOption
		want []string
	}{
		{
			name: "empty config yields empty args",
			opts: nil,
			want: nil,
		},
		{
			name: "format + output type",
			opts: []RasterizeOption{
				WithRasterizeFormat("GTiff"),
				WithRasterizeOutputType("Byte"),
			},
			want: []string{"-of", "GTiff", "-ot", "Byte"},
		},
		{
			name: "burn values emit one -burn per value",
			opts: []RasterizeOption{
				WithRasterizeBurnValues(1.0, 2.5, 3.75),
			},
			want: []string{"-burn", "1", "-burn", "2.5", "-burn", "3.75"},
		},
		{
			name: "attribute field",
			opts: []RasterizeOption{WithRasterizeAttribute("HEIGHT")},
			want: []string{"-a", "HEIGHT"},
		},
		{
			name: "layer + where filter",
			opts: []RasterizeOption{
				WithRasterizeLayer("admin"),
				WithRasterizeWhere("CONTINENT = 'Africa'"),
			},
			want: []string{"-l", "admin", "-where", "CONTINENT = 'Africa'"},
		},
		{
			name: "target resolution",
			opts: []RasterizeOption{WithRasterizeTargetResolution(0.01, 0.01)},
			want: []string{"-tr", "0.01", "0.01"},
		},
		{
			name: "output size in pixels",
			opts: []RasterizeOption{WithRasterizeOutputSize(512, 512)},
			want: []string{"-ts", "512", "512"},
		},
		{
			name: "output bounds (-te in source CRS)",
			opts: []RasterizeOption{
				WithRasterizeOutputBounds(-180, -90, 180, 90),
			},
			want: []string{"-te", "-180", "-90", "180", "90"},
		},
		{
			name: "creation options preserve order",
			opts: []RasterizeOption{
				WithRasterizeCreationOption("COMPRESS", "DEFLATE"),
				WithRasterizeCreationOption("TILED", "YES"),
			},
			want: []string{"-co", "COMPRESS=DEFLATE", "-co", "TILED=YES"},
		},
		{
			name: "raw options append at end",
			opts: []RasterizeOption{
				WithRasterizeFormat("GTiff"),
				WithRasterizeRawOptions("-init", "0", "-a_nodata", "255"),
			},
			want: []string{"-of", "GTiff", "-init", "0", "-a_nodata", "255"},
		},
		{
			name: "full pipeline: format + ot + attr + layer + where + tr + te + co",
			opts: []RasterizeOption{
				WithRasterizeFormat("GTiff"),
				WithRasterizeOutputType("Float32"),
				WithRasterizeAttribute("ELEV"),
				WithRasterizeLayer("contours"),
				WithRasterizeWhere("ELEV > 0"),
				WithRasterizeTargetResolution(0.001, 0.001),
				WithRasterizeOutputBounds(-10, -10, 10, 10),
				WithRasterizeCreationOption("COMPRESS", "ZSTD"),
			},
			want: []string{
				"-of", "GTiff",
				"-ot", "Float32",
				"-a", "ELEV",
				"-l", "contours",
				"-where", "ELEV > 0",
				"-tr", "0.001", "0.001",
				"-te", "-10", "-10", "10", "10",
				"-co", "COMPRESS=ZSTD",
			},
		},
		{
			name: "nil options tolerated",
			opts: []RasterizeOption{nil, WithRasterizeFormat("GTiff"), nil},
			want: []string{"-of", "GTiff"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := newRasterizeConfig(tc.opts)
			got := buildRasterizeArgs(cfg)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestWithRasterizeLogger_NilFallsBackToDefault(t *testing.T) {
	cfg := newRasterizeConfig([]RasterizeOption{WithRasterizeLogger(nil)})
	assert.NotNil(t, cfg.logger)
}

func TestRasterize_CtxCancelledFailsFast(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := Rasterize(ctx, "ignored.geojson", "ignored.tif")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}

func TestRasterize_OpenError_WrapsErrConvertFailed(t *testing.T) {
	err := Rasterize(context.Background(),
		"../testdata/__definitely_does_not_exist__.geojson",
		"/tmp/should_not_be_created.tif")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrConvertFailed))

	var gerr *errs.GISError
	assert.True(t, errors.As(err, &gerr))
	assert.Equal(t, "Rasterize", gerr.Op)
}
