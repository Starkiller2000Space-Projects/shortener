package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
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
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Location", url)
	ctx := context.WithValue(r.Context(), "audit_data", models.AuditData{
		Action: models.AuditFollow,
		URL:    url,
	})
	r = r.WithContext(ctx)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// handle adding url to storage POST `/`
func (h *Handler) PostUrlHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	originalUrl := string(body)
	ctx := context.WithValue(r.Context(), "audit_data", models.AuditData{
		Action: models.AuditShorten,
		URL:    originalUrl,
	})
	r = r.WithContext(ctx)
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
