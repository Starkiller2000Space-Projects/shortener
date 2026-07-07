package audit

import (
	"os"
	"testing"
	"time"

	"github.com/max-marek-projects/shortener/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// test audit initialization
func TestInitAudit(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		url      string
		wantFile bool
		wantHTTP bool
	}{
		{"no observers", "", "", false, false},
		{"file only", "/tmp/audit.log", "", true, false},
		{"http only", "", "http://example.com", false, true},
		{"both", "/tmp/audit.log", "http://example.com", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auditPtr := InitAudit(tt.filePath, tt.url)
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
				if _, ok := observer.(*FileAuditObserver); ok {
					foundFile = true
				}
				if _, ok := observer.(*HTTPAuditObserver); ok {
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
	a := &audit{observers: []Observer{}}
	mock1 := NewMockObserver(t)
	mock1.EXPECT().Notify(mock.Anything).Once()
	mock2 := NewMockObserver(t)
	mock2.EXPECT().Notify(mock.Anything).Once()
	a.Register(mock1)
	a.Register(mock2)
	event := models.AuditEvent{UserID: "test", Action: "login"}
	a.NotifyAll(event)
	time.Sleep(100 * time.Millisecond)
}

func BenchmarkFileAuditNotify(b *testing.B) {
	tmpFile, _ := os.CreateTemp("", "audit_*.log")
	defer os.Remove(tmpFile.Name())
	observer := NewFileAuditObserver(tmpFile.Name())
	event := models.AuditEvent{Timestamp: 123, Action: "shorten", UserID: "user", URL: "http://example.com"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		observer.Notify(event)
	}
}
