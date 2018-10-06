package gismanager

import "fmt"

type datastore struct {
	Host   string `yaml:"host"`
	Port   uint   `yaml:"port"`
	DBName string `yaml:"database"`
	DBUser string `yaml:"username"`
	DBPass string `yaml:"password"`
	Name   string `yaml:"name"`
}

//BuildConnectionString return postgres connection as string
func (ds *datastore) BuildConnectionString() string {
	return fmt.Sprintf("PG: host=%s port=%d dbname=%s user=%s password=%s", ds.Host, ds.Port, ds.DBName, ds.DBUser, ds.DBPass)
}
