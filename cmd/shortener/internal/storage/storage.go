package storage

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
)

// Storage interface
type StorageInterface interface {
	Add(url string) string
	Get(id string) (string, bool)
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
func (store *Storage) Add(value string) string {
	store.mu.Lock() // lock storage for writing
	defer store.mu.Unlock() // unlock storage for writing after function completion
	id := generateId(store.idSize)
	store.data[id] = value
	return id
}

// get url by id from storage if exists
func (store *Storage) Get(id string) (string, bool) {
	store.mu.RLock() // lock storage for reading
	defer store.mu.RUnlock() // unlock storage for reading after function completion
	value, ok := store.data[id]
	return value, ok
}

// generate ID  for url with desired size
func generateId(size int) string {
	b := make([]byte, size) // empty bytes array
	_, _ = rand.Read(b)  // fill bytes array with random bytes
	return base64.RawURLEncoding.EncodeToString(b)[:size] // encode bytes array to string
}