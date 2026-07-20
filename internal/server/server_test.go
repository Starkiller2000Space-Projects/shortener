package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/max-marek-projects/shortener/internal/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestServerRoutes(t *testing.T) {
	mockService := NewMockService(t)
	mockService.EXPECT().
		Ping(mock.Anything).
		Return(nil)
	mockAuditor := NewMockAudit(t)

	h := handlers.NewHandler(mockService, 10)
	srv := NewServer("", h, 1*time.Second, 1*time.Second, mockAuditor, "secret")
	handler := srv.Handler
	ts := httptest.NewServer(handler)
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/ping")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
