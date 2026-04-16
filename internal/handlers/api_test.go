package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// test adding url to storage
func TestShortenJSONHandler(t *testing.T) {
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
			request: `{"url":"https://www.example0.com/"}`,
			want: want{
				code:        http.StatusCreated,
				contentType: "text/plain",
				success:     true,
			},
		},
		{
			name:    "extra spaces test",
			method:  http.MethodPost,
			request: `{"url":"   https://www.example1.com/   "}`,
			want: want{
				code:        http.StatusCreated,
				contentType: "text/plain",
				success:     true,
			},
		},
		{
			name:    "wrong method test",
			method:  http.MethodGet,
			request: `{"url":"https://www.example2.com/"}`,
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
