package repository

import (
	"context"
	"fmt"

	"github.com/max-marek-projects/shortener/internal/config"
	"github.com/max-marek-projects/shortener/internal/logger"
)

// Common storage interface
//
//go:generate mockery --name=Storage --output=../service/mocks --with-expecter
type Storage interface {
	Add(ctx context.Context, info Row) error
	Get(ctx context.Context, id string) (string, error)
	Ping(ctx context.Context) error
	AddBatch(ctx context.Context, items []Row) error
	GetUserURLs(ctx context.Context, userID string) ([]UserURL, error)
	DeleteBatch(ctx context.Context, userID string, shortIDs []string) error
}

func GetStorage(cfg *config.Config) (Storage, error) {
	if (cfg.DatabaseUrl != "") && (cfg.FileStoragePath != "") {
		return nil, fmt.Errorf("%w: cannot use both -d (database storage) and -f (file storage)", ErrMutuallyExclusiveFlags)
	}
	var storage Storage
	var err error
	switch {
	case cfg.DatabaseUrl != "":
		logger.Log.Info("Initializing database storage")
		storage, err = NewDBStorage(cfg.DatabaseUrl)
	case cfg.FileStoragePath != "":
		logger.Log.Info("Initializing file storage")
		storage, err = NewFileStorage(cfg.FileStoragePath)
	default:
		logger.Log.Info("Initializing simple memory storage")
		storage, err = NewMemStorage()
	}
	if err != nil {
		return nil, fmt.Errorf("Failed to create storage: %w", err)
	}
	return storage, nil
}

type Row struct {
	ID          string
	OriginalURL string
	UserID      string
}

type UserURL struct {
	ShortURL    string
	OriginalURL string
}

type urlInfo struct {
	originalURL string
	userID      string
	deleted     bool
}
