package gismanager

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/datastores"
	"github.com/hishamkaram/geoserver/v2/rest/featuretypes"
	"github.com/hishamkaram/geoserver/v2/rest/workspaces"
	"github.com/lukeroth/gdal"
)

// GdalLayer wraps a *gdal.Layer with the helper methods gismanager uses
// (publish, copy-to-PostGIS, schema introspection).
//
// The logger field is intentionally lowercase: callers should construct
// GdalLayers via [(*ManagerConfig).NewLayer] so the manager's configured
// logger is stamped on automatically. The zero-value form
// (`GdalLayer{Layer: l}`) is still supported for back-compat — methods
// that need a logger fall back to [GetLogger] when the field is nil.
type GdalLayer struct {
	*gdal.Layer
	logger *slog.Logger
}

// NewLayer wraps a *gdal.Layer for use with gismanager's helpers,
// stamping the manager's configured logger on the result. Prefer this
// over the zero-value form so per-manager logger configuration
// (custom handler, attached attrs, etc.) reaches the layer's helper
// methods.
func (manager *ManagerConfig) NewLayer(l *gdal.Layer) *GdalLayer {
	return &GdalLayer{Layer: l, logger: manager.logger}
}

// loggerOrDefault returns the layer's stamped logger or, if nil, the
// project default. Used by methods that emit structured logs but
// must keep working on zero-value GdalLayers from pre-NewLayer
// callers.
func (layer *GdalLayer) loggerOrDefault() *slog.Logger {
	if layer == nil || layer.logger == nil {
		return GetLogger()
	}
	return layer.logger
}

// LayerField is one column in a layer's schema (geometry or attribute).
type LayerField struct {
	Name string
	Type string
}

// PublishGeoserverLayer publishes the given GDAL layer as a GeoServer feature
// type, ensuring the configured workspace and PostGIS datastore exist first.
//
// The flow uses the v2 client's "Get + ErrNotFound" idiom (v2 has no Exists
// methods): for each resource we try to Get it; only if Get returns
// [geoserver.ErrNotFound] do we Create it. Any other error short-circuits.
// PublishGeoserverLayer publishes the given GDAL layer as a GeoServer
// feature type, ensuring the configured workspace and PostGIS datastore
// exist first.
//
// Errors are wrapped with [ErrGeoServerPublish]; the underlying
// *geoserver.APIError is recoverable via [errors.As] for fields and
// [errors.Is] against the v2 sentinels (e.g. ErrConflict, ErrServerError).
func (manager *ManagerConfig) PublishGeoserverLayer(ctx context.Context, layer *GdalLayer) error {
	catalog, err := manager.GetGeoserverCatalog()
	if err != nil {
		manager.logger.Error("build geoserver client", "err", err)
		return newGISError("PublishGeoserverLayer", "", ErrGeoServerPublish, err)
	}

	ws := manager.Geoserver.WorkspaceName
	if err := ensureWorkspace(ctx, catalog, ws); err != nil {
		manager.logger.Error("ensure workspace", "workspace", ws, "err", err)
		return newGISError("PublishGeoserverLayer", ws, ErrGeoServerPublish, err)
	}

	dsName := manager.Datastore.Name
	if err := ensureDatastore(ctx, catalog, ws, manager.Datastore); err != nil {
		manager.logger.Error("ensure datastore", "workspace", ws, "datastore", dsName, "err", err)
		return newGISError("PublishGeoserverLayer", ws+"/"+dsName, ErrGeoServerPublish, err)
	}

	layerName := layer.Name()
	scoped := catalog.FeatureTypes.InWorkspace(ws).InDatastore(dsName)
	if _, err := scoped.Get(ctx, layerName); err == nil {
		// Already published — idempotent no-op.
		manager.logger.Debug("publish feature type: already exists",
			"workspace", ws, "datastore", dsName, "layer", layerName)
		return nil
	} else if !errors.Is(err, geoserver.ErrNotFound) {
		manager.logger.Error("get feature type", "workspace", ws, "datastore", dsName, "layer", layerName, "err", err)
		return newGISError("PublishGeoserverLayer", ws+"/"+dsName+"/"+layerName, ErrGeoServerPublish, err)
	}
	ft := &featuretypes.FeatureType{
		Name:       layerName,
		NativeName: layerName,
		Enabled:    true,
	}
	if err := scoped.Create(ctx, ft); err != nil {
		manager.logger.Error("publish feature type", "workspace", ws, "datastore", dsName, "layer", layerName, "err", err)
		return newGISError("PublishGeoserverLayer", ws+"/"+dsName+"/"+layerName, ErrGeoServerPublish, err)
	}
	return nil
}

