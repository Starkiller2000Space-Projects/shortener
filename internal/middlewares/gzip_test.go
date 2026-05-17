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
		w.Write([]byte(`{"test":true}`))
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
		gz.Write([]byte(`{"data":"compressed"}`))
		gz.Close()

		req := httptest.NewRequest(http.MethodPost, "/", &buf)
		req.Header.Set("Content-Encoding", "gzip")
		w := httptest.NewRecorder()
		gzipHandler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// response should be normal
		assert.Equal(t, `{"test":true}`, w.Body.String())
	})
}
