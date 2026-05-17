package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/models"
	"github.com/max-marek-projects/shortener/internal/service"
	"go.uber.org/zap"
)

func (h *Handler) ShortenJSONHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var requestData models.ShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		logger.Log.Debug("cannot decode request JSON body", zap.Error(err))
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	shortURL, err := h.service.CreateShortURL(r.Context(), requestData.URL, r.Header.Get("X-Forwarded-Proto"), r.Host)
	if err != nil {
		if errors.Is(err, service.ErrorEmptyUrl) {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	responseData := models.ShortenResponse{Result: shortURL}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(responseData); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) PostBatchShortenHandler(w http.ResponseWriter, r *http.Request) {
	var req []models.BatchShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if len(req) == 0 {
		http.Error(w, "bad request", http.StatusBadRequest)
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
			http.Error(w, "bad request", http.StatusBadRequest)
		default:
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}
