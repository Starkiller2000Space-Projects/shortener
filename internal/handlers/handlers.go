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

// Endpoints handler
type Handler struct {
	service            service.Service
	maxParallelWorkers int
}

// Get new endpoints handler
func NewHandler(service service.Service, maxParallelWorkers int) *Handler {
	return &Handler{service: service, maxParallelWorkers: maxParallelWorkers}
}

// handle passed id GET `/{id}`
func (h *Handler) IdHandler(w http.ResponseWriter, r *http.Request) {
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

// handle adding url to storage POST `/`
func (h *Handler) PostUrlHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, "Empty body", http.StatusBadRequest)
		return
	}
	originalUrl := string(body)
	auditData, ok := r.Context().Value(audit.AuditKey).(*models.AuditData)
	if !ok {
		logger.Log.Error("No audit data in context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	auditData.Action = models.AuditShorten
	auditData.URL = originalUrl
	shortURL, err := h.service.CreateShortURL(r.Context(), originalUrl, r.Header.Get("X-Forwarded-Proto"), r.Host)
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

func (h *Handler) PingHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Ping(r.Context()); err != nil {
		logger.Log.Error("Failed to ping storage", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
