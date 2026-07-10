// Package repository defines storage interfaces and implementations (memory, file, DB).

package repository

import (
	"context"
	"fmt"

	"github.com/max-marek-projects/shortener/internal/config"
	"github.com/max-marek-projects/shortener/internal/config/db"
	"github.com/max-marek-projects/shortener/internal/logger"
)

// Storage defines the interface for URL storage backends.
// Implementations must be safe for concurrent use.
//
//go:generate mockery --name=Storage --output=../service  --outpkg=service --filename=mock_storage_test.gen.go --with-expecter --structname=MockStorage
type Storage interface {
	Add(ctx context.Context, info Row) error
	Get(ctx context.Context, id string) (string, error)
	Ping(ctx context.Context) error
	AddBatch(ctx context.Context, items []Row) error
	GetUserURLs(ctx context.Context, userID string) ([]UserURL, error)
	DeleteBatch(ctx context.Context, userID string, shortIDs []string) error
}

// GetStorage creates a new Storage instance based on the provided configuration.
// It chooses between database storage (if DatabaseURL is set), file storage (if FileStoragePath is set),
// or in-memory storage as a fallback.
// Returns an error if both database and file storage are configured (mutually exclusive).
func GetStorage(cfg *config.Config) (Storage, error) {
	if (cfg.DatabaseURL != "") && (cfg.FileStoragePath != "") {
		return nil, fmt.Errorf("%w: cannot use both -d (database storage) and -f (file storage)", ErrMutuallyExclusiveFlags)
	}
	var storage Storage
	var err error
	switch {
	case cfg.DatabaseURL != "":
		logger.Log.Info("Initializing database storage")
		storage, err = NewDBStorage(db.NewDBConf(cfg.DatabaseURL))
	case cfg.FileStoragePath != "":
		logger.Log.Info("Initializing file storage")
		storage, err = NewFileStorage(cfg.FileStoragePath)
	default:
		logger.Log.Info("Initializing simple memory storage")
		storage, err = NewMemStorage()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create storage: %w", err)
	}
	return storage, nil
}

// Row represents a single URL record to be stored.
type Row struct {
	ID          string
	OriginalURL string
	UserID      string
}

// UserURL represents a user's URL mapping (short ID and original URL).
type UserURL struct {
	ShortURL    string
	OriginalURL string
}

// urlInfo holds the original URL, user ID, and deletion status for a short ID.
type urlInfo struct {
	originalURL string
	userID      string
	deleted     bool
}
