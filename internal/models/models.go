// models package defines data structures used across the application.

package models

// generate:reset
//
// ShortenRequest is the JSON body for POST /api/shorten.
type ShortenRequest struct {
	URL string `json:"url"`
}

// generate:reset
//
// ShortenResponse is the JSON response for POST /api/shorten.
type ShortenResponse struct {
	Result string `json:"result"`
}

// generate:reset
//
// BatchShortenRequest is the JSON body for POST /api/shorten/batch.
type BatchShortenRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// generate:reset
//
// BatchShortenResponse is the JSON response for POST /api/shorten/batch.
type BatchShortenResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// generate:reset
//
// UserURL represents a user's URL list entry.
type UserURL struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}
