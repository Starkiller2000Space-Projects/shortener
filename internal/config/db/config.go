// Package db provides database configuration and connection utilities.

package db

import "time"

// DBConf holds database connection configuration.
type DBConf struct {
	URL             string        // database connection url
	MaxOpenConns    int           // max amount of opened database connections
	MaxIdleConns    int           // max amount of idle database connections
	ConnMaxLifetime time.Duration // max database connection lifetime
	MigrationsPath  string        // path to folder with migrations files
}

// NewDBConf creates a new DBConf with the provided DSN and default settings.
// Defaults: MaxOpenConns=10, MaxIdleConns=5, ConnMaxLifetime=5m, MigrationsPath="./migrations".
func NewDBConf(dbURL, migrationsPath string) *DBConf {
	return &DBConf{
		URL:             dbURL,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		MigrationsPath:  migrationsPath,
	}
}
