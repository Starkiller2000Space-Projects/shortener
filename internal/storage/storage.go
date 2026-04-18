package storage

import (
	"context"
	"encoding/json"
	"os"
	"sync"

	"github.com/Starkiller2000Space-Projects/shortener/internal/utils"
)

// Storage interface
type StorageInterface interface {
	Add(ctx context.Context, url string) (string, error)
	Get(ctx context.Context, id string) (string, error)
}

type fileRecord struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type fileStorage struct {
	mu       sync.RWMutex
	data     map[string]string
	filePath string
	idSize   int
}

func NewStorage(filePath string, idSize int) (*fileStorage, error) {
	fs := &fileStorage{
		data:     make(map[string]string),
		filePath: filePath,
		idSize:   idSize,
	}
	if err := fs.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return fs, nil
}

func (fs *fileStorage) load() error {
	f, err := os.Open(fs.filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	var records []fileRecord
	if err := json.NewDecoder(f).Decode(&records); err != nil {
		return err
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	for _, rec := range records {
		fs.data[rec.ShortURL] = rec.OriginalURL
	}
	return nil
}

func (fs *fileStorage) save() error {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	records := make([]fileRecord, 0, len(fs.data))
	for short, original := range fs.data {
		records = append(records, fileRecord{ShortURL: short, OriginalURL: original})
	}
	f, err := os.Create(fs.filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(records)
}

// add url into storage and return generated id
func (fs *fileStorage) Add(ctx context.Context, url string) (string, error) {
	select {
	case <-ctx.Done(): // cancel or deadline
		return "", ctx.Err()
	default:
	}
	fs.mu.Lock()         // lock storage for writing
	defer fs.mu.Unlock() // unlock storage for writing after function completion
	id := utils.GenerateId(fs.idSize)
	fs.data[id] = url
	if err := fs.save(); err != nil {
		delete(fs.data, id) // undo on error
		return "", err
	}
	return id, nil
}

// get url by id from storage if exists
func (fs *fileStorage) Get(ctx context.Context, id string) (string, error) {
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
