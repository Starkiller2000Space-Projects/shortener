// Package handlers implements HTTP endpoints for URL shortening and redirection.

package handlers

import (
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/max-marek-projects/shortener/internal/audit"
	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/models"
	"github.com/max-marek-projects/shortener/internal/repository"
	"github.com/max-marek-projects/shortener/internal/service"
	"go.uber.org/zap"
)

// Handler handles HTTP endpoints for URL shortening and redirection.
type Handler struct {
	service            service.Service
	maxParallelWorkers int
}

// NewHandler creates a new Handler with the given service and max parallel workers.
func NewHandler(service service.Service, maxParallelWorkers int) *Handler {
	return &Handler{service: service, maxParallelWorkers: maxParallelWorkers}
}

// IDHandler handles GET /{id} – redirects to the original URL.
// If the ID does not exist, returns 400 Bad Request.
// If the URL has been deleted, returns 410 Gone.
// It also records an audit follow action.
func (h *Handler) IDHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	w.Header().Set("Content-Type", "text/plain")
	url, err := h.service.GetOriginalURL(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrGone) {
			w.WriteHeader(http.StatusGone)
			return
		}
		http.Error(w, "Wrong id", http.StatusBadRequest)
		return
	}
	auditData, ok := r.Context().Value(audit.AuditKey).(*models.AuditData)
	if !ok {
		logger.Log.Error("No audit data in context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	auditData.Action = models.AuditFollow
	auditData.URL = url
	w.Header().Set("Location", url)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// PostURLHandler handles POST / – creates a short URL from the plain text body.
// The body should contain the original URL.
// On success, returns 201 Created with the short URL in plain text.
// If the URL already exists, returns 409 Conflict with the existing short URL.
func (h *Handler) PostURLHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, "Empty body", http.StatusBadRequest)
		return
	}
	originalURL := string(body)
	auditData, ok := r.Context().Value(audit.AuditKey).(*models.AuditData)
	if !ok {
		logger.Log.Error("No audit data in context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	auditData.Action = models.AuditShorten
	auditData.URL = originalURL
	shortURL, err := h.service.CreateShortURL(r.Context(), originalURL, r.Header.Get("X-Forwarded-Proto"), r.Host)
	if err != nil {
		if errors.Is(err, service.ErrorEmptyUrl) {
			http.Error(w, "Empty url", http.StatusBadRequest)
			return
		}
		if errors.Is(err, service.ErrorDuplicate) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(shortURL))
			return
		}
		logger.Log.Error("Failed create short URL", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

// PingHandler handles GET /ping – checks the storage availability.
// Returns 200 OK if the storage is reachable, otherwise 500 Internal Server Error.
func (h *Handler) PingHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Ping(r.Context()); err != nil {
		logger.Log.Error("Failed to ping storage", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
