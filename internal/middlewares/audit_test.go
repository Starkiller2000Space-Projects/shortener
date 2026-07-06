package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/max-marek-projects/shortener/internal/audit"
	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/models"
	"github.com/max-marek-projects/shortener/internal/requests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAuditMiddleware_Notify(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auditData, ok := r.Context().Value(audit.AuditKey).(*models.AuditData)
		if !ok {
			logger.Log.Error("No audit data in context")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		auditData.Action = models.AuditFollow
		auditData.URL = "https://example.com"
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	testUserID := "user123"

	mockAuditor := NewMockAudit(t)
	mockAuditor.EXPECT().NotifyAll(mock.Anything).Once()

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
		w.Write([]byte("ok"))
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
	auditor.EXPECT().NotifyAll(mock.Anything).Return()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if data, ok := r.Context().Value(audit.AuditKey).(*models.AuditData); ok {
			data.Action = models.AuditShorten
			data.URL = "http://example.com"
		}
		w.WriteHeader(http.StatusOK)
	})
	handler := AuditMiddleware(auditor)(next)

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}
