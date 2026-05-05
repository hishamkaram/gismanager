package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseBBox(t *testing.T) {
	cases := []struct {
		in   string
		ok   bool
		want [4]float64
	}{
		{"-25,-40,60,40", true, [4]float64{-25, -40, 60, 40}},
		{" -180.5 , -90 , 180.5 , 90 ", true, [4]float64{-180.5, -90, 180.5, 90}},
		{"1,2,3", false, [4]float64{}},
		{"a,b,c,d", false, [4]float64{}},
		{"", false, [4]float64{}},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			minX, minY, maxX, maxY, err := parseBBox(tc.in)
			if !tc.ok {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.want[0], minX)
			assert.Equal(t, tc.want[1], minY)
			assert.Equal(t, tc.want[2], maxX)
			assert.Equal(t, tc.want[3], maxY)
		})
	}
}

func TestParseBands(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		got, err := parseBands("1,2,3")
		assert.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, got)
	})
	t.Run("with whitespace", func(t *testing.T) {
		got, err := parseBands(" 1 , 2 , 3 ")
		assert.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, got)
	})
	t.Run("bad int", func(t *testing.T) {
		_, err := parseBands("1,not-a-number,3")
		assert.Error(t, err)
	})
}

func TestParseRes(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		x, y, err := parseRes("30,30")
		assert.NoError(t, err)
		assert.Equal(t, 30.0, x)
		assert.Equal(t, 30.0, y)
	})
	t.Run("non-square", func(t *testing.T) {
		x, y, err := parseRes("0.0001,0.00005")
		assert.NoError(t, err)
		assert.Equal(t, 0.0001, x)
		assert.Equal(t, 0.00005, y)
	})
	t.Run("missing comma", func(t *testing.T) {
		_, _, err := parseRes("30")
		assert.Error(t, err)
	})
}

func TestRasterCreationOptions_AccumulatesAcrossSet(t *testing.T) {
	var r rasterCreationOptions
	assert.NoError(t, r.Set("COMPRESS=DEFLATE"))
	assert.NoError(t, r.Set("BLOCKSIZE=512"))
	assert.NoError(t, r.Set("OVERVIEW_RESAMPLING=NEAREST"))
	assert.Equal(t, []string{"COMPRESS=DEFLATE", "BLOCKSIZE=512", "OVERVIEW_RESAMPLING=NEAREST"}, []string(r))
	// String() formats as space-joined for flag-help display.
	assert.Equal(t, "COMPRESS=DEFLATE BLOCKSIZE=512 OVERVIEW_RESAMPLING=NEAREST", r.String())
}

func TestRun_RejectsMissingArgs(t *testing.T) {
	t.Run("no mode", func(t *testing.T) {
		err := run([]string{"-src", "in.shp", "-dst", "out.gpkg"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "-mode")
	})
	t.Run("no src", func(t *testing.T) {
		err := run([]string{"-mode", "vector", "-dst", "out.gpkg"})
		assert.Error(t, err)
	})
	t.Run("no dst", func(t *testing.T) {
		err := run([]string{"-mode", "vector", "-src", "in.shp"})
		assert.Error(t, err)
	})
	t.Run("unknown mode", func(t *testing.T) {
		err := run([]string{"-mode", "satellite", "-src", "in", "-dst", "out"})
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "-mode must be"))
	})
}

func TestRun_RejectsRasterReprojectWithoutBothSRS(t *testing.T) {
	// Only -t-srs supplied; gisconvert requires both -s-srs and -t-srs.
	err := run([]string{
		"-mode", "raster",
		"-src", "in.tif",
		"-dst", "out.tif",
		"-t-srs", "EPSG:3857",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "-s-srs and -t-srs")
}
