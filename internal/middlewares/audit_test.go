package middlewares

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/max-marek-projects/shortener/internal/audit"
	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/models"
	"github.com/max-marek-projects/shortener/internal/requests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestAuditMiddleware_Notify(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auditData, ok := audit.GetAuditDataFromContext(r.Context())
		if !ok {
			logger.Log.Error("No audit data in context")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		auditData.Action = models.AuditFollow
		auditData.URL = "https://example.com"
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("ok"))
		require.NoError(t, err)
	})
	testUserID := "user123"

	mockAuditor := NewMockAudit(t)
	mockAuditor.EXPECT().NotifyAll(mock.Anything).Return((*sync.WaitGroup)(nil)).Once()

	auditHandler := AuditMiddleware(mockAuditor)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(requests.SetUserIDToContext(req.Context(), testUserID))
	w := httptest.NewRecorder()
	auditHandler.ServeHTTP(w, req)

	// check response as expected
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", w.Body.String())
}

func TestAuditMiddleware_DoNothing(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("ok"))
		require.NoError(t, err)
	})
	testUserID := "user123"

	mockAuditor := NewMockAudit(t)
	mockAuditor.AssertNotCalled(t, "NotifyAll", mock.Anything)

	auditHandler := AuditMiddleware(mockAuditor)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req = req.WithContext(requests.SetUserIDToContext(req.Context(), testUserID))
	w := httptest.NewRecorder()
	auditHandler.ServeHTTP(w, req)

	// check response as expected
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "ok", w.Body.String())
}

func BenchmarkAuditMiddleware(b *testing.B) {
	auditor := NewMockAudit(b)
	auditor.EXPECT().NotifyAll(mock.Anything).Return(nil)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if data, ok := audit.GetAuditDataFromContext(r.Context()); ok {
			data.Action = models.AuditShorten
			data.URL = "http://example.com"
		}
		w.WriteHeader(http.StatusOK)
	})
	handler := AuditMiddleware(auditor)(next)

	req := httptest.NewRequest(http.MethodPost, "/", nil)

	for b.Loop() {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

func TestGRPCAuditInterceptor_Notify(t *testing.T) {
	mockAuditor := NewMockAudit(t)
	mockAuditor.EXPECT().
		NotifyAll(mock.MatchedBy(func(event models.AuditEvent) bool {
			return event.Action == models.AuditFollow &&
				event.URL == "https://example.com"
		})).
		Return((*sync.WaitGroup)(nil)).
		Once()
	interceptor := GRPCAuditInterceptor(mockAuditor)
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		auditData, ok := audit.GetAuditDataFromContext(ctx)
		require.True(t, ok, "audit data should be in context")
		auditData.Action = models.AuditFollow
		auditData.URL = "https://example.com"
		assert.Equal(t, "", auditData.UserID)
		return "ok", nil
	}

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{FullMethod: "/test"}
	resp, err := interceptor(ctx, nil, info, handler)

	assert.NoError(t, err)
	assert.Equal(t, "ok", resp)
	mockAuditor.AssertExpectations(t)
}

func TestGRPCAuditInterceptor_DoNothing(t *testing.T) {
	mockAuditor := NewMockAudit(t)
	mockAuditor.AssertNotCalled(t, "NotifyAll", mock.Anything)
	interceptor := GRPCAuditInterceptor(mockAuditor)
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		_, ok := audit.GetAuditDataFromContext(ctx)
		require.True(t, ok, "audit data should be in context")
		return "ok", nil
	}
	ctx := context.Background()
	info := &grpc.UnaryServerInfo{FullMethod: "/test"}
	resp, err := interceptor(ctx, nil, info, handler)
	assert.NoError(t, err)
	assert.Equal(t, "ok", resp)
	mockAuditor.AssertExpectations(t)
}

func TestGRPCAuditInterceptor_WithUserID(t *testing.T) {
	mockAuditor := NewMockAudit(t)
	mockAuditor.EXPECT().
		NotifyAll(mock.MatchedBy(func(event models.AuditEvent) bool {
			return event.UserID == "user123"
		})).
		Return((*sync.WaitGroup)(nil)).
		Once()
	interceptor := GRPCAuditInterceptor(mockAuditor)
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		auditData, ok := audit.GetAuditDataFromContext(ctx)
		require.True(t, ok)
		auditData.Action = models.AuditShorten
		auditData.URL = "http://short"
		auditData.UserID = "user123"
		return "ok", nil
	}

	ctx := context.Background()
	info := &grpc.UnaryServerInfo{FullMethod: "/test"}
	resp, err := interceptor(ctx, nil, info, handler)

	assert.NoError(t, err)
	assert.Equal(t, "ok", resp)
	mockAuditor.AssertExpectations(t)
}
