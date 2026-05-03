package repository

import (
	"context"
	"fmt"

	"github.com/max-marek-projects/shortener/internal/config"
	"github.com/max-marek-projects/shortener/internal/logger"
)

// Common storage interface
type StorageInterface interface {
	Add(ctx context.Context, url string) (string, error)
	Get(ctx context.Context, id string) (string, error)
	Ping(ctx context.Context) error
}

func DetermineStorage(cfg *config.Config) (StorageInterface, error) {
	if (cfg.DatabaseUrl != "") && (cfg.FileStoragePath != "") {
		return nil, fmt.Errorf("%w: cannot use both -d (database storage) and -f (file storage)", ErrMutuallyExclusiveFlags)
	}
	var storage StorageInterface
	var err error
	switch {
	case cfg.DatabaseUrl != "":
		logger.Log.Info("Initializing database storage")
		storage, err = NewDBStorage(cfg.DatabaseUrl, cfg.IdSize)
	case cfg.FileStoragePath != "":
		logger.Log.Info("Initializing file storage")
		storage, err = NewFileStorage(cfg.FileStoragePath, cfg.IdSize)
	default:
		logger.Log.Info("Initializing simple memory storage")
		storage, err = NewMemStorage(cfg.IdSize)
	}
	if err != nil {
		return nil, err
	}
	return storage, nil
}
