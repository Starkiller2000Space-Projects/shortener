package repository

import (
	"context"
	"sync"
)

type memStorage struct {
	mu   sync.RWMutex
	data map[string]string
}

func NewMemStorage() (*memStorage, error) {
	return &memStorage{
		data: make(map[string]string),
	}, nil // error not possible, but output is consistent with other storage types
}

// add url into storage and return generated id
func (fs *memStorage) Add(ctx context.Context, id, url string) error {
	select {
	case <-ctx.Done(): // cancel or deadline
		return ctx.Err()
	default:
	}
	fs.mu.Lock()         // lock storage for writing
	defer fs.mu.Unlock() // unlock storage for writing after function completion
	fs.data[id] = url
	return nil
}

// get url by id from storage if exists
func (fs *memStorage) Get(ctx context.Context, id string) (string, error) {
	select {
	case <-ctx.Done(): // cancel or deadline
		return "", ctx.Err()
	default:
	}
	fs.mu.RLock()         // lock storage for reading
	defer fs.mu.RUnlock() // unlock storage for reading after function completion
	val, ok := fs.data[id]
	if !ok {
		return "", ErrNotFound
	}
	return val, nil
}

// ping storage and check if it is available
func (fs *memStorage) Ping(ctx context.Context) error {
	return nil // always available
}

func (m *memStorage) AddBatch(ctx context.Context, items []Row) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, item := range items {
		m.data[item.ID] = item.OriginalURL
	}
	return nil
}
