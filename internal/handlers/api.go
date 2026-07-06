package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/max-marek-projects/shortener/internal/audit"
	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/models"
	"github.com/max-marek-projects/shortener/internal/requests"
	"github.com/max-marek-projects/shortener/internal/service"
	"go.uber.org/zap"
)

func (h *Handler) ShortenJSONHandler(w http.ResponseWriter, r *http.Request) {

	var requestData models.ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	auditData, ok := r.Context().Value(audit.AuditKey).(*models.AuditData)
	if !ok {
		logger.Log.Error("No audit data in context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	auditData.Action = models.AuditShorten
	auditData.URL = requestData.URL
	shortURL, err := h.service.CreateShortURL(r.Context(), requestData.URL, r.Header.Get("X-Forwarded-Proto"), r.Host)
	if err != nil {
		if errors.Is(err, service.ErrorEmptyUrl) {
			http.Error(w, "Empty url", http.StatusBadRequest)
			return
		}
		if errors.Is(err, service.ErrorDuplicate) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(models.ShortenResponse{Result: shortURL})
			return
		}
		logger.Log.Error("Failed to create short url", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	responseData := models.ShortenResponse{Result: shortURL}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(responseData); err != nil {
		logger.Log.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) PostBatchShortenHandler(w http.ResponseWriter, r *http.Request) {
	var req []models.BatchShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid json", http.StatusBadRequest)
		return
	}
	if len(req) == 0 {
		http.Error(w, "Empty json", http.StatusBadRequest)
		return
	}
	resp, err := h.service.CreateShortURLsBatch(
		r.Context(),
		req,
		r.Header.Get("X-Forwarded-Proto"),
		r.Host,
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrorEmptyUrl), errors.Is(err, service.ErrorEmptyBatch):
			http.Error(w, "Empty url", http.StatusBadRequest)
		default:
			logger.Log.Error("Failed to create short URLs for batch", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) GetUserURLsHandler(w http.ResponseWriter, r *http.Request) {
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "http"
	}
	urls, err := h.service.GetUserURLs(r.Context(), scheme, r.Host)
	if err != nil {
		logger.Log.Error("failed to get user URLs", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if len(urls) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(urls); err != nil {
		logger.Log.Error("failed to encode response", zap.Error(err))
	}
}

// delete user urls
func (h *Handler) DeleteUserURLsHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := requests.GetUserIDFromContext(r.Context())
	if !ok {
		logger.Log.Error("Missing user id in context")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	var shortIDs []string
	if err := json.NewDecoder(r.Body).Decode(&shortIDs); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if len(shortIDs) == 0 {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	var sem = make(chan struct{}, h.maxParallelWorkers)
	go func() {
		sem <- struct{}{}
		defer func() { <-sem }()
		if err := h.service.DeleteUserURLs(context.Background(), userID, shortIDs); err != nil {
			logger.Log.Error("Failed to delete user URLs", zap.Error(err))
		}
	}()

	w.WriteHeader(http.StatusAccepted)
}
