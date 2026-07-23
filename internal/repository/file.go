// Package repository defines storage interfaces and implementations (memory, file, DB).
package repository

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// fileRecord is the JSON structure used for storing a single URL in the file.
type fileRecord struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id"`
	Deleted     bool   `json:"deleted"`
}

// fileStorage is an implementation of Storage that persists data to a file.
// It embeds memStorage and synchronizes writes to disk.
type fileStorage struct {
	*memStorage
	filePath string
}

// NewFileStorage creates a new file-based storage instance.
// It loads existing data from the file (if present) into memory.
// Returns an error if the file cannot be read or parsed.
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
		return nil, fmt.Errorf("failed to create file storage: %w", err)
	}
	return fs, nil
}

// load reads the file and populates the in-memory maps.
// If the file does not exist, it does nothing (no error).
func (fs *fileStorage) load() error {
	f, err := os.Open(fs.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	fs.mu.Lock()
	defer fs.mu.Unlock()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	fs.data = make(map[string]urlInfo)
	fs.urlToID = make(map[string]string)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var rec fileRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			return fmt.Errorf("failed to parse line: %w", err)
		}
		fs.data[rec.ShortURL] = urlInfo{
			originalURL: rec.OriginalURL,
			userID:      rec.UserID,
			deleted:     rec.Deleted,
		}
		fs.urlToID[rec.OriginalURL] = rec.ShortURL
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}
	return nil
}

// save rewrites the entire file with the current in-memory data.
// Used for full rebuilds (e.g., after deletion).
func (fs *fileStorage) save() error {
	records := make([]fileRecord, 0, len(fs.data))
	for short, info := range fs.data {
		records = append(records, fileRecord{ShortURL: short, OriginalURL: info.originalURL})
	}
	f, err := os.Create(fs.filePath)
	if err != nil {
		return fmt.Errorf("failed to save storage file: %w", err)
	}
	defer f.Close()
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "    ")
	return encoder.Encode(records)
}

// appendRecords appends one or more file records to the end of the file.
// Used for batch inserts.
func (fs *fileStorage) appendRecords(recs []fileRecord) error {
	if len(recs) == 0 {
		return nil
	}
	f, err := os.OpenFile(fs.filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	var buf bytes.Buffer
	for _, rec := range recs {
		data, err := json.Marshal(rec)
		if err != nil {
			return err
		}
		buf.Write(data)
		buf.WriteByte('\n')
	}
	_, err = f.Write(buf.Bytes())
	return err
}

// add is the internal version of Add without locking.
// It assumes the caller holds the mutex.
func (fs *fileStorage) add(info Row) error {
	err := fs.memStorage.add(info)
	if err != nil {
		return err
	}
	recs := []fileRecord{
		{
			ShortURL:    info.ID,
			OriginalURL: info.OriginalURL,
			UserID:      info.UserID,
			Deleted:     false,
		},
	}
	if err := fs.appendRecords(recs); err != nil {
		delete(fs.data, info.ID) // undo on error
		delete(fs.urlToID, info.OriginalURL)
		return fmt.Errorf("failed to add data to storage file: %w", err)
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
			return fmt.Errorf("failed to check storage file: %w", err)
		}
		return f.Close()
	} else if os.IsNotExist(err) {
		dir := filepath.Dir(fs.filePath)
		tmp, err := os.CreateTemp(dir, "ping_test_*") // create temporary file in order to check permissions
		if err != nil {
			return fmt.Errorf("failed to check storage file: %w", err)
		}
		err = tmp.Close()
		if err != nil {
			return err
		}
		return os.Remove(tmp.Name())
	} else {
		return fmt.Errorf("failed to check storage file: %w", err)
	}
}

// addBatch is the internal version of AddBatch without locking.
// It assumes the caller holds the mutex.
func (fs *fileStorage) addBatch(items []Row) error {
	file, err := os.OpenFile(fs.filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		return fmt.Errorf("failed to add batch to file: %w", err)
	}
	defer file.Close()
	err = fs.memStorage.addBatch(items)
	if err != nil {
		return err
	}
	recs := make([]fileRecord, len(items))
	for i, item := range items {
		recs[i] = fileRecord{
			ShortURL:    item.ID,
			OriginalURL: item.OriginalURL,
			UserID:      item.UserID,
			Deleted:     false,
		}
	}
	if err := fs.appendRecords(recs); err != nil {
		for _, item := range items {
			delete(fs.data, item.ID) // undo on error
			delete(fs.urlToID, item.OriginalURL)
		}
		return fmt.Errorf("failed to add batch to file: %w", err)
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

// deleteBatch is the internal version of DeleteBatch without locking.
// It assumes the caller holds the mutex.
func (fs *fileStorage) deleteBatch(userID string, shortIDs []string) error {
	for _, id := range shortIDs {
		if info, ok := fs.data[id]; ok && info.userID == userID {
			info.deleted = true
			fs.data[id] = info
		}
	}
	if err := fs.save(); err != nil {
		return err
	}
	return nil
}

// delete batch by user id and short ids with mutex
func (fs *fileStorage) DeleteBatch(ctx context.Context, userID string, shortIDs []string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if len(shortIDs) == 0 {
		return nil
	}
	fs.mu.Lock()
	defer fs.mu.Unlock()
	return fs.deleteBatch(userID, shortIDs)
}
