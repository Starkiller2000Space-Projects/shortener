package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"github.com/max-marek-projects/shortener/internal/config/db"
	"github.com/max-marek-projects/shortener/internal/utils"
)

type dbStorage struct {
	mu      sync.RWMutex
	storage *sql.DB
	idSize  int
}

func NewDBStorage(dbURL string, idSize int) (*dbStorage, error) {
	storage, err := db.Connect(db.NewDbConf(dbURL))
	if err != nil {
		return nil, err
	}
	dbs := &dbStorage{
		storage: storage,
		idSize:  idSize,
	}
	err = dbs.create()
	if err != nil {
		return nil, err
	}
	return dbs, nil
}

func (dbs *dbStorage) create() error {
	createTableSQL := fmt.Sprintf(`
    CREATE TABLE IF NOT EXISTS urls (
        id VARCHAR(%d) PRIMARY KEY,
        original_url TEXT NOT NULL
    );`, dbs.idSize)
	if _, err := dbs.storage.Exec(createTableSQL); err != nil {
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
	dbs.mu.Lock()         // lock storage for writing
	defer dbs.mu.Unlock() // unlock storage for writing after function completion
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
	dbs.mu.RLock()         // lock storage for reading
	defer dbs.mu.RUnlock() // unlock storage for reading after function completion
	var val string
	err := dbs.storage.QueryRowContext(ctx, "SELECT FROM urls urls(original_url) WHERE id = $1", id).Scan(&val)
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
