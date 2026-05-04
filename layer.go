package gismanager

import (
	"context"
	"errors"
	"fmt"

	geoserver "github.com/hishamkaram/geoserver/v2"
	"github.com/hishamkaram/geoserver/v2/rest/datastores"
	"github.com/hishamkaram/geoserver/v2/rest/featuretypes"
	"github.com/hishamkaram/geoserver/v2/rest/workspaces"
	"github.com/lukeroth/gdal"
)

// GdalLayer wraps a *gdal.Layer with the helper methods gismanager uses
// (publish, copy-to-PostGIS, schema introspection).
type GdalLayer struct {
	*gdal.Layer
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

func ensureWorkspace(ctx context.Context, c *geoserver.Client, name string) error {
	if _, err := c.Workspaces.Get(ctx, name); err == nil {
		return nil
	} else if !errors.Is(err, geoserver.ErrNotFound) {
		return fmt.Errorf("get workspace %q: %w", name, err)
	}
	if err := c.Workspaces.Create(ctx, &workspaces.Workspace{Name: name}); err != nil {
		return fmt.Errorf("create workspace %q: %w", name, err)
	}
	return nil
}

func ensureDatastore(ctx context.Context, c *geoserver.Client, ws string, ds DatastoreConfig) error {
	scoped := c.Datastores.InWorkspace(ws)
	if _, err := scoped.Get(ctx, ds.Name); err == nil {
		return nil
	} else if !errors.Is(err, geoserver.ErrNotFound) {
		return fmt.Errorf("get datastore %q: %w", ds.Name, err)
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
		return fmt.Errorf("create datastore %q: %w", ds.Name, err)
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

// GetGeomtryName returns the OGR geometry-type name for this layer ("POINT",
// "LINESTRING", etc.), defaulting to "geom" if the layer has no geometry
// column.
func (layer *GdalLayer) GetGeomtryName() (geometryName string) {
	geom := gdal.Create(layer.Type())
	geometryName = geom.Name()
	if geometryName == "" {
		geometryName = "geom"
	}
	return
}

// GetLayerSchema returns the layer's geometry column followed by every
// attribute field, each as a LayerField with name + OGR type.
func (layer *GdalLayer) GetLayerSchema() (fields []*LayerField) {
	if layer.Layer != nil {
		layerDef := layer.Definition()
		geomName := layer.GeometryColumn()
		geomField := LayerField{
			Name: geomName,
			Type: layer.GetGeomtryName(),
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
func (layer *GdalLayer) GetFeatures() (features []*gdal.Feature) {
	logger := GetLogger()
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