// ensureWorkspace looks up the GeoServer workspace by name and creates it
// if missing. Errors are wrapped as *GISError with Op="ensureWorkspace"
// and Sentinel=ErrGeoServerPublish so that errors.Is(...,
// ErrGeoServerPublish) succeeds at every layer of the chain and the
// underlying *geoserver.APIError remains recoverable via errors.As.
// Callers (notably PublishGeoserverLayer) typically re-wrap the returned
// *GISError with their own Op for the public-facing error envelope; the
// inner GISError remains reachable via Unwrap for finer-grained triage.
//
// Concurrent-call safety: when N goroutines all hit the Get-not-found
// branch simultaneously, only the first Create wins; the rest receive
// 409 Conflict ([geoserver.ErrConflict]). That outcome is operationally
// equivalent to "workspace exists" — exactly what the ensure semantics
// promise — so we treat ErrConflict as success rather than propagating
// the race up to the caller. PublishAll's worker-pool publish path
// depends on this idempotency.
func ensureWorkspace(ctx context.Context, c *geoserver.Client, name string) error {
	if _, err := c.Workspaces.Get(ctx, name); err == nil {
		return nil
	} else if !errors.Is(err, geoserver.ErrNotFound) {
		return newGISError("ensureWorkspace", name, ErrGeoServerPublish, err)
	}
	if err := c.Workspaces.Create(ctx, &workspaces.Workspace{Name: name}); err != nil {
		if errors.Is(err, geoserver.ErrConflict) {
			return nil
		}
		return newGISError("ensureWorkspace", name, ErrGeoServerPublish, err)
	}
	return nil
}

// ensureDatastore looks up the PostGIS-backed datastore in the given
// workspace and creates it if missing. Same error-wrapping contract and
// same concurrent-create idempotency (ErrConflict treated as success)
// as [ensureWorkspace] — errors are *GISError with Op="ensureDatastore"
// and Sentinel=ErrGeoServerPublish.
func ensureDatastore(ctx context.Context, c *geoserver.Client, ws string, ds DatastoreConfig) error {
	scoped := c.Datastores.InWorkspace(ws)
	if _, err := scoped.Get(ctx, ds.Name); err == nil {
		return nil
	} else if !errors.Is(err, geoserver.ErrNotFound) {
		return newGISError("ensureDatastore", ds.Name, ErrGeoServerPublish, err)
	}
	conn := datastores.PostGIS{
		Name:     ds.Name,
		Host:     ds.Host,
		Port:     int(ds.Port),
		Database: ds.DBName,
		User:     ds.DBUser,
		Password: ds.DBPass,
	}
	if err := scoped.Create(ctx, conn); err != nil {
		if errors.Is(err, geoserver.ErrConflict) {
			return nil
		}
		return newGISError("ensureDatastore", ds.Name, ErrGeoServerPublish, err)
	}
	return nil
}

// LayerToPostgis copies this layer into the given GDAL PostgreSQL data source
// (typically a PostGIS-enabled database opened via the OGR PG: driver),
// preserving the geometry column name and optionally overwriting an existing
// table of the same name.
//
// Returns errors wrapping [ErrPostGISConnect] (PostGIS unreachable),
// [ErrInvalidDatasource] (nil targetSource), or [ErrInvalidLayer] (nil
// embedded *gdal.Layer). Match via [errors.Is].
func (layer *GdalLayer) LayerToPostgis(targetSource *gdal.DataSource, manager *ManagerConfig, overwrite bool) (newLayer *GdalLayer, err error) {
	connStr := manager.Datastore.PostgresConnectionString()
	if dbErr := DBIsAlive("postgres", connStr); dbErr != nil {
		err = newGISError("LayerToPostgis", manager.Datastore.Name, ErrPostGISConnect, dbErr)
		return
	}
	if targetSource == nil {
		err = newGISError("LayerToPostgis", "", ErrInvalidDatasource, nil)
		return
	}
	if layer.Layer == nil {
		err = newGISError("LayerToPostgis", "", ErrInvalidLayer, nil)
		return
	}
	datasource := *targetSource
	var options []string
	geomName := layer.GeometryColumn()
	if geomName != "" {
		options = append(options, fmt.Sprintf("GEOMETRY_NAME=%s", layer.GeometryColumn()))
	}
	if overwrite {
		options = append(options, "OVERWRITE=YES")
	}
	innerLayer := datasource.CopyLayer(*layer.Layer, layer.Name(), options)
	newLayer = &GdalLayer{
		Layer: &innerLayer,
	}
	return
}

