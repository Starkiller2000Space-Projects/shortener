package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type fileRecord struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type fileStorage struct {
	mu       sync.RWMutex
	data     map[string]string
	filePath string
}

func NewFileStorage(filePath string) (*fileStorage, error) {
	fs := &fileStorage{
		data:     make(map[string]string),
		filePath: filePath,
	}
	if err := fs.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("Failed to create file storage: %w", err)
	}
	return fs, nil
}

func (fs *fileStorage) load() error {
	f, err := os.Open(fs.filePath)
	if err != nil {
		return fmt.Errorf("Failed to load storage file: %w", err)
	}
	defer f.Close()
	var records []fileRecord
	if err := json.NewDecoder(f).Decode(&records); err != nil {
		return fmt.Errorf("Failed to load storage file: %w", err)
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	for _, rec := range records {
		fs.data[rec.ShortURL] = rec.OriginalURL
	}
	return nil
}

func (fs *fileStorage) save() error {
	records := make([]fileRecord, 0, len(fs.data))
	for short, original := range fs.data {
		records = append(records, fileRecord{ShortURL: short, OriginalURL: original})
	}
	f, err := os.Create(fs.filePath)
	if err != nil {
		return fmt.Errorf("Failed to save storage file: %w", err)
	}
	defer f.Close()
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "    ")
	return encoder.Encode(records)
}

// add url into storage and return generated id
func (fs *fileStorage) Add(ctx context.Context, id, url string) error {
	select {
	case <-ctx.Done(): // cancel or deadline
		return ctx.Err()
	default:
	}
	fs.mu.Lock()         // lock storage for writing
	defer fs.mu.Unlock() // unlock storage for writing after function completion
	fs.data[id] = url
	if err := fs.save(); err != nil {
		delete(fs.data, id) // undo on error
		return fmt.Errorf("Failed to add data to storage file: %w", err)
	}
	return nil
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

// ping storage and check if it is available
func (fs *fileStorage) Ping(ctx context.Context) error {
	if _, err := os.Stat(fs.filePath); err == nil {
		f, err := os.Open(fs.filePath)
		if err != nil {
			return fmt.Errorf("Failed to check storage file: %w", err)
		}
		f.Close()
		return nil
	} else if os.IsNotExist(err) {
		dir := filepath.Dir(fs.filePath)
		tmp, err := os.CreateTemp(dir, "ping_test_*") // create temporary file in order to check permissions
		if err != nil {
			return fmt.Errorf("Failed to check storage file: %w", err)
		}
		tmp.Close()
		os.Remove(tmp.Name())
		return nil
	} else {
		return fmt.Errorf("Failed to check storage file: %w", err)
	}
}

func (fs *fileStorage) AddBatch(ctx context.Context, items []Row) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()

	file, err := os.OpenFile(fs.filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("Failed to add batch to file: %w", err)
	}
	defer file.Close()

	for _, item := range items {
		fs.data[item.ID] = item.OriginalURL
	}
	if err := fs.save(); err != nil {
		for _, item := range items {
			delete(fs.data, item.ID) // undo on error
		}
		return fmt.Errorf("Failed to add batch to file: %w", err)
	}
	return nil
}
