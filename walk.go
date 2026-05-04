package gismanager

import (
	"context"
	"iter"

	"github.com/lukeroth/gdal"
)

// WalkItem is one yielded element of [(*ManagerConfig).Walk]: a
// (path, layer) pair the caller can consume.
//
// Convention: when the iterator's second yield value is a non-nil error,
// the WalkItem is the zero value — Path is "" and Layer is nil. When the
// error is nil, both fields are populated.
type WalkItem struct {
	// Path is the source file path the layer was discovered under
	// (e.g. "./data/cities.geojson"). For multi-layer files (most
	// GeoPackages) the same Path is yielded once per layer.
	Path string

	// Layer is the wrapped GDAL layer, with the manager's logger
	// stamped via [(*ManagerConfig).NewLayer]. Use it directly with
	// helpers like [LayerToPostgis] / [PublishGeoserverLayer], or read
	// its embedded *gdal.Layer for OGR-level operations.
	Layer *GdalLayer
}

// Walk returns an iterator over every layer of every supported GIS file
// under the manager's [SourceConfig.Path]. Iteration order is files-outer
// (in [GetGISFiles] order) and layers-inner (in OGR-index order).
//
// For each discovered file Walk opens the source once, yields one
// [WalkItem] per layer in OGR-index order, then `defer source.Destroy()`
// releases the underlying C handle before moving to the next file. Memory
// is therefore bounded to one file's worth of OGR state at a time, even
// for very large source directories.
//
// Per-file errors do NOT abort the iterator. If [(*ManagerConfig).OpenSource]
// fails for a path, Walk yields a zero [WalkItem] paired with the wrapped
// error and continues to the next file. Callers can:
//
//	for item, err := range mgr.Walk(ctx) {
//	    if err != nil {
//	        log.Printf("skip: %v", err)
//	        continue
//	    }
//	    // use item.Layer
//	}
//
// `break`-ing out of the for-range loop runs the deferred Destroy on the
// currently-open source, so early termination doesn't leak.
func (manager *ManagerConfig) Walk(ctx context.Context) iter.Seq2[WalkItem, error] {
	logger := manager.logger
	if logger == nil {
		logger = GetLogger()
	}
	return func(yield func(WalkItem, error) bool) {
		files, err := getGISFiles(manager.Source.Path, logger)
		if err != nil {
			yield(WalkItem{}, err)
			return
		}
		for _, path := range files {
			if err := ctx.Err(); err != nil {
				yield(WalkItem{}, err)
				return
			}
			source, srcErr := manager.OpenSource(ctx, path, 0)
			if srcErr != nil {
				if !yield(WalkItem{}, srcErr) {
					return
				}
				continue
			}
			// Bound the source's lifetime to this file: the C handle is
			// released before the next iteration starts, even if the
			// caller break-s out of the inner yield loop.
			done := walkLayers(manager, ctx, source, path, yield)
			source.Destroy()
			if done {
				return
			}
		}
	}
}

// walkLayers iterates the OGR layers in a single open source and yields
// each one via the iterator's yield function. Returns true when the
// caller signaled stop (yield returned false), letting the outer loop
// release the source and exit.
func walkLayers(manager *ManagerConfig, ctx context.Context, source *gdal.DataSource, path string, yield func(WalkItem, error) bool) bool {
	count := source.LayerCount()
	for i := 0; i < count; i++ {
		if err := ctx.Err(); err != nil {
			yield(WalkItem{}, err)
			return true
		}
		layer := source.LayerByIndex(i)
		wrapped := manager.NewLayer(&layer)
		if !yield(WalkItem{Path: path, Layer: wrapped}, nil) {
			return true
		}
	}
	return false
}

// PublishAll runs the full pipeline: walk → load each layer into PostGIS
// → publish each PostGIS table as a GeoServer feature type. It is the
// library equivalent of `cmd/gismanager`'s main loop.
//
// PublishAll opens the configured PostGIS datastore once at the top
// (re-using it across every file's layers), so the caller doesn't pay
// the per-file PostGIS-open cost. Per-file failures are logged via the
// manager's logger and the loop continues.
//
// The returned error is non-nil only when a setup step fails (e.g.
// the PostGIS datastore cannot be opened). Per-layer publish failures
// are logged and skipped — see the manager's logger output for
// diagnostics. Future versions may aggregate per-layer errors; the
// current shape matches `cmd/gismanager`'s pre-PR-3 behavior.
func (manager *ManagerConfig) PublishAll(ctx context.Context) error {
	logger := manager.logger
	if logger == nil {
		logger = GetLogger()
	}
	target, err := manager.OpenSource(ctx, manager.Datastore.BuildConnectionString(), 1)
	if err != nil {
		return err
	}
	defer target.Destroy()

	for item, err := range manager.Walk(ctx) {
		if err != nil {
			logger.Error("walk", "err", err)
			continue
		}
		newLayer, postgisErr := item.Layer.LayerToPostgis(target, manager, true)
		if postgisErr != nil {
			logger.Error("load to postgis", "file", item.Path, "err", postgisErr)
			continue
		}
		if newLayer == nil || newLayer.Layer == nil {
			continue
		}
		if pubErr := manager.PublishGeoserverLayer(ctx, newLayer); pubErr != nil {
			logger.Error("publish", "file", item.Path, "err", pubErr)
			continue
		}
		logger.Info("published", "file", item.Path, "layer", newLayer.Name())
	}
	return nil
}
