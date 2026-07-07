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

// FileAuditObserver writes audit events to a file in JSON format,
// appending each event as a new line.
type FileAuditObserver struct {
	filePath string
	mu       sync.Mutex
}

// NewFileAuditObserver creates a new file audit observer with the given file path.
func NewFileAuditObserver(path string) *FileAuditObserver {
	return &FileAuditObserver{filePath: path}
}

// Notify writes the audit event to the file.
// It acquires a lock to ensure thread-safety and logs errors if writing fails.
func (f *FileAuditObserver) Notify(event models.AuditEvent) {
	f.mu.Lock()
	defer f.mu.Unlock()

	file, err := os.OpenFile(f.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Log.Error("Failed to open audit file: %v", zap.Error(err))
		return
	}
	defer file.Close()

	data, err := json.Marshal(event)
	if err != nil {
		logger.Log.Error("Failed to marshal audit event: %v", zap.Error(err))
		return
	}
	_, err = file.Write(append(data, '\n'))
	if err != nil {
		logger.Log.Error("Failed to write audit event: %v", zap.Error(err))
	}
}
