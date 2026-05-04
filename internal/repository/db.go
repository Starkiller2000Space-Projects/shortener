package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/max-marek-projects/shortener/internal/config/db"
	"github.com/max-marek-projects/shortener/internal/logger"
	"go.uber.org/zap"

	_ "github.com/golang-migrate/migrate/v4/database/postgres" // required for migrations
	_ "github.com/golang-migrate/migrate/v4/source/file"       // required for migrations
)

type dbStorage struct {
	storage *sql.DB
	config  *db.DBConf
}

func NewDBStorage(dbURL string) (*dbStorage, error) {
	config := db.NewDbConf(dbURL)
	storage, err := db.Connect(config)
	if err != nil {
		return nil, err
	}
	dbs := &dbStorage{
		storage: storage,
		config:  config,
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
func (dbs *dbStorage) Add(ctx context.Context, id, url string) error {
	select {
	case <-ctx.Done(): // cancel or deadline
		return ctx.Err()
	default:
	}
	_, err := dbs.storage.ExecContext(ctx, "INSERT INTO urls(id, original_url) VALUES($1, $2)", id, url)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			var existingID string
			queryErr := dbs.storage.QueryRowContext(ctx, "SELECT id FROM urls WHERE original_url = $1", url).Scan(&existingID)
			if queryErr != nil {
				return fmt.Errorf("failed to fetch existing url: %w", queryErr)
			}
			return &ErrAlreadyExists{ExistingID: existingID}
		}
		return err
	}
	return nil
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

// add data as batch
func (dbs *dbStorage) AddBatch(ctx context.Context, items []Row) error {
	tx, err := dbs.storage.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO urls(id, original_url) VALUES($1, $2)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, item := range items {
		if _, err := stmt.ExecContext(ctx, item.ID, item.OriginalURL); err != nil {
			return err
		}
	}

	return tx.Commit()
}
