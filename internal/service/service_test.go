package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/max-marek-projects/shortener/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStorage implements storage.StorageInterface for tests
type mockStorage struct {
	data map[string]string
}

func (m *mockStorage) Add(ctx context.Context, id, url string) error {
	m.data[id] = url
	return nil
}

func (m *mockStorage) Get(ctx context.Context, id string) (string, error) {
	val, ok := m.data[id]
	if !ok {
		return "", fmt.Errorf("Not found")
	}
	return val, nil
}

func (m *mockStorage) AddBatch(ctx context.Context, items []repository.Row) error {
	for _, item := range items {
		m.data[item.ID] = item.OriginalURL
	}
	return nil
}

func (m *mockStorage) Ping(ctx context.Context) error {
	return nil
}

func TestEndpointService_CreateShortURL(t *testing.T) {
	idSize := 8
	testStorage := &mockStorage{
		data: make(map[string]string),
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
				short: "https://short.com/",
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
				short: "https://example.com/",
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
				short: "http://mysite.com/",
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
			testService := NewEndpointService(testStorage, tt.showAddr, idSize)
			got, err := testService.CreateShortURL(context.Background(), tt.originalURL, tt.scheme, tt.host)
			if tt.want.err != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.want.err.Error(), err.Error())
			} else {
				assert.NoError(t, err)
				assert.True(t, strings.HasPrefix(got, tt.want.short))
				parsedUrl, err := url.Parse(got)
				require.NoError(t, err)
				createdId := strings.TrimLeft(parsedUrl.Path, "/")
				assert.Equal(t, len(createdId), idSize)
			}
		})
	}
}

func TestEndpointService_GetOriginalURL(t *testing.T) {
	fixedId := "test1234"
	idSize := 8
	testStorage := &mockStorage{
		data: make(map[string]string),
	}
	existingUrl := "https://existing-url.com"
	testStorage.data[fixedId] = existingUrl
	testService := NewEndpointService(testStorage, "", idSize)
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
