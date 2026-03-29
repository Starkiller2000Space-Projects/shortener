package handlers

import (
	"io"
	"net/http"
	"strings"

	"github.com/Starkiller2000Space-Projects/shortener/cmd/shortener/internal/storage"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	store storage.StorageInterface
}

func NewHandler(store storage.StorageInterface) *Handler {
	return &Handler{store: store}
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
	scheme := "http"
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	shortURL := scheme + "://" + r.Host + "/" + shortID
	w.Header().Set("Content-Type", "text/plain")
    w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(shortURL))
}
