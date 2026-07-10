package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/max-marek-projects/shortener/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// test file audit notification
func TestFileAuditObserver_Notify(t *testing.T) {
	// create temp file
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "audit.log")
	// create observer
	observer, err := NewFileAuditObserver(filePath)
	defer observer.Stop()
	require.NoError(t, err)
	// notify
	event := models.AuditEvent{
		Timestamp: 1234567890,
		UserID:    "123",
		Action:    models.AuditShorten,
		URL:       "https://example.com",
	}
	observer.Notify(event)
	// check value added
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	lines := string(data)
	require.NotEmpty(t, lines)
	var received models.AuditEvent
	err = json.Unmarshal(data, &received)
	require.NoError(t, err)
	assert.Equal(t, event, received)
}
