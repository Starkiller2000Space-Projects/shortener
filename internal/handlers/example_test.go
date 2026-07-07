// Package handlers_test contains example usage of the URL shortener HTTP endpoints.
// These examples appear in the godoc documentation and demonstrate how to interact
// with each API endpoint.
package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/max-marek-projects/shortener/internal/handlers"
	"github.com/max-marek-projects/shortener/internal/models"
	"github.com/max-marek-projects/shortener/internal/repository"
	"github.com/max-marek-projects/shortener/internal/service"
)

// simpleStorage is a minimal in-memory stub that implements repository.Storage.
// It is used only for documentation examples to keep them self-contained and readable.
type simpleStorage struct {
	repository.Storage
}

func (s *simpleStorage) Add(ctx context.Context, info repository.Row) error {
	// In a real implementation, this would store the URL.
	// For the example, we just return nil (success).
	return nil
}

func (s *simpleStorage) Get(ctx context.Context, id string) (string, error) {
	// Return a known URL for the example ID "abc123".
	if id == "abc123" {
		return "https://example.com", nil
	}
	return "", repository.ErrNotFound
}

func (s *simpleStorage) GetUserURLs(ctx context.Context, userID string) ([]repository.UserURL, error) {
	// Return a fake list of URLs for the authenticated user.
	return []repository.UserURL{
		{ShortURL: "abc123", OriginalURL: "https://example.com"},
		{ShortURL: "def456", OriginalURL: "https://golang.org"},
	}, nil
}

func (s *simpleStorage) DeleteBatch(ctx context.Context, userID string, shortIDs []string) error {
	// Simulate successful deletion.
	return nil
}

func (s *simpleStorage) Ping(ctx context.Context) error {
	// Simulate a healthy storage backend.
	return nil
}

// ExampleHandler_PostURLHandler demonstrates creating a short URL using plain text.
// This endpoint accepts a raw URL in the request body and returns the shortened URL.
func ExampleHandler_PostURLHandler() {
	store := &simpleStorage{}
	svc := service.NewEndpointService(store, "http://localhost:8080", 8)
	h := handlers.NewHandler(svc, 10)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte("https://example.com")))
	w := httptest.NewRecorder()

	h.PostURLHandler(w, req)

	// In a real scenario, the response body would contain the shortened URL.
	// This example is only for documentation purposes.
}

// ExampleHandler_ShortenJSONHandler demonstrates creating a short URL via JSON.
// It sends a JSON object with the "url" field and receives a JSON response with the result.
func ExampleHandler_ShortenJSONHandler() {
	store := &simpleStorage{}
	svc := service.NewEndpointService(store, "http://localhost:8080", 8)
	h := handlers.NewHandler(svc, 10)

	body := models.ShortenRequest{URL: "https://example.com"}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ShortenJSONHandler(w, req)

	// The response is JSON with the "result" field containing the short URL.
}

// ExampleHandler_PostBatchShortenHandler demonstrates batch creation of short URLs.
// It accepts a JSON array of objects with "correlation_id" and "original_url" fields.
// The response contains the same correlation IDs paired with the generated short URLs.
func ExampleHandler_PostBatchShortenHandler() {
	store := &simpleStorage{}
	svc := service.NewEndpointService(store, "http://localhost:8080", 8)
	h := handlers.NewHandler(svc, 10)

	batch := []models.BatchShortenRequest{
		{CorrelationID: "1", OriginalURL: "https://example.com"},
		{CorrelationID: "2", OriginalURL: "https://golang.org"},
	}
	data, _ := json.Marshal(batch)
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.PostBatchShortenHandler(w, req)

	// The response is a JSON array with the same correlation IDs and generated short URLs.
}

// ExampleHandler_IDHandler demonstrates redirection by short ID.
// It responds with a 307 Temporary Redirect and sets the Location header to the original URL.
func ExampleHandler_IDHandler() {
	store := &simpleStorage{}
	svc := service.NewEndpointService(store, "", 8) // no base URL, use host from request
	h := handlers.NewHandler(svc, 10)

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	w := httptest.NewRecorder()

	h.IDHandler(w, req)

	// The response contains a Location header with the original URL and status 307.
	// Note: In this example, the ID is passed via URL param; in real usage it comes from chi.
}

// ExampleHandler_GetUserURLsHandler demonstrates retrieving all URLs created by the authenticated user.
// It returns a JSON array of objects containing both short and original URLs.
func ExampleHandler_GetUserURLsHandler() {
	store := &simpleStorage{}
	svc := service.NewEndpointService(store, "http://localhost:8080", 8)
	h := handlers.NewHandler(svc, 10)

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	w := httptest.NewRecorder()

	// In a real request, the user would be authenticated via cookie middleware.
	// This example only shows the handler logic.
	h.GetUserURLsHandler(w, req)

	// On success, returns 200 OK with JSON array.
	// If no URLs, returns 204 No Content.
}

// ExampleHandler_DeleteUserURLsHandler demonstrates deleting multiple URLs for a user.
// It accepts a JSON array of short IDs and returns 202 Accepted immediately.
// The actual deletion happens asynchronously in the background.
func ExampleHandler_DeleteUserURLsHandler() {
	store := &simpleStorage{}
	svc := service.NewEndpointService(store, "http://localhost:8080", 8)
	h := handlers.NewHandler(svc, 10)

	shortIDs := []string{"abc123", "def456"}
	data, _ := json.Marshal(shortIDs)
	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.DeleteUserURLsHandler(w, req)

	// Response always returns 202 Accepted, deletion is performed asynchronously.
}

// ExampleHandler_PingHandler demonstrates checking the health of the storage backend.
// Returns 200 OK if the storage is reachable, otherwise 500 Internal Server Error.
func ExampleHandler_PingHandler() {
	store := &simpleStorage{}
	svc := service.NewEndpointService(store, "http://localhost:8080", 8)
	h := handlers.NewHandler(svc, 10)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	h.PingHandler(w, req)

	// Returns 200 OK on success.
}
