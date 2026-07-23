package main

import (
	"io"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeRequest(t *testing.T) {
	data := url.Values{}
	data.Set("url", "https://example.com")
	req, err := makeRequest("http://localhost:8080/", data)
	assert.NoError(t, err)
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "application/x-www-form-urlencoded", req.Header.Get("Content-Type"))
	body, _ := io.ReadAll(req.Body)
	assert.Contains(t, string(body), "url=https%3A%2F%2Fexample.com")
}
