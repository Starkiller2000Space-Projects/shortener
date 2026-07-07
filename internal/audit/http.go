// Package audit implements an audit logging system with multiple observers.

package audit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/models"
	"go.uber.org/zap"
)

// HTTPAuditObserver sends audit events as JSON POST requests to a specified URL.
type HTTPAuditObserver struct {
	url    string
	client *http.Client
}

// NewHTTPAuditObserver creates a new HTTP audit observer with the given endpoint URL.
// The HTTP client uses a 5-second timeout.
func NewHTTPAuditObserver(url string) *HTTPAuditObserver {
	return &HTTPAuditObserver{
		url:    url,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// Notify sends the audit event to the configured URL via HTTP POST.
// If the server returns a non-2xx status, an error is logged.
func (h *HTTPAuditObserver) Notify(event models.AuditEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		logger.Log.Error("Failed to marshal audit event: %v", zap.Error(err))
		return
	}

	resp, err := h.client.Post(h.url, "application/json", bytes.NewReader(data))
	if err != nil {
		logger.Log.Error("Failed to send audit event: %v", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Log.Error("Audit server returned non-2xx status: %d", zap.Int("code", resp.StatusCode))
	}
}
