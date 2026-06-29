package audit

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/models"
	"go.uber.org/zap"
)

// file audit observer
type FileAuditObserver struct {
	filePath string
	mu       sync.Mutex
}

// get new file audit observer
func NewFileAuditObserver(path string) *FileAuditObserver {
	return &FileAuditObserver{filePath: path}
}

// notify in file
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
