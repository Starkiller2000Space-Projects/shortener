package db

import "time"

type DBConf struct {
	URL             string        // database connection url
	MaxOpenConns    int           // max amount of opened database connections
	MaxIdleConns    int           // max amount of idle database connections
	ConnMaxLifetime time.Duration // max database connection lifetime
}

func NewDbConf(dbUrl string) DBConf {
	return DBConf{
		URL:             dbUrl,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}
}
