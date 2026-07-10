package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoggerMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	loggedHandler := LoggerMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	loggedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", w.Body.String())
}

func BenchmarkLoggerMiddleware(b *testing.B) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handler := LoggerMiddleware(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	for b.Loop() {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}
