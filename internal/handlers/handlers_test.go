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
	"github.com/stretchr/testify/require"
)

// mock with service interface
type mockService struct {
	data    map[string]string
	fixedID string
}

// add url into storage and return generated id
func (service *mockService) CreateShortURL(ctx context.Context, original, scheme, host string) (string, error) {
	service.data[service.fixedID] = original
	return service.fixedID, nil
}

// get url by id from storage if exists
func (service *mockService) GetOriginalURL(ctx context.Context, short string) (string, error) {
	val, ok := service.data[short]
	if !ok {
		return "", fmt.Errorf("Not found")
	}
	return val, nil
}

func (service *mockService) GetScheme(ctx context.Context, headers http.Header) (string, error) {
	scheme := "http"
	if headers.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme, nil
}

// create new router for handlers testing
func newTestRouter(service *mockService) http.Handler {
	h := NewHandler(service)

	r := chi.NewRouter()
	r.HandleFunc("/", h.PostUrlHandler)
	r.HandleFunc("/{id}", h.IdHandler)

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
	testStorage := &mockService{
		data:    make(map[string]string),
		fixedID: "test1234",
	}
	existingUrl := "https://example.com/"
	existingId, _ := testStorage.CreateShortURL(context.TODO(), existingUrl, "", "")

	ts := httptest.NewServer(newTestRouter(testStorage))
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
			id:     existingId,
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
			id:     existingId,
			want: want{
				code:        http.StatusBadRequest,
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
	fixedId := "test1234"
	testStorage := &mockService{
		data:    make(map[string]string),
		fixedID: fixedId,
	}

	ts := httptest.NewServer(newTestRouter(testStorage))
	defer ts.Close()

	type want struct {
		code        int
		contentType string
		success     bool
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
			},
		},
		{
			name:    "wrong method test",
			method:  http.MethodGet,
			request: "https://www.example2.com/",
			want: want{
				code:        http.StatusBadRequest,
				contentType: "",
				success:     false,
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
			assert.Equal(t, createdId, fixedId)
			savedUrl, err := testStorage.GetOriginalURL(context.TODO(), createdId)
			require.NoError(t, err)
			assert.Equal(t, savedUrl, strings.TrimSpace(test.request))
		})
	}
}
