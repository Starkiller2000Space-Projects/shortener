package middlewares

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/max-marek-projects/shortener/internal/auth"
	"github.com/max-marek-projects/shortener/internal/requests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware_Authorized(t *testing.T) {
	testUserID := "user123"
	testSecretKey := "12345"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	authHandler := AuthMiddleware(testSecretKey)(handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	err := auth.SetUserCookie(w, testUserID, testSecretKey)
	require.NoError(t, err)
	authHandler.ServeHTTP(w, req)
	// check response as expected
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", w.Body.String())
}

// test cookie addition to request
func TestAuthMiddleware_Unauthorized(t *testing.T) {
	testSecretKey := "12345"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
		contextUserID, ok := requests.GetUserIDFromContext(r.Context())
		assert.True(t, ok)
		assert.NotEqual(t, contextUserID, "")
	})
	authHandler := AuthMiddleware(testSecretKey)(handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	authHandler.ServeHTTP(w, req)
	// check response as expected
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", w.Body.String())
}

// test invalid user cookie
func TestAuthMiddleware_InvalidCookie(t *testing.T) {
	testSecretKey := "12345"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	authHandler := AuthMiddleware(testSecretKey)(handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	req.AddCookie(&http.Cookie{
		Name:     "token",
		Value:    "invalid token",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	authHandler.ServeHTTP(w, req)
	// check response as expected
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, fmt.Sprintf("%s\n", http.StatusText(http.StatusUnauthorized)), w.Body.String())
}

func BenchmarkAuthMiddleware_NewUser(b *testing.B) {
	secret := "test-secret"
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handler := AuthMiddleware(secret)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkAuthMiddleware_ExistingUser(b *testing.B) {
	secret := "test-secret"
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	handler := AuthMiddleware(secret)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	response := w.Result()
	defer response.Body.Close()
	cookie := response.Cookies()[0]

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		req2.AddCookie(cookie)
		w2 := httptest.NewRecorder()
		handler.ServeHTTP(w2, req2)
	}
}
