package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/max-marek-projects/shortener/internal/models"
	"github.com/max-marek-projects/shortener/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// test adding url to storage
func TestShortenJSONHandler(t *testing.T) {
	fixedID := "test1234"
	type want struct {
		code        int
		contentType string
		success     bool
	}
	type serviceData struct {
		value string
		err   error
	}
	tests := []struct {
		name    string
		method  string
		request string
		service *serviceData
		audit   bool
		want    want
	}{
		{
			name:    "positive test",
			method:  http.MethodPost,
			request: `{"url":"https://www.example0.com/"}`,
			service: &serviceData{
				value: fmt.Sprintf("http://example/%s", fixedID),
				err:   nil,
			},
			audit: true,
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
			service: nil,
			audit:   true,
			want: want{
				code:    http.StatusMethodNotAllowed,
				success: false,
			},
		},
		{
			name:    "empty url",
			method:  http.MethodPost,
			request: `{"url":""}`,
			service: &serviceData{
				value: "",
				err:   service.ErrorEmptyUrl,
			},
			audit: true,
			want: want{
				code:    http.StatusBadRequest,
				success: false,
			},
		},
		{
			name:    "duplicate",
			method:  http.MethodPost,
			request: `{"url":"https://www.example2.com/"}`,
			service: &serviceData{
				value: "",
				err:   service.ErrorDuplicate,
			},
			audit: true,
			want: want{
				code:    http.StatusConflict,
				success: false,
			},
		},
		{
			name:    "unknown error",
			method:  http.MethodPost,
			request: `{"url":"https://www.example2.com/"}`,
			service: &serviceData{
				value: "",
				err:   fmt.Errorf("unknown error"),
			},
			audit: true,
			want: want{
				code:    http.StatusInternalServerError,
				success: false,
			},
		},
		{
			name:    "no audit",
			method:  http.MethodPost,
			request: `{"url":"https://www.example0.com/"}`,
			service: nil,
			audit:   false,
			want: want{
				code:    http.StatusInternalServerError,
				success: false,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// create service mock
			mockService := NewMockService(t)
			if test.service != nil {
				mockService.EXPECT().CreateShortURL(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(test.service.value, test.service.err)
			} else {
				mockService.AssertNotCalled(t, "CreateShortURL", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			}
			ts := httptest.NewServer(newTestRouter(mockService, test.audit, false))
			defer ts.Close()
			// make request
			resp, body := testRequest(t, ts, test.method, "/api/shorten", test.request)
			assert.Equal(t, test.want.code, resp.StatusCode)
			if !test.want.success {
				return
			}
			// validate response
			assert.Equal(t, test.want.contentType, resp.Header.Get("Content-Type"))
			var responseData models.ShortenResponse
			require.NoError(t, json.Unmarshal([]byte(body), &responseData))
			parsedUrl, err := url.Parse(responseData.Result)
			require.NoError(t, err)
			createdId := strings.TrimLeft(parsedUrl.Path, "/")
			assert.Equal(t, fixedID, createdId)
			var requestData models.ShortenRequest
			require.NoError(t, json.Unmarshal([]byte(test.request), &requestData))
		})
	}
}

// test adding data as batch
func TestPostBatchShortenHandler(t *testing.T) {
	fixedID := "test1234"

	type want struct {
		code        int
		contentType string
		success     bool
	}
	type serviceData struct {
		value []models.BatchShortenResponse
		err   error
	}
	tests := []struct {
		name    string
		method  string
		request string
		service *serviceData
		want    want
	}{
		{
			name:    "positive test",
			method:  http.MethodPost,
			request: `[{"correlation_id": "1", "original_url": "https://www.example0.com/"},{"correlation_id": "2", "original_url": "https://www.example1.com/"}]`,
			service: &serviceData{
				value: []models.BatchShortenResponse{{CorrelationID: "1", ShortURL: fmt.Sprintf("http://example.com/%s", fixedID)}},
				err:   nil,
			},
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
			service: nil,
			want: want{
				code:    http.StatusMethodNotAllowed,
				success: false,
			},
		},
		{
			name:    "invalid json",
			method:  http.MethodPost,
			request: `invalid json`,
			service: nil,
			want: want{
				code:    http.StatusBadRequest,
				success: false,
			},
		},
		{
			name:    "empty json",
			method:  http.MethodPost,
			request: `[]`,
			service: nil,
			want: want{
				code:    http.StatusBadRequest,
				success: false,
			},
		},
		{
			name:    "empty url",
			method:  http.MethodPost,
			request: `[{"correlation_id": "1", "original_url": ""},{"correlation_id": "2", "original_url": "https://www.example1.com/"}]`,
			service: &serviceData{
				value: nil,
				err:   service.ErrorEmptyUrl,
			},
			want: want{
				code:    http.StatusBadRequest,
				success: false,
			},
		},
		{
			name:    "unknown error",
			method:  http.MethodPost,
			request: `[{"correlation_id": "1", "original_url": "https://www.example0.com/"},{"correlation_id": "2", "original_url": "https://www.example1.com/"}]`,
			service: &serviceData{
				value: nil,
				err:   fmt.Errorf("Unknown error"),
			},
			want: want{
				code:    http.StatusInternalServerError,
				success: false,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockService := NewMockService(t)
			if test.service != nil {
				mockService.EXPECT().CreateShortURLsBatch(
					mock.Anything, mock.Anything, mock.Anything, mock.Anything,
				).Return(test.service.value, test.service.err)
			} else {
				mockService.AssertNotCalled(t, "CreateShortURLsBatch", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			}
			ts := httptest.NewServer(newTestRouter(mockService, false, false))
			defer ts.Close()
			resp, body := testRequest(t, ts, test.method, "/api/shorten/batch", test.request)
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

func TestGetUserURLsHandler(t *testing.T) {
	type want struct {
		code        int
		contentType string
		success     bool
	}
	type serviceData struct {
		value []models.UserURL
		err   error
	}
	tests := []struct {
		name    string
		method  string
		userID  string
		service *serviceData
		want    want
	}{
		{
			name:   "positive test",
			method: http.MethodGet,
			userID: `123user`,
			service: &serviceData{
				value: []models.UserURL{
					{ShortURL: "https://example.com/12345", OriginalURL: "https://example.com/very/long/"},
				},
				err: nil,
			},
			want: want{
				code:        http.StatusOK,
				contentType: "application/json",
				success:     true,
			},
		},
		{
			name:   "wrong method",
			method: http.MethodPost,
			userID: `123user`,
			want: want{
				code:    http.StatusMethodNotAllowed,
				success: false,
			},
		},
		{
			name:   "no content",
			method: http.MethodGet,
			userID: `123user`,
			service: &serviceData{
				value: []models.UserURL{},
				err:   nil,
			},
			want: want{
				code:    http.StatusNoContent,
				success: false,
			},
		},
		{
			name:   "unknown error",
			method: http.MethodGet,
			userID: `123user`,
			service: &serviceData{
				value: nil,
				err:   fmt.Errorf("unknown error"),
			},
			want: want{
				code:    http.StatusInternalServerError,
				success: false,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockService := NewMockService(t)
			if test.service != nil {
				mockService.EXPECT().GetUserURLs(mock.Anything, mock.Anything, mock.Anything).Return(test.service.value, test.service.err)
			} else {
				mockService.AssertNotCalled(t, "GetUserURLs", mock.Anything, mock.Anything, mock.Anything)
			}
			ts := httptest.NewServer(newTestRouter(mockService, false, false))
			defer ts.Close()
			resp, body := testRequest(t, ts, test.method, "/api/user/urls", "")
			assert.Equal(t, test.want.code, resp.StatusCode)
			if !test.want.success {
				return
			}
			assert.Equal(t, test.want.contentType, resp.Header.Get("Content-Type"))
			var responseData []models.UserURL
			require.NoError(t, json.Unmarshal([]byte(body), &responseData))
			for _, responseItem := range responseData {
				_, err := url.Parse(responseItem.ShortURL)
				require.NoError(t, err)
			}
		})
	}
}

func TestDeleteUserURLsHandler(t *testing.T) {
	testUserID := "123user"

	type want struct {
		code int
	}
	type serviceData struct {
		value []string
		err   error
	}
	tests := []struct {
		name    string
		method  string
		request string
		service *serviceData
		auth    bool
		want    want
	}{
		{
			name:    "positive test",
			method:  http.MethodDelete,
			request: `["abc123", "abc234", "abc345"]`,
			service: &serviceData{
				value: []string{"abc123", "abc234", "abc345"},
				err:   nil,
			},
			auth: true,
			want: want{
				code: http.StatusAccepted,
			},
		},
		{
			name:    "wrong method",
			method:  http.MethodPost,
			request: `["abc123", "abc234", "abc345"]`,
			auth:    true,
			want: want{
				code: http.StatusMethodNotAllowed,
			},
		},
		{
			name:    "empty request",
			method:  http.MethodDelete,
			request: `[]`,
			auth:    true,
			want: want{
				code: http.StatusAccepted,
			},
		},
		{
			name:    "invalid json",
			method:  http.MethodDelete,
			request: `invalid json`,
			auth:    true,
			want: want{
				code: http.StatusBadRequest,
			},
		},
		{
			name:    "no user id",
			method:  http.MethodDelete,
			request: `[]`,
			auth:    false,
			want: want{
				code: http.StatusInternalServerError,
			},
		},
		{
			name:    "background error",
			method:  http.MethodDelete,
			request: `["abc123", "abc234", "abc345"]`,
			service: &serviceData{
				value: []string{"abc123", "abc234", "abc345"},
				err:   fmt.Errorf("unknown error"),
			},
			auth: true,
			want: want{
				code: http.StatusAccepted,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockService := NewMockService(t)
			if test.service != nil {
				mockService.EXPECT().DeleteUserURLs(mock.Anything, testUserID, test.service.value).Return(test.service.err)
			} else {
				mockService.AssertNotCalled(t, "DeleteUserURLs", mock.Anything, mock.Anything, mock.Anything)
			}
			ts := httptest.NewServer(newTestRouter(mockService, false, test.auth))
			defer ts.Close()
			resp, _ := testRequest(t, ts, test.method, "/api/user/urls", test.request)
			assert.Equal(t, test.want.code, resp.StatusCode)
		})
	}
}
