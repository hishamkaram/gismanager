package gismanager

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildTranslateArgs(t *testing.T) {
	cases := []struct {
		name string
		opts []RasterConvertOption
		want []string
	}{
		{
			name: "empty config",
			opts: nil,
			want: nil,
		},
		{
			name: "format only",
			opts: []RasterConvertOption{WithRasterFormat("COG")},
			want: []string{"-of", "COG"},
		},
		{
			name: "single creation option",
			opts: []RasterConvertOption{
				WithRasterCreationOption("COMPRESS", "DEFLATE"),
			},
			want: []string{"-co", "COMPRESS=DEFLATE"},
		},
		{
			name: "multiple creation options preserve order",
			opts: []RasterConvertOption{
				WithRasterCreationOption("COMPRESS", "DEFLATE"),
				WithRasterCreationOption("BLOCKSIZE", "512"),
				WithRasterCreationOption("OVERVIEW_RESAMPLING", "NEAREST"),
			},
			want: []string{
				"-co", "COMPRESS=DEFLATE",
				"-co", "BLOCKSIZE=512",
				"-co", "OVERVIEW_RESAMPLING=NEAREST",
			},
		},
		{
			name: "band selection emits one -b per band",
			opts: []RasterConvertOption{WithRasterBands(1, 2, 3)},
			want: []string{"-b", "1", "-b", "2", "-b", "3"},
		},
		{
			name: "resampling alg",
			opts: []RasterConvertOption{WithRasterResamplingAlg("bilinear")},
			want: []string{"-r", "bilinear"},
		},
		{
			name: "output bounds use -projwin (ulx uly lrx lry — y-flip)",
			opts: []RasterConvertOption{
				WithRasterOutputBounds(-180, -90, 180, 90),
			},
			want: []string{"-projwin", "-180", "90", "180", "-90"},
		},
		{
			name: "target resolution",
			opts: []RasterConvertOption{
				WithRasterTargetResolution(0.0001, 0.0001),
			},
			want: []string{"-tr", "0.0001", "0.0001"},
		},
		{
			name: "raw options append at end",
			opts: []RasterConvertOption{
				WithRasterFormat("GTiff"),
				WithRasterRawOptions("-a_srs", "EPSG:4326"),
			},
			want: []string{"-of", "GTiff", "-a_srs", "EPSG:4326"},
		},
		{
			name: "ToCOG defaults expand correctly when applied",
			opts: []RasterConvertOption{
				WithRasterFormat("COG"),
				WithRasterCreationOption("COMPRESS", "DEFLATE"),
				WithRasterCreationOption("BLOCKSIZE", "512"),
				WithRasterCreationOption("OVERVIEW_RESAMPLING", "NEAREST"),
			},
			want: []string{
				"-of", "COG",
				"-co", "COMPRESS=DEFLATE",
				"-co", "BLOCKSIZE=512",
				"-co", "OVERVIEW_RESAMPLING=NEAREST",
			},
		},
		{
			name: "nil options tolerated",
			opts: []RasterConvertOption{nil, WithRasterFormat("PNG"), nil},
			want: []string{"-of", "PNG"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := newRasterConvertConfig(tc.opts)
			got := buildTranslateArgs(cfg)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestWithRasterLogger_NilFallsBackToDefault(t *testing.T) {
	cfg := newRasterConvertConfig([]RasterConvertOption{WithRasterLogger(nil)})
	assert.NotNil(t, cfg.logger)
}

func TestConvertRaster_CtxCancelledFailsFast(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := ConvertRaster(ctx, "ignored.tif", "ignored.cog")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}

func TestToCOG_CtxCancelledFailsFast(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := ToCOG(ctx, "ignored.tif", "ignored.cog")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}

func TestConvertRaster_OpenError_WrapsErrConvertFailed(t *testing.T) {
	err := ConvertRaster(context.Background(),
		"./testdata/__definitely_does_not_exist__.tif",
		"/tmp/should_not_be_created.cog")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrConvertFailed),
		"expected ErrConvertFailed, got %v", err)

	var gerr *GISError
	assert.True(t, errors.As(err, &gerr))
	assert.Equal(t, "ConvertRaster", gerr.Op)
}
