package handlers

import (
	"io"
	"net/http"
	"strings"

	"github.com/Starkiller2000Space-Projects/shortener/internal/storage"
	"github.com/go-chi/chi/v5"
)

// Endpoints handler
type Handler struct {
	store storage.StorageInterface
	showAddr string
}

// Get new endpoints handler
func NewHandler(store storage.StorageInterface, showAddr string) *Handler {
	return &Handler{store: store, showAddr: showAddr}
}

// handle passed id GET `/{id}`
func (h *Handler) IdHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	id := chi.URLParam(r, "id")
	w.Header().Set("Content-Type", "text/plain")
	url, ok := h.store.Get(id)
	if !ok {
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
	longURL := strings.TrimSpace(string(body))
	if longURL == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	shortID := h.store.Add(longURL)
	var shortURL string
	if h.showAddr != "" {
		shortURL = h.showAddr
	} else {
		scheme := "http"
		if r.Header.Get("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		shortURL = scheme + "://" + r.Host + "/" + shortID 
	}
	w.Header().Set("Content-Type", "text/plain")
    w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(shortURL))
}
