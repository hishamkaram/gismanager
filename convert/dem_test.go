package convert

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hishamkaram/gismanager/v2/errs"
)

func TestBuildDEMArgs(t *testing.T) {
	cases := []struct {
		name string
		opts []DEMOption
		want []string
	}{
		{
			name: "empty config",
			opts: nil,
			want: nil,
		},
		{
			name: "format and output type",
			opts: []DEMOption{
				WithDEMFormat("GTiff"),
				WithDEMOutputType("Float32"),
			},
			want: []string{"-of", "GTiff", "-ot", "Float32"},
		},
		{
			name: "algorithm",
			opts: []DEMOption{WithDEMAlgorithm("ZevenbergenThorne")},
			want: []string{"-alg", "ZevenbergenThorne"},
		},
		{
			name: "z-factor and scale",
			opts: []DEMOption{
				WithDEMZFactor(2.5),
				WithDEMScale(111120),
			},
			want: []string{"-z", "2.5", "-s", "111120"},
		},
		{
			name: "hillshade params (azimuth + altitude + combined)",
			opts: []DEMOption{
				WithDEMAzimuth(135),
				WithDEMAltitude(60),
				WithDEMCombined(),
			},
			want: []string{
				"-az", "135",
				"-alt", "60",
				"-combined",
			},
		},
		{
			name: "multidirectional hillshade",
			opts: []DEMOption{WithDEMMultidirectional()},
			want: []string{"-multidirectional"},
		},
		{
			name: "creation options preserve order",
			opts: []DEMOption{
				WithDEMCreationOption("COMPRESS", "DEFLATE"),
				WithDEMCreationOption("TILED", "YES"),
			},
			want: []string{
				"-co", "COMPRESS=DEFLATE",
				"-co", "TILED=YES",
			},
		},
		{
			name: "raw options append at end",
			opts: []DEMOption{
				WithDEMFormat("GTiff"),
				WithDEMRawOptions("-p", "0", "-igor"),
			},
			want: []string{"-of", "GTiff", "-p", "0", "-igor"},
		},
		{
			name: "nil options tolerated",
			opts: []DEMOption{nil, WithDEMFormat("GTiff"), nil},
			want: []string{"-of", "GTiff"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := newDEMConfig(tc.opts)
			got := buildDEMArgs(cfg)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestWithDEMLogger_NilFallsBackToDefault(t *testing.T) {
	cfg := newDEMConfig([]DEMOption{WithDEMLogger(nil)})
	assert.NotNil(t, cfg.logger)
}

func TestDEMProcessing_CtxCancelledFailsFast(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := DEMProcessing(ctx, "ignored.tif", "ignored.hs.tif", "hillshade")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}

func TestDEMProcessing_RejectsEmptyMode(t *testing.T) {
	err := DEMProcessing(context.Background(),
		"/dev/null", "/tmp/out.tif", "")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrConvertFailed))
	assert.Contains(t, err.Error(), "mode")
}

func TestDEMProcessing_ColorReliefRequiresColorFile(t *testing.T) {
	err := DEMProcessing(context.Background(),
		"/dev/null", "/tmp/out.tif", "color-relief")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrConvertFailed))
	assert.Contains(t, err.Error(), "WithDEMColorFile")
}

func TestDEMProcessing_OpenError_WrapsErrConvertFailed(t *testing.T) {
	err := DEMProcessing(context.Background(),
		"../testdata/__definitely_does_not_exist__.tif",
		"/tmp/should_not_be_created.hs.tif",
		"hillshade")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrConvertFailed))

	var gerr *errs.GISError
	assert.True(t, errors.As(err, &gerr))
	assert.Equal(t, "DEMProcessing", gerr.Op)
}
