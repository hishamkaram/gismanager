package convert

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hishamkaram/gismanager/v2/errs"
)

func TestValidateGDALDriver(t *testing.T) {
	t.Run("empty name is allowed (format inference)", func(t *testing.T) {
		assert.NoError(t, validateGDALDriver(""))
	})
	t.Run("known raster driver", func(t *testing.T) {
		assert.NoError(t, validateGDALDriver("GTiff"))
	})
	t.Run("known COG driver", func(t *testing.T) {
		assert.NoError(t, validateGDALDriver("COG"))
	})
	t.Run("known vector driver", func(t *testing.T) {
		assert.NoError(t, validateGDALDriver("GPKG"))
	})
	t.Run("known GeoJSON driver", func(t *testing.T) {
		assert.NoError(t, validateGDALDriver("GeoJSON"))
	})
	t.Run("unknown driver returns error", func(t *testing.T) {
		err := validateGDALDriver("DEFINITELY_NOT_A_DRIVER")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DEFINITELY_NOT_A_DRIVER")
	})
}

// TestConvertVector_PreValidatesFormat closes the v1.2 silent-failure
// gap on the vector side: an unknown WithVectorFormat is rejected before
// the C call, with the standard *errs.GISError envelope.
func TestConvertVector_PreValidatesFormat(t *testing.T) {
	err := ConvertVector(context.Background(),
		"../testdata/neighborhood_names_gis.geojson",
		"/tmp/should_not_be_created.bogus",
		WithVectorFormat("BOGUS_FORMAT_NEVER_EXISTS"),
	)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrConvertFailed))

	var gerr *errs.GISError
	assert.True(t, errors.As(err, &gerr))
	assert.Equal(t, "ConvertVector", gerr.Op)
}

func TestConvertRaster_PreValidatesFormat(t *testing.T) {
	err := ConvertRaster(context.Background(),
		"/dev/null", "/tmp/out.tif",
		WithRasterFormat("BOGUS_FORMAT_NEVER_EXISTS"),
	)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrConvertFailed))

	var gerr *errs.GISError
	assert.True(t, errors.As(err, &gerr))
	assert.Equal(t, "ConvertRaster", gerr.Op)
}

func TestRasterize_PreValidatesFormat(t *testing.T) {
	err := Rasterize(context.Background(),
		"../testdata/neighborhood_names_gis.geojson",
		"/tmp/out.tif",
		WithRasterizeFormat("BOGUS_FORMAT_NEVER_EXISTS"),
	)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrConvertFailed))

	var gerr *errs.GISError
	assert.True(t, errors.As(err, &gerr))
	assert.Equal(t, "Rasterize", gerr.Op)
}

func TestDEMProcessing_PreValidatesFormat(t *testing.T) {
	err := DEMProcessing(context.Background(),
		"/dev/null", "/tmp/out.tif", "hillshade",
		WithDEMFormat("BOGUS_FORMAT_NEVER_EXISTS"),
	)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, errs.ErrConvertFailed))

	var gerr *errs.GISError
	assert.True(t, errors.As(err, &gerr))
	assert.Equal(t, "DEMProcessing", gerr.Op)
}
