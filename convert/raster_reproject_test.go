package convert

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hishamkaram/gismanager/v2/errs"
)

func TestBuildWarpArgs(t *testing.T) {
	cases := []struct {
		name   string
		opts   []RasterOption
		srcSRS string
		dstSRS string
		want   []string
	}{
		{
			name:   "minimal: just src and dst SRS",
			srcSRS: "EPSG:32618",
			dstSRS: "EPSG:3857",
			want:   []string{"-s_srs", "EPSG:32618", "-t_srs", "EPSG:3857"},
		},
		{
			name: "with format and creation option",
			opts: []RasterOption{
				WithRasterFormat("GTiff"),
				WithRasterCreationOption("COMPRESS", "DEFLATE"),
			},
			srcSRS: "EPSG:4326",
			dstSRS: "EPSG:3857",
			want: []string{
				"-of", "GTiff",
				"-s_srs", "EPSG:4326", "-t_srs", "EPSG:3857",
				"-co", "COMPRESS=DEFLATE",
			},
		},
		{
			name: "output bounds use -te (target CRS) not -projwin",
			opts: []RasterOption{
				WithRasterOutputBounds(-180, -90, 180, 90),
			},
			srcSRS: "EPSG:4326",
			dstSRS: "EPSG:3857",
			want: []string{
				"-s_srs", "EPSG:4326", "-t_srs", "EPSG:3857",
				"-te", "-180", "-90", "180", "90",
			},
		},
		{
			name: "resampling + target resolution",
			opts: []RasterOption{
				WithRasterResamplingAlg("bilinear"),
				WithRasterTargetResolution(30, 30),
			},
			srcSRS: "EPSG:32618",
			dstSRS: "EPSG:3857",
			want: []string{
				"-s_srs", "EPSG:32618", "-t_srs", "EPSG:3857",
				"-r", "bilinear",
				"-tr", "30", "30",
			},
		},
		{
			name: "cutline emits -cutline + -cl + -crop_to_cutline",
			opts: []RasterOption{
				WithRasterCutline("clip.geojson", "outline"),
			},
			srcSRS: "EPSG:4326",
			dstSRS: "EPSG:3857",
			want: []string{
				"-s_srs", "EPSG:4326", "-t_srs", "EPSG:3857",
				"-cutline", "clip.geojson",
				"-cl", "outline",
				"-crop_to_cutline",
			},
		},
		{
			name: "cutline without explicit layer omits -cl",
			opts: []RasterOption{
				WithRasterCutline("clip.geojson", ""),
			},
			srcSRS: "EPSG:4326",
			dstSRS: "EPSG:3857",
			want: []string{
				"-s_srs", "EPSG:4326", "-t_srs", "EPSG:3857",
				"-cutline", "clip.geojson",
				"-crop_to_cutline",
			},
		},
		{
			name: "raw options append at end",
			opts: []RasterOption{
				WithRasterRawOptions("-overwrite", "-multi"),
			},
			srcSRS: "EPSG:4326",
			dstSRS: "EPSG:3857",
			want: []string{
				"-s_srs", "EPSG:4326", "-t_srs", "EPSG:3857",
				"-overwrite", "-multi",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := newRasterConfig(tc.opts)
			got := buildWarpArgs(cfg, tc.srcSRS, tc.dstSRS)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestReprojectRaster_CtxCancelledFailsFast(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := ReprojectRaster(ctx, "ignored.tif", "ignored.warped.tif",
		"EPSG:32618", "EPSG:3857")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.Canceled))
}

func TestReprojectRaster_RejectsEmptySRS(t *testing.T) {
	t.Run("empty srcSRS", func(t *testing.T) {
		err := ReprojectRaster(context.Background(),
			"/dev/null", "/tmp/out.tif", "", "EPSG:3857")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, errs.ErrConvertFailed))
	})
	t.Run("empty dstSRS", func(t *testing.T) {
		err := ReprojectRaster(context.Background(),
			"/dev/null", "/tmp/out.tif", "EPSG:4326", "")
		assert.Error(t, err)
		assert.True(t, errors.Is(err, errs.ErrConvertFailed))
	})
}

func TestReprojectRaster_OpenError_WrapsErrConvertFailed(t *testing.T) {
	err := ReprojectRaster(context.Background(),
		"../testdata/__definitely_does_not_exist__.tif",
		"/tmp/should_not_be_created.warped.tif",
		"EPSG:32618", "EPSG:3857")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrConvertFailed))

	var gerr *errs.GISError
	assert.True(t, errors.As(err, &gerr))
	assert.Equal(t, "ReprojectRaster", gerr.Op)
}
