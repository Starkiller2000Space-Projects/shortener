package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/max-marek-projects/shortener/internal/auth"
	"github.com/max-marek-projects/shortener/internal/requests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuthMiddleware_Authorized(t *testing.T) {
	testUserID := "user123"
	testSecretKey := "12345"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("ok"))
		require.NoError(t, err)
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
		_, err := w.Write([]byte("ok"))
		require.NoError(t, err)
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
		_, err := w.Write([]byte("ok"))
		require.NoError(t, err)
	})
	authHandler := AuthMiddleware(testSecretKey)(handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	req.AddCookie(&http.Cookie{ // #nosec G124
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

	for b.Loop() {
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

	for b.Loop() {
		req2 := httptest.NewRequest(http.MethodGet, "/", nil)
		req2.AddCookie(cookie)
		w2 := httptest.NewRecorder()
		handler.ServeHTTP(w2, req2)
	}
}

func genToken(userID, secret string) string {
	claims := auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
		UserID:           userID,
	}
	t, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	return t
}

func TestGRPCAuthInterceptor(t *testing.T) {
	secret := "secret"
	interceptor := GRPCAuthInterceptor(secret)

	t.Run("public method skip", func(t *testing.T) {
		handler := func(ctx context.Context, req interface{}) (interface{}, error) { return "ok", nil }
		info := &grpc.UnaryServerInfo{FullMethod: "/shortener.ShortenerService/ExpandURL"}
		resp, err := interceptor(context.Background(), nil, info, handler)
		assert.NoError(t, err)
		assert.Equal(t, "ok", resp)
	})

	t.Run("valid token", func(t *testing.T) {
		md := metadata.Pairs("authorization", "Bearer "+genToken("user", secret))
		ctx := metadata.NewIncomingContext(context.Background(), md)
		handler := func(ctx context.Context, req interface{}) (interface{}, error) { return "ok", nil }
		info := &grpc.UnaryServerInfo{FullMethod: "/shortener.ShortenerService/ShortenURL"}
		resp, err := interceptor(ctx, nil, info, handler)
		assert.NoError(t, err)
		assert.Equal(t, "ok", resp)
	})

	t.Run("missing metadata", func(t *testing.T) {
		handler := func(ctx context.Context, req interface{}) (interface{}, error) { return nil, nil }
		info := &grpc.UnaryServerInfo{FullMethod: "/shortener.ShortenerService/ShortenURL"}
		_, err := interceptor(context.Background(), nil, info, handler)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.Unauthenticated, st.Code())
	})

	t.Run("missing auth header", func(t *testing.T) {
		ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("other", "val"))
		handler := func(ctx context.Context, req interface{}) (interface{}, error) { return nil, nil }
		info := &grpc.UnaryServerInfo{FullMethod: "/shortener.ShortenerService/ShortenURL"}
		_, err := interceptor(ctx, nil, info, handler)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.Unauthenticated, st.Code())
	})

	t.Run("invalid scheme", func(t *testing.T) {
		md := metadata.Pairs("authorization", "Basic token")
		ctx := metadata.NewIncomingContext(context.Background(), md)
		handler := func(ctx context.Context, req interface{}) (interface{}, error) { return nil, nil }
		info := &grpc.UnaryServerInfo{FullMethod: "/shortener.ShortenerService/ShortenURL"}
		_, err := interceptor(ctx, nil, info, handler)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.Unauthenticated, st.Code())
	})

	t.Run("empty token", func(t *testing.T) {
		md := metadata.Pairs("authorization", "Bearer ")
		ctx := metadata.NewIncomingContext(context.Background(), md)
		handler := func(ctx context.Context, req interface{}) (interface{}, error) { return nil, nil }
		info := &grpc.UnaryServerInfo{FullMethod: "/shortener.ShortenerService/ShortenURL"}
		_, err := interceptor(ctx, nil, info, handler)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.Unauthenticated, st.Code())
	})

	t.Run("invalid token", func(t *testing.T) {
		md := metadata.Pairs("authorization", "Bearer invalid")
		ctx := metadata.NewIncomingContext(context.Background(), md)
		handler := func(ctx context.Context, req interface{}) (interface{}, error) { return nil, nil }
		info := &grpc.UnaryServerInfo{FullMethod: "/shortener.ShortenerService/ShortenURL"}
		_, err := interceptor(ctx, nil, info, handler)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.Unauthenticated, st.Code())
	})
}
