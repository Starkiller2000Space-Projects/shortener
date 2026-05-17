package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/stretchr/testify/assert"
)

func TestRequestsLogger(t *testing.T) {
	logger.Initialize("INFO")
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	loggedHandler := RequestsLogger(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	loggedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", w.Body.String())
}
