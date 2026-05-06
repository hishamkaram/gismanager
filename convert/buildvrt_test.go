package convert

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hishamkaram/gismanager/internal/errs"
)

func TestBuildVRTArgs(t *testing.T) {
	cases := []struct {
		name string
		opts []VRTOption
		want []string
	}{
		{
			name: "empty config",
			opts: nil,
			want: nil,
		},
		{
			name: "resolution mode",
			opts: []VRTOption{WithVRTResolution("highest")},
			want: []string{"-resolution", "highest"},
		},
		{
			name: "user resolution",
			opts: []VRTOption{
				WithVRTResolution("user"),
				WithVRTUserResolution(30, 30),
			},
			want: []string{"-resolution", "user", "-tr", "30", "30"},
		},
		{
			name: "separate emits one band per input",
			opts: []VRTOption{WithVRTSeparate()},
			want: []string{"-separate"},
		},
		{
			name: "add alpha + resampling",
			opts: []VRTOption{
				WithVRTAddAlpha(),
				WithVRTResamplingAlg("bilinear"),
			},
			want: []string{"-addalpha", "-r", "bilinear"},
		},
		{
			name: "src + vrt nodata + hide",
			opts: []VRTOption{
				WithVRTSrcNoData("0,0,0"),
				WithVRTNoData("255"),
				WithVRTHideNoData(),
			},
			want: []string{
				"-srcnodata", "0,0,0",
				"-vrtnodata", "255",
				"-hidenodata",
			},
		},
		{
			name: "band selection",
			opts: []VRTOption{WithVRTBands(1, 2, 3)},
			want: []string{"-b", "1", "-b", "2", "-b", "3"},
		},
		{
			name: "allow projection difference",
			opts: []VRTOption{WithVRTAllowProjectionDifference()},
			want: []string{"-allow_projection_difference"},
		},
		{
			name: "raw options append at end",
			opts: []VRTOption{
				WithVRTResolution("highest"),
				WithVRTRawOptions("-input_file_list", "list.txt"),
			},
			want: []string{
				"-resolution", "highest",
				"-input_file_list", "list.txt",
			},
		},
		{
			name: "nil options tolerated",
			opts: []VRTOption{nil, WithVRTResolution("average"), nil},
			want: []string{"-resolution", "average"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := newVRTConfig(tc.opts)
			got := buildVRTArgs(cfg)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestWithVRTLogger_NilFallsBackToDefault(t *testing.T) {
	cfg := newVRTConfig([]VRTOption{WithVRTLogger(nil)})
	assert.NotNil(t, cfg.logger)
}

func TestBuildVRT_CtxCancelledFailsFast(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := BuildVRT(ctx, "ignored.vrt", []string{"a.tif"})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}

func TestBuildVRT_RejectsEmptySources(t *testing.T) {
	err := BuildVRT(context.Background(), "/tmp/out.vrt", nil)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrConvertFailed))

	var gerr *errs.GISError
	assert.True(t, errors.As(err, &gerr))
	assert.Equal(t, "BuildVRT", gerr.Op)
}

func TestBuildVRT_OpenError_WrapsErrConvertFailed(t *testing.T) {
	err := BuildVRT(context.Background(),
		"/tmp/should_not_be_created.vrt",
		[]string{"../testdata/__definitely_does_not_exist__.tif"})
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrConvertFailed))

	var gerr *errs.GISError
	assert.True(t, errors.As(err, &gerr))
	assert.Equal(t, "BuildVRT", gerr.Op)
}
