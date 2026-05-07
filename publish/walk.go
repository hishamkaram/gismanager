package publish

import (
	"context"
	"errors"
	"iter"
	"sync"

	"github.com/lukeroth/gdal"

	"github.com/hishamkaram/gismanager/v2/errs"
	"github.com/hishamkaram/gismanager/v2/internal/slogx"
)

// defaultPublishConcurrency caps how many GeoServer feature-type
// creations PublishAll dispatches in parallel. The publish step is
// HTTP-only (no GDAL CGo, no PostGIS connection from gismanager —
// GeoServer queries PostGIS itself), so it parallelizes cleanly. The
// upstream walk and the LayerToPostgis CopyLayer call stay serial
// because both touch the shared *gdal.DataSource and the underlying
// CGo handle is not reentrancy-safe across goroutines.
//
// Default 8 was picked from typical observation: GeoServer FeatureType
// creation is ~100-500ms per call, so 8 parallel takes ~50ms aggregate
// per layer instead of ~250ms. Higher concurrency mostly bottlenecks
// on GeoServer's own thread pool rather than gismanager. Expose as a
// functional option in v1.5+ if there's demand to tune.
const defaultPublishConcurrency = 8

// WalkItem is one yielded element of [(*Manager).Walk]: a
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
	// stamped via [(*Manager).NewLayer]. Use it directly with
	// helpers like [LayerToPostgis] / [PublishGeoserverLayer], or read
	// its embedded *gdal.Layer for OGR-level operations.
	Layer *Layer
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
// Per-file errors do NOT abort the iterator. If [(*Manager).OpenSource]
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
func (manager *Manager) Walk(ctx context.Context) iter.Seq2[WalkItem, error] {
	logger := manager.logger
	if logger == nil {
		logger = slogx.Default()
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
func walkLayers(manager *Manager, ctx context.Context, source *gdal.DataSource, path string, yield func(WalkItem, error) bool) bool {
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
// the per-file PostGIS-open cost.
//
// Concurrency model (since v1.4):
//
//   - Walk and LayerToPostgis are SERIAL across the whole call. Both
//     touch the shared *gdal.DataSource (the PostGIS target) and the
//     underlying CGo handle is not reentrancy-safe across goroutines.
//   - PublishGeoserverLayer (HTTP-only — GeoServer queries PostGIS
//     itself; gismanager just registers the feature type) runs in a
//     bounded worker pool of [defaultPublishConcurrency] goroutines.
//     Layers with publish-side latency (typical GeoServer
//     FeatureType.Create is 100-500ms) overlap, so a 50-layer publish
//     takes O(N/8 * latency) instead of O(N * latency).
//   - The publish-side ordering is therefore non-deterministic, but
//     that has never been part of PublishAll's contract.
//
// Error semantics (since v1.4):
//
//   - Setup failures abort the operation. If [(*Manager).OpenSource]
//     can't open the PostGIS datastore, PublishAll returns the wrapped
//     error directly without attempting any layer.
//   - Per-layer failures (walk errors, PostGIS load errors, GeoServer
//     publish errors) do NOT abort the loop. Each is logged via the
//     manager's logger AND collected for aggregation. Aggregation is
//     mutex-protected since publish goroutines run concurrently.
//   - The final return value is [errors.Join] of every collected
//     per-layer error, or nil if every layer succeeded.
//
// Callers can branch on the aggregated result with
// `errors.Is(err, errs.ErrGeoServerPublish)` / `errors.Is(err, errs.ErrPostGISConnect)`
// — the joined error walks through every collected failure, so any
// match in the chain returns true. To enumerate per-layer failures
// individually, type-assert via `errors.As(err, &gerr)` repeatedly or
// use a small unwrap walker:
//
//	if err := mgr.PublishAll(ctx); err != nil {
//	    var u interface{ Unwrap() []error }
//	    if errors.As(err, &u) {
//	        for _, e := range u.Unwrap() {
//	            // inspect each per-layer failure
//	        }
//	    }
//	}
//
// Pre-v1.4 callers that did `if err := mgr.PublishAll(ctx); err != nil`
// continue to work — the change strictly adds information. Callers that
// relied on PublishAll returning nil even when individual layers failed
// will now see those failures surface; that previous behavior was an
// acknowledged silent-failure mode (see the v1.3 doc comment) and the
// aggregated form is the documented path forward.
func (manager *Manager) PublishAll(ctx context.Context) error {
	logger := manager.logger
	if logger == nil {
		logger = slogx.Default()
	}
	target, err := manager.OpenSource(ctx, manager.Datastore.BuildConnectionString(), 1)
	if err != nil {
		return err
	}
	defer target.Destroy()

	// Pre-warm the workspace + datastore once, serially, BEFORE
	// dispatching the parallel publish goroutines. Without this, N
	// publish goroutines hit the Get-not-found branch of ensure*
	// simultaneously and race to Create. For workspaces that surfaces
	// as a clean 409 Conflict (geoserver.ErrConflict), but for
	// datastores GeoServer instead returns 500 with "already exists"
	// in the body — a non-idempotent wire-format quirk that's hard to
	// match defensively. Pre-warming sidesteps the race entirely:
	// every per-layer ensureWorkspace / ensureDatastore call finds
	// the resource on the Get and short-circuits. (The conflict
	// tolerance in ensureWorkspace is still useful as defense-in-depth
	// for callers invoking PublishGeoserverLayer directly without
	// going through PublishAll.)
	catalog, catErr := manager.GetGeoserverCatalog()
	if catErr != nil {
		return errs.NewGISError("PublishAll", "", errs.ErrGeoServerPublish, catErr)
	}
	if wsErr := ensureWorkspace(ctx, catalog, manager.Geoserver.WorkspaceName); wsErr != nil {
		return wsErr
	}
	if dsErr := ensureDatastore(ctx, catalog, manager.Geoserver.WorkspaceName, manager.Datastore); dsErr != nil {
		return dsErr
	}

	// Bounded worker pool for the publish step. The semaphore acts as
	// the concurrency gate; wg ensures we wait for every dispatched
	// publish before returning. errsMu protects perLayerErrs from
	// concurrent writes by the publish goroutines.
	//
	// Concurrency comes from [WithPublishConcurrency] when set (>0);
	// otherwise falls back to [defaultPublishConcurrency].
	concurrency := manager.publishConcurrency
	if concurrency <= 0 {
		concurrency = defaultPublishConcurrency
	}
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var errsMu sync.Mutex
	perLayerErrs := []error{}

	recordErr := func(err error) {
		errsMu.Lock()
		perLayerErrs = append(perLayerErrs, err)
		errsMu.Unlock()
	}

	for item, walkErr := range manager.Walk(ctx) {
		if walkErr != nil {
			logger.Error("walk", "err", walkErr)
			recordErr(walkErr)
			continue
		}
		newLayer, postgisErr := item.Layer.LayerToPostgis(target, manager, true)
		if postgisErr != nil {
			logger.Error("load to postgis", "file", item.Path, "err", postgisErr)
			recordErr(postgisErr)
			continue
		}
		if newLayer == nil || newLayer.Layer == nil {
			continue
		}

		// Dispatch the publish step into the worker pool. Acquiring
		// the semaphore here (BEFORE go func) means a producer that
		// outpaces the publishers blocks on the channel send rather
		// than spawning unbounded goroutines.
		sem <- struct{}{}
		wg.Add(1)
		go func(item WalkItem, newLayer *Layer) {
			defer wg.Done()
			defer func() { <-sem }()
			if pubErr := manager.PublishGeoserverLayer(ctx, newLayer); pubErr != nil {
				logger.Error("publish", "file", item.Path, "err", pubErr)
				recordErr(pubErr)
				return
			}
			logger.Info("published", "file", item.Path, "layer", newLayer.Name())
		}(item, newLayer)
	}

	// Drain the worker pool before reading the error slice. wg.Wait()
	// is the synchronization point that publishes every recordErr
	// write to the goroutine reading perLayerErrs below.
	wg.Wait()
	return errors.Join(perLayerErrs...)
}
