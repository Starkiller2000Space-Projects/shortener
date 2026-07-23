// Package audit implements an audit logging system with multiple observers.
package audit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/models"
	"go.uber.org/zap"
)

// httpAuditObserver sends audit events as JSON POST requests to a specified URL.
type httpAuditObserver struct {
	url    string
	client *http.Client
}

// NewHTTPAuditObserver creates a new HTTP audit observer with the given endpoint URL.
// The HTTP client uses a 5-second timeout.
func NewHTTPAuditObserver(url string) *httpAuditObserver {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.RetryWaitMin = 1 * time.Second
	retryClient.RetryWaitMax = 5 * time.Second
	retryClient.HTTPClient.Timeout = 5 * time.Second
	stdClient := retryClient.StandardClient()
	return &httpAuditObserver{
		url:    url,
		client: stdClient,
	}
}

// Notify sends the audit event to the configured URL via HTTP POST.
// If the server returns a non-2xx status, an error is logged.
func (h *httpAuditObserver) Notify(event models.AuditEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		logger.Log.Error("failed to marshal audit event", zap.Error(err))
		return
	}

	resp, err := h.client.Post(h.url, "application/json", bytes.NewReader(data))
	if err != nil {
		logger.Log.Error("failed to send audit event", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Log.Error("audit server returned non-2xx status", zap.Int("code", resp.StatusCode))
	}
}

// close http observer
func (h *httpAuditObserver) Stop() error { return nil }
