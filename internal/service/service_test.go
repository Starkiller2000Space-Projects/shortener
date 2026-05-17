package service

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/max-marek-projects/shortener/internal/repository"
	"github.com/max-marek-projects/shortener/internal/service/mocks"

	"github.com/max-marek-projects/shortener/internal/utils"
)

func ctxWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, utils.UserIDKey, userID)
}

func TestService_CreateShortURL(t *testing.T) {
	idSize := 8
	mockStorage := mocks.NewStorage(t)
	mockStorage.EXPECT().Add(mock.Anything, mock.Anything).Return(nil)
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
			testService := NewEndpointService(mockStorage, tt.showAddr, idSize)
			got, err := testService.CreateShortURL(ctxWithUserID(context.Background(), "test-user-id"), tt.originalURL, tt.scheme, tt.host)
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

func TestService_GetOriginalURL(t *testing.T) {
	fixedId := "test1234"
	idSize := 8
	existingUrl := "https://existing-url.com"
	// mock service
	mockStorage := mocks.NewStorage(t)
	mockStorage.EXPECT().Get(mock.Anything, fixedId).Return(existingUrl, nil)
	mockStorage.EXPECT().Get(mock.Anything, mock.Anything).Return("", repository.ErrNotFound)
	testService := NewEndpointService(mockStorage, "", idSize)
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
				err: repository.ErrNotFound,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := testService.GetOriginalURL(ctxWithUserID(context.Background(), "test-user-id"), tt.shortID)
			if tt.want.err != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.want.err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want.url, got)
			}
		})
	}
}
