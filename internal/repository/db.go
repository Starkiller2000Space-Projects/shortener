// Package repository defines storage interfaces and implementations (memory, file, DB).

package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/max-marek-projects/shortener/internal/config/db"
	"github.com/max-marek-projects/shortener/internal/logger"
	"go.uber.org/zap"

	_ "github.com/golang-migrate/migrate/v4/database/postgres" // required for migrations
	_ "github.com/golang-migrate/migrate/v4/source/file"       // required for migrations
)

// dbStorage is an implementation of Storage backed by a PostgreSQL database.
type dbStorage struct {
	storage *sql.DB
	config  *db.DBConf
}

func NewDBStorage(config *db.DBConf) (*dbStorage, error) {
	storage, err := db.Connect(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create DB storage: %w", err)
	}
	dbs := &dbStorage{
		storage: storage,
		config:  config,
	}
	err = dbs.runMigrations()
	if err != nil {
		return nil, fmt.Errorf("failed to create DB storage: %w", err)
	}
	return dbs, nil
}

// runMigrations applies all pending migrations using golang-migrate.
// Returns an error if migration fails (or if no change and not ErrNoChange).
func (dbs *dbStorage) runMigrations() error {
	logger.Log.Info("Running migrations", zap.String("path", dbs.config.MigrationsPath))
	m, err := migrate.New(
		"file://"+dbs.config.MigrationsPath,
		dbs.config.URL,
	)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}

// add url into storage and return generated id
func (dbs *dbStorage) Add(ctx context.Context, info Row) error {
	var existingID string
	query := `--sql
		INSERT INTO urls (id, original_url, user_id, is_deleted)
		VALUES ($1, $2, $3, false)
		ON CONFLICT (original_url) DO UPDATE
		SET original_url = EXCLUDED.original_url
		RETURNING id
	`
	err := dbs.storage.QueryRowContext(ctx, query, info.ID, info.OriginalURL, info.UserID).Scan(&existingID)
	if err != nil {
		return fmt.Errorf("failed to add url to storage: %w", err)
	}
	if existingID != info.ID {
		return &ErrAlreadyExists{ExistingID: existingID}
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
	var isDeleted bool
	err := dbs.storage.QueryRowContext(ctx, `SELECT original_url, is_deleted FROM urls WHERE id = $1`, id).Scan(&val, &isDeleted)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("failed to get url from storage by id: %w", err)
	}
	if isDeleted {
		return "", fmt.Errorf("%w: %s", ErrGone, id)
	}
	return val, nil
}

// ping storage and check if it is available
func (dbs *dbStorage) Ping(ctx context.Context) error {
	return dbs.storage.PingContext(ctx)
}

// add data as batch
func (dbs *dbStorage) AddBatch(ctx context.Context, items []Row) error {
	if len(items) == 0 {
		return nil
	}
	query, args := dbs.buildBatchInsertQuery(items)
	if _, err := dbs.storage.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to add data as batch: %w", err)
	}
	return nil
}

// buildBatchInsertQuery constructs a SQL INSERT query for multiple rows.
// Returns the query string and the argument slice.
func (dbs *dbStorage) buildBatchInsertQuery(items []Row) (string, []any) {
	var b strings.Builder
	b.WriteString(`INSERT INTO urls(id, original_url, user_id, is_deleted) VALUES `)

	args := make([]any, 0, len(items)*3)
	for i, item := range items {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(fmt.Sprintf("($%d, $%d, $%d, false)", i*3+1, i*3+2, i*3+3))
		args = append(args, item.ID, item.OriginalURL, item.UserID)
	}

	return b.String(), args
}

func (dbs *dbStorage) GetUserURLs(ctx context.Context, userID string) ([]UserURL, error) {
	query := `SELECT id, original_url FROM urls WHERE user_id = $1 AND is_deleted = false`
	rows, err := dbs.storage.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []UserURL
	for rows.Next() {
		var id, original string
		if err := rows.Scan(&id, &original); err != nil {
			return nil, err
		}
		res = append(res, UserURL{ShortURL: id, OriginalURL: original})
	}
	return res, nil
}

func (dbs *dbStorage) DeleteBatch(ctx context.Context, userID string, shortIDs []string) error {
	if len(shortIDs) == 0 {
		return nil
	}
	query := `UPDATE urls SET is_deleted = true WHERE user_id = $1 AND id = ANY($2)`
	_, err := dbs.storage.ExecContext(ctx, query, userID, shortIDs)
	if err != nil {
		return fmt.Errorf("failed to delete batch: %w", err)
	}
	return nil
}
