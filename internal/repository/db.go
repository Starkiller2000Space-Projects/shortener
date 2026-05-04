package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/golang-migrate/migrate/v4"
	"github.com/max-marek-projects/shortener/internal/config/db"
	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/utils"
	"go.uber.org/zap"

	_ "github.com/golang-migrate/migrate/v4/database/postgres" // required for migrations
	_ "github.com/golang-migrate/migrate/v4/source/file"       // required for migrations
)

type dbStorage struct {
	storage *sql.DB
	config  *db.DBConf
	idSize  int
}

func NewDBStorage(dbURL string, idSize int) (*dbStorage, error) {
	config := db.NewDbConf(dbURL)
	storage, err := db.Connect(config)
	if err != nil {
		return nil, err
	}
	dbs := &dbStorage{
		storage: storage,
		config:  config,
		idSize:  idSize,
	}
	err = dbs.runMigrations()
	if err != nil {
		return nil, err
	}
	return dbs, nil
}

// run database migrations
func (dbs *dbStorage) runMigrations() error {
	logger.Log.Info("Running migrations", zap.String("path", dbs.config.MigrationsPath))
	m, err := migrate.New(
		"file://"+dbs.config.MigrationsPath,
		dbs.config.URL,
	)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// add url into storage and return generated id
func (dbs *dbStorage) Add(ctx context.Context, url string) (string, error) {
	select {
	case <-ctx.Done(): // cancel or deadline
		return "", ctx.Err()
	default:
	}
	id := utils.GenerateId(dbs.idSize)
	_, err := dbs.storage.ExecContext(ctx, "INSERT INTO urls(id, original_url) VALUES($1, $2)", id, url)
	if err != nil {
		return "", err
	}
	return id, nil
}

// get url by id from storage if exists
func (dbs *dbStorage) Get(ctx context.Context, id string) (string, error) {
	select {
	case <-ctx.Done(): // cancel or deadline
		return "", ctx.Err()
	default:
	}
	var val string
	err := dbs.storage.QueryRowContext(ctx, "SELECT original_url FROM urls WHERE id = $1", id).Scan(&val)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return val, nil
}

// ping storage and check if it is available
func (dbs *dbStorage) Ping(ctx context.Context) error {
	return dbs.storage.PingContext(ctx)
}
