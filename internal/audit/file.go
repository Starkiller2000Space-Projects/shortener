// Package audit implements an audit logging system with multiple observers.
package audit

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/models"
	"go.uber.org/zap"
)

// fileAuditObserver writes audit events to a file in JSON format,
// appending each event as a new line.
type fileAuditObserver struct {
	file *os.File
	mu   sync.Mutex
}

// NewFileAuditObserver creates a new file audit observer with the given file path.
func NewFileAuditObserver(path string) (*fileAuditObserver, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600) // #nosec G304
	if err != nil {
		logger.Log.Error("failed to open audit file", zap.Error(err))
		return nil, err
	}
	return &fileAuditObserver{file: file}, nil
}

// Notify writes the audit event to the file.
// It acquires a lock to ensure thread-safety and logs errors if writing fails.
func (f *fileAuditObserver) Notify(event models.AuditEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		logger.Log.Error("failed to marshal audit event", zap.Error(err))
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	_, err = f.file.Write(append(data, '\n'))
	if err != nil {
		logger.Log.Error("failed to write audit event", zap.Error(err))
	}
}

// close file observer
func (f *fileAuditObserver) Stop() error {
	return f.file.Close()
}
