package storage

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"sync"
)

// Storage interface
type StorageInterface interface {
	Add(ctx context.Context, url string) (string, error)
	Get(ctx context.Context, id string) (string, error)
}

// storage for all ids and corresponding urls
type Storage struct {
	mu   sync.RWMutex
	data map[string]string
	idSize int
}

// create new storage with desired size of ids
func NewStorage(idSize int) *Storage {
	return &Storage{
		data: make(map[string]string),
		idSize: idSize,
	}
}

// add url into storage and return generated id
func (store *Storage) Add(ctx context.Context, value string) (string, error) {
	select {
    case <-ctx.Done():
        return "", ctx.Err() // отмена или дедлайн
    default:
    }
	store.mu.Lock() // lock storage for writing
	defer store.mu.Unlock() // unlock storage for writing after function completion
	id := generateId(store.idSize)
	store.data[id] = value
	return id, nil
}

// get url by id from storage if exists
func (store *Storage) Get(ctx context.Context, id string) (string, error) {
	select {
    case <-ctx.Done():
        return "", ctx.Err() // отмена или дедлайн
    default:
    }
	store.mu.RLock() // lock storage for reading
	defer store.mu.RUnlock() // unlock storage for reading after function completion
	value, ok := store.data[id]
	if !ok {
		return "", ErrNotFound
	}
	return value, nil
}

// generate ID  for url with desired size
func generateId(size int) string {
	b := make([]byte, size) // empty bytes array
	rand.Read(b)  // fill bytes array with random bytes
	return base64.RawURLEncoding.EncodeToString(b)[:size] // encode bytes array to string
}