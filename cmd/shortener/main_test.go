package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIdHandler(t *testing.T) {
	idSize := 8
	testStorage := newStorage(idSize)
	existingUrl := "https://www.example.com/"
	existingId := testStorage.add(existingUrl)
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
            name: "positive test",
			method: http.MethodGet,
			id: existingId,
            want: want{
                code:        http.StatusTemporaryRedirect,
                response:    "",
                contentType: "text/plain",
				location:    existingUrl,
            },
        },
		{
			name: "wrong method test",
			method: http.MethodPost,
			id: existingId,
			want: want{
				code:        http.StatusBadRequest,
				response:    "",
				contentType: "",
				location: "",
			},
		},
		{
			name: "missing id test",
			method: http.MethodGet,
			id: "missingId",
			want: want{
				code:        http.StatusBadRequest,
				response:    "",
				contentType: "text/plain",
				location: "",
			},
		},
    }
    for _, test := range tests {
        t.Run(test.name, func(t *testing.T) {
            request := httptest.NewRequest(test.method, fmt.Sprintf("/%v", test.id), nil)
			request.SetPathValue("id", test.id)

            w := httptest.NewRecorder()
            idHandler(w, request, testStorage)
            res := w.Result()

            assert.Equal(t, test.want.code, res.StatusCode)            

            defer res.Body.Close()
            resBody, err := io.ReadAll(res.Body)

            require.NoError(t, err)
            assert.Equal(t, test.want.response, string(resBody))
            assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
			assert.Equal(t, test.want.location, res.Header.Get("Location"))
        })
    }
}


func TestPostUrlHandler(t *testing.T) {
	idSize := 8
	testStorage := newStorage(idSize)
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
            name: "positive test",
			method: http.MethodPost,
			request: "https://www.example0.com/",
            want: want{
                code:        http.StatusCreated,
                contentType: "text/plain",
				success: 	 true,
            },
        },
		{
            name: "extra spaces test",
			method: http.MethodPost,
			request: "   https://www.example1.com/   ",
            want: want{
                code:        http.StatusCreated,
                contentType: "text/plain",
				success: 	 true,
            },
        },
		{
			name: "wrong method test",
			method: http.MethodGet,
			request: "https://www.example2.com/",
			want: want{
				code:        http.StatusBadRequest,
				contentType: "",
				success: 	 false,
			},
		},
		{
			name: "missing url test",
			method: http.MethodPost,
			request: "    ",
			want: want{
				code:        http.StatusBadRequest,
				contentType: "",
				success: 	 false,
			},
		},
    }
    for _, test := range tests {
        t.Run(test.name, func(t *testing.T) {
            request := httptest.NewRequest(test.method, "/", bytes.NewBuffer([]byte(test.request)))

            w := httptest.NewRecorder()
            postUrlHandler(w, request, testStorage)
            res := w.Result()

            assert.Equal(t, test.want.code, res.StatusCode)
			if !test.want.success {
				return
			}
            defer res.Body.Close()
            resBody, err := io.ReadAll(res.Body)

            require.NoError(t, err)
			assert.Equal(t, test.want.contentType, res.Header.Get("Content-Type"))
			parsedUrl, err := url.Parse(string(resBody))
			require.NoError(t, err)
			createdId := strings.TrimLeft(parsedUrl.Path, "/")
			assert.Equal(t, len(createdId), idSize)
			savedUrl, found := testStorage.get(createdId)
			require.True(t, found)
            assert.Equal(t, savedUrl, strings.TrimSpace(test.request))
        })
    }
}