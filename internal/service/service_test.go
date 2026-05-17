package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockStorage реализует storage.StorageInterface для тестов
type mockStorage struct {
	data    map[string]string
	fixedID string
}

func (m *mockStorage) Add(ctx context.Context, url string) (string, error) {
	m.data[m.fixedID] = url
	return m.fixedID, nil
}

func (m *mockStorage) Get(ctx context.Context, id string) (string, error) {
	val, ok := m.data[id]
	if !ok {
		return "", fmt.Errorf("Not found")
	}
	return val, nil
}

func (m *mockStorage) Ping(ctx context.Context) error {
	return nil
}

func TestEndpointService_CreateShortURL(t *testing.T) {
	fixedId := "test1234"
	testStorage := &mockStorage{
		data:    make(map[string]string),
		fixedID: fixedId,
	}
	type want struct {
		short string
		err   error
	}
	tests := []struct {
		name        string
		showAddr    string
		scheme      string
		host        string
		originalURL string
		addError    error
		want        want
	}{
		{
			name:        "success with showAddr",
			showAddr:    "https://short.com",
			scheme:      "http",
			host:        "localhost:8080",
			originalURL: "https://example.com",
			addError:    nil,
			want: want{
				short: fmt.Sprintf("https://short.com/%s", fixedId),
				err:   nil,
			},
		},
		{
			name:        "success without showAddr (use scheme+host)",
			showAddr:    "",
			scheme:      "https",
			host:        "example.com",
			originalURL: "https://example.com/long",
			addError:    nil,
			want: want{
				short: fmt.Sprintf("https://example.com/%s", fixedId),
				err:   nil,
			},
		},
		{
			name:        "default scheme http when empty",
			showAddr:    "",
			scheme:      "",
			host:        "mysite.com",
			originalURL: "https://example.com",
			addError:    nil,
			want: want{
				short: fmt.Sprintf("http://mysite.com/%s", fixedId),
				err:   nil,
			},
		},
		{
			name:        "empty url after trim",
			showAddr:    "",
			scheme:      "http",
			host:        "localhost",
			originalURL: "   ",
			addError:    nil,
			want: want{
				short: "",
				err:   ErrorEmptyUrl,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testService := NewEndpointService(testStorage, tt.showAddr)
			got, err := testService.CreateShortURL(context.Background(), tt.originalURL, tt.scheme, tt.host)
			if tt.want.err != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.want.err.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.short, got)
			}
		})
	}
}

func TestEndpointService_GetOriginalURL(t *testing.T) {
	fixedId := "test1234"
	testStorage := &mockStorage{
		data:    make(map[string]string),
		fixedID: fixedId,
	}
	existingUrl := "https://existing-url.com"
	testStorage.data[fixedId] = existingUrl
	testService := NewEndpointService(testStorage, "")
	type want struct {
		url string
		err error
	}
	tests := []struct {
		name    string
		shortID string
		want    want
	}{
		{
			name:    "success",
			shortID: fixedId,
			want: want{
				url: existingUrl,
				err: nil,
			},
		},
		{
			name:    "storage error",
			shortID: "missing",
			want: want{
				url: "",
				err: errors.New("Not found"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := testService.GetOriginalURL(context.Background(), tt.shortID)
			if tt.want.err != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.want.err.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.url, got)
			}
		})
	}
}
