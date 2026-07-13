package middlewares

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGzipMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"test":true}`))
		require.NoError(t, err)
	})
	gzipHandler := GzipMiddleware(handler)

	t.Run("accept-encoding gzip", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()
		gzipHandler.ServeHTTP(w, req)

		assert.Equal(t, "gzip", w.Header().Get("Content-Encoding"))
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		// decode
		gr, err := gzip.NewReader(w.Body)
		require.NoError(t, err)
		body, err := io.ReadAll(gr)
		require.NoError(t, err)
		assert.Equal(t, `{"test":true}`, string(body))
	})

	t.Run("content-encoding gzip", func(t *testing.T) {
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		_, err := gz.Write([]byte(`{"data":"compressed"}`))
		require.NoError(t, err)
		err = gz.Close()
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/", &buf)
		req.Header.Set("Content-Encoding", "gzip")
		w := httptest.NewRecorder()
		gzipHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// response should be normal
		assert.Equal(t, `{"test":true}`, w.Body.String())
	})
}

func BenchmarkGzipMiddleware_NoCompression(b *testing.B) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("hello world"))
		require.NoError(b, err)
	})
	handler := GzipMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	for b.Loop() {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkGzipMiddleware_WithAcceptGzip(b *testing.B) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"status":"ok"}`))
		require.NoError(b, err)
	})
	handler := GzipMiddleware(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	for b.Loop() {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkGzipMiddleware_WithGzipBody(b *testing.B) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write([]byte(`{"data":"test"}`))
	require.NoError(b, err)
	err = gz.Close()
	require.NoError(b, err)
	body := buf.Bytes()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := GzipMiddleware(next)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Encoding", "gzip")

	for b.Loop() {
		w := httptest.NewRecorder()
		req.Body = io.NopCloser(bytes.NewReader(body))
		handler.ServeHTTP(w, req)
	}
}
