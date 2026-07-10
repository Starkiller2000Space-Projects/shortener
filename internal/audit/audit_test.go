package audit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/max-marek-projects/shortener/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// test audit initialization
func TestInitAudit(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")
	tests := []struct {
		name     string
		filePath string
		url      string
		wantFile bool
		wantHTTP bool
	}{
		{"no observers", "", "", false, false},
		{"file only", logPath, "", true, false},
		{"http only", "", "http://example.com", false, true},
		{"both", logPath, "http://example.com", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auditPtr, err := InitAudit(tt.filePath, tt.url, 100)
			defer auditPtr.Stop()
			require.NoError(t, err)
			wantLen := 0
			for _, wantValue := range []bool{tt.wantFile, tt.wantHTTP} {
				if wantValue {
					wantLen++
				}
			}
			assert.Len(t, auditPtr.observers, wantLen)
			foundFile := false
			foundHTTP := false
			for _, observer := range auditPtr.observers {
				if _, ok := observer.(*fileAuditObserver); ok {
					foundFile = true
				}
				if _, ok := observer.(*httpAuditObserver); ok {
					foundHTTP = true
				}
			}
			assert.Equal(t, tt.wantFile, foundFile)
			assert.Equal(t, tt.wantHTTP, foundHTTP)
		})
	}
}

// Test register and all observers notification
func TestAudit_RegisterAndNotifyAll(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "audit.log")
	a, err := InitAudit(logPath, "", 100)
	defer a.Stop()
	require.NoError(t, err)
	mock1 := NewMockObserver(t)
	mock1.EXPECT().Notify(mock.Anything).Once()
	mock1.EXPECT().Stop().Return(nil)
	mock2 := NewMockObserver(t)
	mock2.EXPECT().Notify(mock.Anything).Once()
	mock2.EXPECT().Stop().Return(nil)
	a.Register(mock1)
	a.Register(mock2)
	event := models.AuditEvent{UserID: "test", Action: "login"}
	wg := a.NotifyAll(event)
	wg.Wait()
}

func BenchmarkFileAuditNotify(b *testing.B) {
	tmpFile, _ := os.CreateTemp("", "audit_*.log")
	defer os.Remove(tmpFile.Name())
	observer, err := NewFileAuditObserver(tmpFile.Name())
	defer observer.Stop()
	require.NoError(b, err)
	event := models.AuditEvent{Timestamp: 123, Action: "shorten", UserID: "user", URL: "http://example.com"}

	for b.Loop() {
		observer.Notify(event)
	}
}
