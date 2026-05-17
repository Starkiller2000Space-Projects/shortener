package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/max-marek-projects/shortener/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/max-marek-projects/shortener/internal/handlers/mocks"
)

// create new router for handlers testing
func newAPITestRouter(service *mocks.Service) http.Handler {
	h := NewHandler(service)

	r := chi.NewRouter()
	r.Post("/shorten", h.ShortenJSONHandler)
	r.Post("/shorten/batch", h.PostBatchShortenHandler)

	return r
}

// test adding url to storage
func TestShortenJSONHandler(t *testing.T) {
	fixedID := "test1234"
	mockService := mocks.NewService(t)
	mockService.EXPECT().CreateShortURL(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fmt.Sprintf("http://example/%s", fixedID), nil)

	ts := httptest.NewServer(newAPITestRouter(mockService))
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
			request: `{"url":"https://www.example0.com/"}`,
			want: want{
				code:        http.StatusCreated,
				contentType: "application/json",
				success:     true,
			},
		},
		{
			name:    "wrong method test",
			method:  http.MethodGet,
			request: `{"url":"https://www.example2.com/"}`,
			want: want{
				code:        http.StatusMethodNotAllowed,
				contentType: "",
				success:     false,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp, body := testRequest(t, ts, test.method, "/shorten", test.request)
			assert.Equal(t, test.want.code, resp.StatusCode)
			if !test.want.success {
				return
			}
			assert.Equal(t, test.want.contentType, resp.Header.Get("Content-Type"))
			var responseData models.ShortenResponse
			require.NoError(t, json.Unmarshal([]byte(body), &responseData))
			parsedUrl, err := url.Parse(responseData.Result)
			require.NoError(t, err)
			createdId := strings.TrimLeft(parsedUrl.Path, "/")
			assert.Equal(t, createdId, fixedID)
			var requestData models.ShortenRequest
			require.NoError(t, json.Unmarshal([]byte(test.request), &requestData))
		})
	}
}

// test adding data as batch
func TestPostBatchShortenHandler(t *testing.T) {
	fixedID := "test1234"
	mockService := mocks.NewService(t)
	mockService.EXPECT().CreateShortURLsBatch(
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return(
		[]models.BatchShortenResponse{{CorrelationID: "1", ShortURL: fmt.Sprintf("http://example.com/%s", fixedID)}},
		nil,
	)

	ts := httptest.NewServer(newAPITestRouter(mockService))
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
			request: `[{"correlation_id": "1", "original_url": "https://www.example0.com/"},{"correlation_id": "2", "original_url": "https://www.example1.com/"}]`,
			want: want{
				code:        http.StatusCreated,
				contentType: "application/json",
				success:     true,
			},
		},
		{
			name:    "wrong method test",
			method:  http.MethodGet,
			request: `{"url":"https://www.example2.com/"}`,
			want: want{
				code:        http.StatusMethodNotAllowed,
				contentType: "",
				success:     false,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resp, body := testRequest(t, ts, test.method, "/shorten/batch", test.request)
			assert.Equal(t, test.want.code, resp.StatusCode)
			if !test.want.success {
				return
			}
			assert.Equal(t, test.want.contentType, resp.Header.Get("Content-Type"))
			var responseData []models.BatchShortenResponse
			require.NoError(t, json.Unmarshal([]byte(body), &responseData))
			var requestData []models.BatchShortenRequest
			require.NoError(t, json.Unmarshal([]byte(test.request), &requestData))
			for _, responseItem := range responseData {
				_, err := url.Parse(responseItem.ShortURL)
				require.NoError(t, err)
			}
		})
	}
}
