package main

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"sync"
)

// storage for all ids and corresponding urls
type storage struct {
	mu   sync.RWMutex
	data map[string]string
	idSize int
}

// create new storage with desired size of ids
func newStorage(idSize int) *storage {
	return &storage{
		data: make(map[string]string),
		idSize: idSize,
	}
}

// add url into storage and return generated id
func (store *storage) add(value string) string {
	store.mu.Lock() // lock storage for writing
	defer store.mu.Unlock() // unlock storage for writing after function completion
	id := generateId(store.idSize)
	store.data[id] = value
	return id
}

// get url by id from storage if exists
func (store *storage) get(id string) (string, bool) {
	store.mu.RLock() // lock storage for reading
	defer store.mu.RUnlock() // unlock storage for reading after function completion
	value, ok := store.data[id]
	return value, ok
}

// generate ID  for url with desired size
func generateId(size int) string {
	b := make([]byte, size) // empty bytes array
	_, _ = rand.Read(b)  // fill bytes array with random bytes
	return base64.RawURLEncoding.EncodeToString(b)[:size] // encode bytes array to string
}

// handle passed id GET `/{id}`
func idHandler(w http.ResponseWriter, r *http.Request, store *storage) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	id := r.PathValue("id")
	w.Header().Set("Content-Type", "text/plain")
	url, ok := store.get(id)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	_, _ = w.Write([]byte(url))
}

// handle adding url to storage POST `/`
func postUrlHandler(w http.ResponseWriter, r *http.Request, store *storage) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
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
	shortID := store.add(longURL)
	scheme := "http"
	if r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	shortURL := scheme + "://" + r.Host + "/" + shortID
	_, _ = w.Write([]byte(shortURL))
}

// entry point
func main() {
	idSize := 8
	store := newStorage(idSize)
	mux := http.NewServeMux()
	mux.HandleFunc(`/{id}`, func(w http.ResponseWriter, r *http.Request) {idHandler(w, r, store)})
	mux.HandleFunc(`/`, func(w http.ResponseWriter, r *http.Request) {postUrlHandler(w, r, store)})
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		panic(err)
	}
}
