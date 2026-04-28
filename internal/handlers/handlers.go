package handlers

import (
	"io"
	"net/http"

	"github.com/Starkiller2000Space-Projects/shortener/internal/service"
	"github.com/go-chi/chi/v5"
)

// Endpoints handler
type Handler struct {
	service service.EndpointService // абстракция!
}

// Get new endpoints handler
func NewHandler(service service.EndpointService) *Handler {
	return &Handler{service: service}
}

// handle passed id GET `/{id}`
func (h *Handler) IdHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	id := chi.URLParam(r, "id")
	w.Header().Set("Content-Type", "text/plain")
	url, err := h.service.GetOriginalURL(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Location", url)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// handle adding url to storage POST `/`
func (h *Handler) PostUrlHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	shortURL, err := h.service.CreateShortURL(r.Context(), string(body), r.Header.Get("X-Forwarded-Proto"), r.Host)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}
