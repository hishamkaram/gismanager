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

//GdalLayer Layer
type GdalLayer struct {
	*gdal.Layer
}

//LayerField Layer Field
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
func (manager *ManagerConfig) PublishGeoserverLayer(ctx context.Context, layer *GdalLayer) error {
	catalog, err := manager.GetGeoserverCatalog()
	if err != nil {
		manager.logger.Error(err)
		return fmt.Errorf("gismanager: build geoserver client: %w", err)
	}

	ws := manager.Geoserver.WorkspaceName
	if err := ensureWorkspace(ctx, catalog, ws); err != nil {
		manager.logger.Error(err)
		return err
	}

	dsName := manager.Datastore.Name
	if err := ensureDatastore(ctx, catalog, ws, manager.Datastore); err != nil {
		manager.logger.Error(err)
		return err
	}

	layerName := layer.Name()
	ft := &featuretypes.FeatureType{
		Name:       layerName,
		NativeName: layerName,
		Enabled:    true,
	}
	if err := catalog.FeatureTypes.InWorkspace(ws).InDatastore(dsName).Create(ctx, ft); err != nil {
		manager.logger.Error(err)
		return fmt.Errorf("gismanager: publish feature type %q: %w", layerName, err)
	}
	return nil
}

func ensureWorkspace(ctx context.Context, c *geoserver.Client, name string) error {
	if _, err := c.Workspaces.Get(ctx, name); err == nil {
		return nil
	} else if !errors.Is(err, geoserver.ErrNotFound) {
		return fmt.Errorf("gismanager: get workspace %q: %w", name, err)
	}
	if err := c.Workspaces.Create(ctx, &workspaces.Workspace{Name: name}); err != nil {
		return fmt.Errorf("gismanager: create workspace %q: %w", name, err)
	}
	return nil
}

func ensureDatastore(ctx context.Context, c *geoserver.Client, ws string, ds DatastoreConfig) error {
	scoped := c.Datastores.InWorkspace(ws)
	if _, err := scoped.Get(ctx, ds.Name); err == nil {
		return nil
	} else if !errors.Is(err, geoserver.ErrNotFound) {
		return fmt.Errorf("gismanager: get datastore %q: %w", ds.Name, err)
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
		return fmt.Errorf("gismanager: create datastore %q: %w", ds.Name, err)
	}
	return nil
}

//LayerToPostgis add Layer to Postgis
func (layer *GdalLayer) LayerToPostgis(targetSource *gdal.DataSource, manager *ManagerConfig, overwrite bool) (newLayer *GdalLayer, err error) {
	connStr := manager.Datastore.PostgresConnectionString()
	dbErr := DBIsAlive("postgres", connStr)
	if dbErr != nil {
		err = dbErr
		return
	}
	if targetSource == nil {
		err = errors.New("Invalid Datasource")
		return
	}
	if layer.Layer == nil {
		err = errors.New("Invalid Layer")
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
	_layer := datasource.CopyLayer(*layer.Layer, layer.Name(), options)
	newLayer = &GdalLayer{
		Layer: &_layer,
	}
	return
}

//GetGeomtryName Get Geometry Name point/line/....etc
func (layer *GdalLayer) GetGeomtryName() (geometryName string) {
	geom := gdal.Create(layer.Layer.Type())
	geometryName = geom.Name()
	if len(geometryName) == 0 {
		geometryName = "geom"
	}
	return
}

//GetLayerSchema return slice of layer fields
func (layer *GdalLayer) GetLayerSchema() (fields []*LayerField) {
	if layer.Layer != nil {
		layerDef := layer.Layer.Definition()
		geomName := layer.Layer.GeometryColumn()
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

//GetFeatures return layer features
func (layer *GdalLayer) GetFeatures() (features []*gdal.Feature) {
	logger := GetLogger()
	if layer.Layer != nil {
		count, ok := layer.Layer.FeatureCount(true)
		if !ok {
			logger.Error("Could not read features")
		} else {
			logger.Infof("We Found %d Feature", count)
			for index := 0; index < count; index++ {
				f := layer.Layer.Feature(int64(index))
				features = append(features, &f)
			}
		}
	}
	return
}
