package handlers

import (
	"bytes"
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

	"github.com/max-marek-projects/shortener/internal/handlers/mocks"
	"github.com/max-marek-projects/shortener/internal/repository"
)

// create new router for handlers testing
func newTestRouter(service *mocks.Service) http.Handler {
	h := NewHandler(service)

	r := chi.NewRouter()
	r.Get("/ping", h.PingHandler)
	r.Get("/{id}", h.IdHandler)
	r.Post("/", h.PostUrlHandler)

	return r
}

// test single request
func testRequest(t *testing.T, ts *httptest.Server, method, path, body string) (*http.Response, string) {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = bytes.NewBufferString(body)
	}

	req, err := http.NewRequest(method, ts.URL+path, reader)
	require.NoError(t, err)

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
func TestIdHandler(t *testing.T) {
	fixedID := "test1234"
	existingUrl := "https://example.com/"
	mockService := mocks.NewService(t)
	mockService.EXPECT().GetOriginalURL(mock.Anything, fixedID).Return(existingUrl, nil)
	mockService.EXPECT().GetOriginalURL(mock.Anything, mock.Anything).Return("", repository.ErrNotFound)

	ts := httptest.NewServer(newTestRouter(mockService))
	defer ts.Close()

	type want struct {
		code        int
		response    string
		contentType string
		location    string
	}
	tests := []struct {
		name   string
		method string
		id     string
		want   want
	}{
		{
			name:   "positive test",
			method: http.MethodGet,
			id:     fixedID,
			want: want{
				code:        http.StatusTemporaryRedirect,
				response:    "",
				contentType: "text/plain",
				location:    existingUrl,
			},
		},
		{
			name:   "wrong method test",
			method: http.MethodPost,
			id:     fixedID,
			want: want{
				code:        http.StatusMethodNotAllowed,
				response:    "",
				contentType: "",
				location:    "",
			},
		},
		{
			name:   "missing id test",
			method: http.MethodGet,
			id:     "missingId",
			want: want{
				code:        http.StatusBadRequest,
				response:    "",
				contentType: "text/plain",
				location:    "",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp, body := testRequest(t, ts, test.method, fmt.Sprintf("/%v", test.id), "")
			assert.Equal(t, test.want.code, resp.StatusCode)
			assert.Equal(t, test.want.response, body)
			assert.Equal(t, test.want.contentType, resp.Header.Get("Content-Type"))
			assert.Equal(t, test.want.location, resp.Header.Get("Location"))
		})
	}
}

// test adding url to storage
func TestPostUrlHandler(t *testing.T) {
	fixedID := "test1234"
	mockService := mocks.NewService(t)
	mockService.EXPECT().CreateShortURL(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Sprintf("http://example/%s", fixedID), nil)

	ts := httptest.NewServer(newTestRouter(mockService))
	defer ts.Close()

	type want struct {
		code        int
		contentType string
		success     bool
		body        string
	}
	tests := []struct {
		name    string
		method  string
		request string
		want    want
	}{
		{
			name:    "positive test",
			method:  http.MethodPost,
			request: "https://www.example0.com/",
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
			want: want{
				code:        http.StatusMethodNotAllowed,
				contentType: "",
				success:     false,
				body:        "",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp, body := testRequest(t, ts, test.method, "/", test.request)
			assert.Equal(t, test.want.code, resp.StatusCode)
			if !test.want.success {
				return
			}
			assert.Equal(t, test.want.contentType, resp.Header.Get("Content-Type"))
			parsedUrl, err := url.Parse(body)
			require.NoError(t, err)
			createdId := strings.TrimLeft(parsedUrl.Path, "/")
			assert.Equal(t, createdId, fixedID)
		})
	}
}

func TestPingHandler(t *testing.T) {
	mockService := mocks.NewService(t)
	mockService.EXPECT().Ping(mock.Anything).Return(nil)
	handler := NewHandler(mockService)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	handler.PingHandler(w, req)

	res := w.Result()
	defer res.Body.Close()
	assert.Equal(t, http.StatusOK, res.StatusCode)
}
