package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/max-marek-projects/shortener/internal/audit"
	"github.com/max-marek-projects/shortener/internal/models"
	"github.com/max-marek-projects/shortener/internal/repository"
	"github.com/max-marek-projects/shortener/internal/requests"
	"github.com/max-marek-projects/shortener/internal/service"
)

// create new router for handlers testing
func newTestRouter(service *MockService, withAudit, withAuth bool) http.Handler {
	h := NewHandler(service, 100)

	r := chi.NewRouter()
	if withAudit {
		r.Use(withTestAudit)
	}
	if withAuth {
		r.Use(withTestAuth)
	}
	r.Get("/ping", h.PingHandler)
	r.Get("/{id}", h.IDHandler)
	r.Post("/", h.PostURLHandler)
	r.Route("/api", func(api chi.Router) {
		api.Post("/shorten", h.ShortenJSONHandler)
		api.Post("/shorten/batch", h.PostBatchShortenHandler)
		api.Get("/user/urls", h.GetUserURLsHandler)
		api.Delete("/user/urls", h.DeleteUserURLsHandler)
	})
	return r
}

func withTestAudit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auditData := &models.AuditData{}
		ctx := context.WithValue(r.Context(), audit.AuditKey, auditData)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

var testUserID string = "123user"

func withTestAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := requests.SetUserIDToContext(r.Context(), testUserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type requestOption func(*http.Request)

// test single request
func testRequest(t *testing.T, ts *httptest.Server, method, path, body string, options ...requestOption) (*http.Response, string) {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = bytes.NewBufferString(body)
	}

	req, err := http.NewRequest(method, ts.URL+path, reader)
	require.NoError(t, err)

	for _, opt := range options {
		opt(req)
	}

	client := ts.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := client.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}

// test getting url by its id
func TestIDHandler(t *testing.T) {
	fixedID := "test1234"
	existingURL := "https://example.com/"

	type want struct {
		code        int
		contentType string
		location    string
	}
	type serviceData struct {
		value string
		err   error
	}
	tests := []struct {
		name    string
		method  string
		service *serviceData
		audit   bool
		want    want
	}{
		{
			name:   "positive test",
			method: http.MethodGet,
			service: &serviceData{
				value: existingURL,
				err:   nil,
			},
			audit: true,
			want: want{
				code:        http.StatusTemporaryRedirect,
				contentType: "text/plain",
				location:    existingURL,
			},
		},
		{
			name:   "wrong method test",
			method: http.MethodPost,
			audit:  true,
			want: want{
				code:        http.StatusMethodNotAllowed,
				contentType: "",
				location:    "",
			},
		},
		{
			name:   "missing id test",
			method: http.MethodGet,
			audit:  true,
			service: &serviceData{
				value: "",
				err:   repository.ErrNotFound,
			},
			want: want{
				code:        http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				location:    "",
			},
		},
		{
			name:   "gone test",
			method: http.MethodGet,
			audit:  true,
			service: &serviceData{
				value: "",
				err:   repository.ErrGone,
			},
			want: want{
				code:        http.StatusGone,
				contentType: "text/plain",
				location:    "",
			},
		},
		{
			name:   "no audit",
			method: http.MethodGet,
			service: &serviceData{
				value: existingURL,
				err:   nil,
			},
			audit: false,
			want: want{
				code:        http.StatusInternalServerError,
				contentType: "text/plain; charset=utf-8",
				location:    "",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockService := NewMockService(t)
			if test.service != nil {
				mockService.EXPECT().GetOriginalURL(mock.Anything, fixedID).Return(test.service.value, test.service.err)
			} else {
				mockService.AssertNotCalled(t, "GetOriginalURL", mock.Anything, mock.Anything)
			}
			ts := httptest.NewServer(newTestRouter(mockService, test.audit, false))
			defer ts.Close()
			resp, _ := testRequest(t, ts, test.method, fmt.Sprintf("/%v", fixedID), "")
			defer resp.Body.Close()
			assert.Equal(t, test.want.code, resp.StatusCode)
			assert.Equal(t, test.want.contentType, resp.Header.Get("Content-Type"))
			assert.Equal(t, test.want.location, resp.Header.Get("Location"))
		})
	}
}

func BenchmarkIDHandler(b *testing.B) {
	mockSvc := NewMockService(b)
	mockSvc.On("GetOriginalURL", mock.Anything, "abc123").Return("https://example.com", nil)

	handler := NewHandler(mockSvc, 10)
	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	req = req.WithContext(requests.SetUserIDToContext(req.Context(), "user123"))
	req = req.WithContext(context.WithValue(context.WithValue(req.Context(), chi.RouteCtxKey, chi.NewRouteContext()), audit.AuditKey, &models.AuditData{}))
	chi.RouteContext(req.Context()).URLParams.Add("id", "abc123")

	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.IDHandler(w, req)
		w.Flush()
	}
}