// GeometryName returns the OGR geometry-type name for this layer
// ("POINT", "LINESTRING", etc.), defaulting to "geom" if the layer has
// no geometry column.
func (layer *GdalLayer) GeometryName() string {
	geom := gdal.Create(layer.Type())
	name := geom.Name()
	if name == "" {
		return "geom"
	}
	return name
}

// GetGeomtryName is the historical (typo'd) name of [GeometryName].
//
// Deprecated: typo (missing 'e' in "Geometry"). Use [GeometryName],
// which is byte-for-byte identical in behaviour. The typo'd form is
// kept for v1.x back-compat and will be removed at v2.
func (layer *GdalLayer) GetGeomtryName() string { return layer.GeometryName() }

// GetLayerSchema returns the layer's geometry column followed by every
// attribute field, each as a LayerField with name + OGR type.
func (layer *GdalLayer) GetLayerSchema() (fields []*LayerField) {
	if layer.Layer != nil {
		layerDef := layer.Definition()
		geomName := layer.GeometryColumn()
		geomField := LayerField{
			Name: geomName,
			Type: layer.GeometryName(),
		}
		fields = append(fields, &geomField)
		for index := 0; index < layerDef.FieldCount(); index++ {
			fieldDef := layerDef.FieldDefinition(index)
			layerField := LayerField{
				Name: fieldDef.Name(),
				Type: fieldDef.Type().Name(),
			}
			fields = append(fields, &layerField)

		}
	}
	return
}

// GetFeatures returns every feature in the layer, materialized into a slice.
// Features are read in OGR-FID order. Returns nil if FeatureCount fails.
//
// Deprecated: leaks gdal.Feature handles — each feature in the returned
// slice owns a C-level handle that must be released via [gdal.Feature.Destroy],
// which the slice form makes easy to forget. Use [(*GdalLayer).Features]
// which destroys each feature as iteration advances and on early break.
func (layer *GdalLayer) GetFeatures() (features []*gdal.Feature) {
	logger := layer.loggerOrDefault()
	if layer.Layer != nil {
		count, ok := layer.FeatureCount(true)
		if !ok {
			logger.Error("could not read features")
		} else {
			logger.Info("read features", "count", count)
			for index := 0; index < count; index++ {
				f := layer.Feature(int64(index))
				features = append(features, &f)
			}
		}
	}
	return
}

// Features returns an iterator over every feature in the layer. Features
// are emitted in OGR-FID order, and each feature is destroyed via
// [gdal.Feature.Destroy] as the iteration advances — so the caller never
// has to remember to free a handle. The cleanup also runs on early break:
// the still-yielded feature is destroyed when the for-range loop exits,
// regardless of why.
//
// ctx is honored between feature reads — if it is canceled mid-iteration,
// the iterator stops cleanly (and the deferred destroy still runs).
//
// Returns an empty iteration if the embedded *gdal.Layer is nil or
// FeatureCount fails (the manager's logger gets an Error record in the
// FeatureCount-failure case).
func (layer *GdalLayer) Features(ctx context.Context) iter.Seq[gdal.Feature] {
	logger := layer.loggerOrDefault()
	return func(yield func(gdal.Feature) bool) {
		if layer == nil || layer.Layer == nil {
			return
		}
		count, ok := layer.FeatureCount(true)
		if !ok {
			logger.Error("could not read features")
			return
		}
		logger.Info("read features", "count", count)
		for i := 0; i < count; i++ {
			if err := ctx.Err(); err != nil {
				return
			}
			f := layer.Feature(int64(i))
			keepGoing := yield(f)
			f.Destroy()
			if !keepGoing {
				return
			}
		}
	}
}
