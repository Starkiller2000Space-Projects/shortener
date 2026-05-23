package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type fileRecord struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id"`
}

type fileStorage struct {
	*memStorage
	filePath string
}

func NewFileStorage(filePath string) (*fileStorage, error) {
	mem, err := NewMemStorage()
	if err != nil {
		return nil, err
	}
	fs := &fileStorage{
		memStorage: mem,
		filePath:   filePath,
	}
	if err := fs.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("Failed to create file storage: %w", err)
	}
	return fs, nil
}

func (fs *fileStorage) load() error {
	f, err := os.Open(fs.filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("failed to load storage file: %w", err)
	}
	defer f.Close()

	var records []fileRecord
	if err := json.NewDecoder(f).Decode(&records); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return fmt.Errorf("failed to load storage file: %w", err)
	}

	fs.mu.Lock()
	defer fs.mu.Unlock()
	for _, rec := range records {
		fs.data[rec.ShortURL] = urlInfo{originalURL: rec.OriginalURL, userID: rec.UserID}
	}
	return nil
}

func (fs *fileStorage) save() error {
	records := make([]fileRecord, 0, len(fs.data))
	for short, info := range fs.data {
		records = append(records, fileRecord{ShortURL: short, OriginalURL: info.originalURL})
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
func (fs *fileStorage) add(info Row) error {
	err := fs.memStorage.add(info)
	if err != nil {
		return err
	}
	if err := fs.save(); err != nil {
		delete(fs.data, info.ID) // undo on error
		return fmt.Errorf("Failed to add data to storage file: %w", err)
	}
	return nil
}

// add url into storage and return generated id
func (fs *fileStorage) Add(ctx context.Context, info Row) error {
	select {
	case <-ctx.Done(): // cancel or deadline
		return ctx.Err()
	default:
	}
	fs.mu.Lock()         // lock storage for writing
	defer fs.mu.Unlock() // unlock storage for writing after function completion
	return fs.add(info)
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

// add multiple values as batch
func (fs *fileStorage) addBatch(items []Row) error {
	file, err := os.OpenFile(fs.filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("Failed to add batch to file: %w", err)
	}
	defer file.Close()
	err = fs.memStorage.addBatch(items)
	if err != nil {
		return err
	}
	if err := fs.save(); err != nil {
		for _, item := range items {
			delete(fs.data, item.ID) // undo on error
		}
		return fmt.Errorf("Failed to add batch to file: %w", err)
	}
	return nil
}

// add multiple values as batch with mutex
func (fs *fileStorage) AddBatch(ctx context.Context, items []Row) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	fs.mu.Lock()         // lock storage for writing
	defer fs.mu.Unlock() // unlock storage for writing after function completion
	return fs.addBatch(items)
}
