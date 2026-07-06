package audit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/max-marek-projects/shortener/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestHTTPAuditObserver_Notify_Success(t *testing.T) {
	// create test http server
	var receivedEvent models.AuditEvent
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		var event models.AuditEvent
		err := json.NewDecoder(r.Body).Decode(&event)
		assert.NoError(t, err)
		receivedEvent = event
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	// create observer
	observer := NewHTTPAuditObserver(server.URL)
	event := models.AuditEvent{
		Timestamp: 1234567890,
		UserID:    "123",
		Action:    models.AuditShorten,
		URL:       "https://example.com",
	}
	observer.Notify(event)
	assert.Equal(t, event, receivedEvent)
}

// test wrong audit server response
func TestHTTPAuditObserver_Notify_Non2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	observer := NewHTTPAuditObserver(server.URL)
	event := models.AuditEvent{
		Timestamp: 1234567890,
		UserID:    "123",
		Action:    models.AuditShorten,
		URL:       "https://example.com",
	}
	assert.NotPanics(t, func() {
		observer.Notify(event)
	})
}
