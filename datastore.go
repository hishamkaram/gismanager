package gismanager

import "fmt"

type DatastoreConfig struct {
	Host   string `yaml:"host"`
	Port   uint   `yaml:"port"`
	DBName string `yaml:"database"`
	DBUser string `yaml:"username"`
	DBPass string `yaml:"password"`
	Name   string `yaml:"name"`
}

//BuildConnectionString return gdal postgres connection as string
func (ds *DatastoreConfig) BuildConnectionString() string {
	return fmt.Sprintf("PG: host=%s port=%d dbname=%s user=%s password=%s", ds.Host, ds.Port, ds.DBName, ds.DBUser, ds.DBPass)
}

//PostgresConnectionString return postgres connection as string
func (ds *DatastoreConfig) PostgresConnectionString() string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%d/%s?sslmode=disable", ds.DBUser, ds.DBPass, ds.Host, ds.Port, ds.DBName)
}