// test adding url to storage
func TestPostURLHandler(t *testing.T) {
	fixedID := "test1234"

	type want struct {
		code        int
		contentType string
		success     bool
		body        string
	}
	type serviceData struct {
		value string
		err   error
	}
	tests := []struct {
		name    string
		method  string
		service *serviceData
		request string
		audit   bool
		want    want
	}{
		{
			name:   "positive test",
			method: http.MethodPost,
			service: &serviceData{
				value: fmt.Sprintf("http://example/%s", fixedID),
				err:   nil,
			},
			request: "https://www.example0.com/",
			audit:   true,
			want: want{
				code:        http.StatusCreated,
				contentType: "text/plain",
				success:     true,
				body:        fixedID,
			},
		},
		{
			name:    "wrong method test",
			method:  http.MethodGet,
			request: "https://www.example2.com/",
			audit:   true,
			want: want{
				code:        http.StatusMethodNotAllowed,
				contentType: "",
				success:     false,
				body:        "",
			},
		},
		{
			name:    "empty body test",
			method:  http.MethodPost,
			request: "",
			audit:   true,
			want: want{
				code:        http.StatusBadRequest,
				contentType: "",
				success:     false,
				body:        "",
			},
		},
		{
			name:    "no audit",
			method:  http.MethodPost,
			request: "https://www.example0.com/",
			audit:   false,
			want: want{
				code:        http.StatusInternalServerError,
				contentType: "text/plain; charset=utf-8",
				success:     false,
				body:        "",
			},
		},
		{
			name:   "empty url",
			method: http.MethodPost,
			service: &serviceData{
				value: "",
				err:   service.ErrEmptyURL,
			},
			request: "https://www.example0.com/",
			audit:   true,
			want: want{
				code:        http.StatusBadRequest,
				contentType: "text/plain",
				success:     false,
				body:        "",
			},
		},
		{
			name:   "duplicate",
			method: http.MethodPost,
			service: &serviceData{
				value: "",
				err:   service.ErrDuplicate,
			},
			request: "https://www.example0.com/",
			audit:   true,
			want: want{
				code:        http.StatusConflict,
				contentType: "text/plain",
				success:     false,
				body:        "",
			},
		},
		{
			name:   "unknown error",
			method: http.MethodPost,
			service: &serviceData{
				value: "",
				err:   fmt.Errorf("unknown error"),
			},
			request: "https://www.example0.com/",
			audit:   true,
			want: want{
				code:        http.StatusInternalServerError,
				contentType: "text/plain",
				success:     false,
				body:        "",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockService := NewMockService(t)
			if test.service != nil {
				mockService.EXPECT().CreateShortURL(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(test.service.value, test.service.err)
			} else {
				mockService.AssertNotCalled(t, "CreateShortURL", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			}
			ts := httptest.NewServer(newTestRouter(mockService, test.audit, false))
			defer ts.Close()
			resp, body := testRequest(t, ts, test.method, "/", test.request)
			defer resp.Body.Close()
			assert.Equal(t, test.want.code, resp.StatusCode)
			if !test.want.success {
				return
			}
			assert.Equal(t, test.want.contentType, resp.Header.Get("Content-Type"))
			parsedURL, err := url.Parse(body)
			require.NoError(t, err)
			createdID := strings.TrimLeft(parsedURL.Path, "/")
			assert.Equal(t, fixedID, createdID)
		})
	}
}

func BenchmarkPostURLHandler(b *testing.B) {
	mockSvc := NewMockService(b)
	mockSvc.EXPECT().CreateShortURL(mock.Anything, "https://example.com", mock.Anything, mock.Anything).
		Return("http://localhost/abc123", nil)

	handler := NewHandler(mockSvc, 10)
	body := []byte("https://example.com")
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(requests.SetUserIDToContext(req.Context(), "user123"), audit.AuditKey, &models.AuditData{}))
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req.Body = io.NopCloser(bytes.NewReader(body))
		handler.PostURLHandler(w, req)
		w.Flush()
	}
}

func TestPingHandler(t *testing.T) {
	mockService := NewMockService(t)
	mockService.EXPECT().Ping(mock.Anything).Return(nil)
	handler := NewHandler(mockService, 100)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	handler.PingHandler(w, req)

	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)

	type want struct {
		code int
	}
	type serviceData struct {
		err error
	}
	tests := []struct {
		name    string
		method  string
		service *serviceData
		want    want
	}{
		{
			name:   "positive test",
			method: http.MethodGet,
			service: &serviceData{
				err: nil,
			},
			want: want{
				code: http.StatusOK,
			},
		},
		{
			name:   "wrong method test",
			method: http.MethodPost,
			want: want{
				code: http.StatusMethodNotAllowed,
			},
		},
		{
			name:   "unknown error",
			method: http.MethodGet,
			service: &serviceData{
				err: fmt.Errorf("unknown error"),
			},
			want: want{
				code: http.StatusInternalServerError,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockService := NewMockService(t)
			if test.service != nil {
				mockService.EXPECT().Ping(mock.Anything).Return(test.service.err)
			} else {
				mockService.AssertNotCalled(t, "CreateShortURL", mock.Anything)
			}
			ts := httptest.NewServer(newTestRouter(mockService, false, false))
			defer ts.Close()
			resp, _ := testRequest(t, ts, test.method, "/ping", "")
			defer resp.Body.Close()
			assert.Equal(t, test.want.code, resp.StatusCode)
		})
	}
}

func BenchmarkPingHandler(b *testing.B) {
	mockSvc := NewMockService(b)
	mockSvc.On("Ping", mock.Anything).Return(nil)

	handler := NewHandler(mockSvc, 10)
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req = req.WithContext(requests.SetUserIDToContext(req.Context(), "user123"))
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler.PingHandler(w, req)
		w.Flush()
	}
}
