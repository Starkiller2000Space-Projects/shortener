package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/max-marek-projects/shortener/internal/models"
	"github.com/max-marek-projects/shortener/internal/repository"
	"github.com/max-marek-projects/shortener/internal/requests"
	"github.com/max-marek-projects/shortener/internal/utils"
)

// test for short url creation
func TestService_CreateShortURL(t *testing.T) {
	idSize := 8

	type want struct {
		shortPrefix string
		err         error
	}

	tests := []struct {
		name        string
		showAddr    string
		scheme      string
		host        string
		originalURL string
		ctx         context.Context
		addError    error
		expectAdd   bool
		want        want
	}{
		{
			name:        "success with showAddr",
			showAddr:    "https://short.com",
			scheme:      "http",
			host:        "localhost:8080",
			originalURL: "https://example.com",
			ctx: requests.SetUserIDToContext(
				context.Background(),
				"test-user-id",
			),
			expectAdd: true,
			want: want{
				shortPrefix: "https://short.com/",
			},
		},
		{
			name:        "success without showAddr (use scheme+host)",
			showAddr:    "",
			scheme:      "https",
			host:        "example.com",
			originalURL: "https://example.com/long",
			ctx: requests.SetUserIDToContext(
				context.Background(),
				"test-user-id",
			),
			expectAdd: true,
			want: want{
				shortPrefix: "https://example.com/",
			},
		},
		{
			name:        "default scheme http when empty",
			showAddr:    "",
			scheme:      "",
			host:        "mysite.com",
			originalURL: "https://example.com",
			ctx: requests.SetUserIDToContext(
				context.Background(),
				"test-user-id",
			),
			expectAdd: true,
			want: want{
				shortPrefix: "http://mysite.com/",
			},
		},
		{
			name:        "empty url after trim",
			showAddr:    "",
			scheme:      "http",
			host:        "localhost",
			originalURL: "   ",
			ctx: requests.SetUserIDToContext(
				context.Background(),
				"test-user-id",
			),
			expectAdd: false,
			want: want{
				err: ErrEmptyURL,
			},
		},
		{
			name:        "missing userID in context",
			showAddr:    "",
			scheme:      "http",
			host:        "localhost",
			originalURL: "https://example.com",
			ctx:         context.Background(),
			expectAdd:   false,
			want: want{
				err: fmt.Errorf("userID not found in context"),
			},
		},
		{
			name:        "storage add error (non-duplicate)",
			showAddr:    "",
			scheme:      "http",
			host:        "localhost",
			originalURL: "https://example.com",
			ctx: requests.SetUserIDToContext(
				context.Background(),
				"test-user-id",
			),
			addError:  sql.ErrConnDone,
			expectAdd: true,
			want: want{
				err: fmt.Errorf(
					"failed to create short url: %w",
					sql.ErrConnDone,
				),
			},
		},
		{
			name:        "storage duplicate error (ErrAlreadyExists)",
			showAddr:    "http://short.me",
			scheme:      "https",
			host:        "example.com",
			originalURL: "https://example.com",
			ctx: requests.SetUserIDToContext(
				context.Background(),
				"test-user-id",
			),
			addError: &repository.ErrAlreadyExists{
				ExistingID: "abc123",
			},
			expectAdd: true,
			want: want{
				shortPrefix: "http://short.me/abc123",
				err:         ErrDuplicate,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := NewMockStorage(t)

			if tt.expectAdd {
				mockStorage.EXPECT().
					Add(mock.Anything, mock.Anything).
					Return(tt.addError)
			}

			svc := NewEndpointService(
				mockStorage,
				tt.showAddr,
				idSize,
			)

			got, err := svc.CreateShortURL(
				tt.ctx,
				tt.originalURL,
				tt.scheme,
				tt.host,
			)

			if tt.want.err != nil {
				require.Error(t, err)

				if errors.Is(tt.want.err, ErrEmptyURL) ||
					errors.Is(tt.want.err, ErrDuplicate) {
					assert.ErrorIs(t, err, tt.want.err)
				} else {
					assert.EqualError(t, err, tt.want.err.Error())
				}

				if errors.Is(tt.want.err, ErrDuplicate) {
					assert.NotEmpty(t, got)
					assert.True(
						t,
						strings.HasPrefix(got, tt.want.shortPrefix),
					)
				} else {
					assert.Empty(t, got)
				}

				return
			}

			require.NoError(t, err)
			assert.True(
				t,
				strings.HasPrefix(got, tt.want.shortPrefix),
			)

			parsed, parseErr := url.Parse(got)
			require.NoError(t, parseErr)

			id := strings.TrimLeft(parsed.Path, "/")
			assert.Equal(t, idSize, len(id))
		})
	}
}

func BenchmarkService_CreateShortURL(b *testing.B) {
	store, _ := repository.NewMemStorage()
	svc := NewEndpointService(store, "", 8)
	ctx := requests.SetUserIDToContext(context.Background(), "bench-user")

	for i := 0; b.Loop(); i++ {
		url := fmt.Sprintf("https://example.com/%d", i)
		_, _ = svc.CreateShortURL(ctx, url, "http", "localhost:8080")
	}
}

func TestService_GetOriginalURL(t *testing.T) {
	fixedID := "test1234"
	idSize := 8
	existingURL := "https://existing-url.com"
	// mock service
	mockStorage := NewMockStorage(t)
	mockStorage.EXPECT().Get(mock.Anything, fixedID).Return(existingURL, nil)
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
			shortID: fixedID,
			want: want{
				url: existingURL,
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
			got, err := testService.GetOriginalURL(requests.SetUserIDToContext(context.Background(), "test-user-id"), tt.shortID)
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

func BenchmarkService_GetOriginalURL(b *testing.B) {
	store, _ := repository.NewMemStorage()
	svc := NewEndpointService(store, "", 8)
	ctx := requests.SetUserIDToContext(context.Background(), "bench-user")

	const n = 1000
	ids := make([]string, n)
	for i := range n {
		id := utils.GenerateID(8)
		ids[i] = id
		_ = store.Add(ctx, repository.Row{ID: id, OriginalURL: fmt.Sprintf("https://example.com/%d", i), UserID: "bench-user"})
	}

	for i := 0; b.Loop(); i++ {
		id := ids[i%n]
		_, _ = svc.GetOriginalURL(ctx, id)
	}
}

func TestService_CreateShortURLsBatch(t *testing.T) {
	idSize := 8
	mockStorage := NewMockStorage(t)
	mockStorage.EXPECT().AddBatch(mock.Anything, mock.Anything).Return(nil)

	testService := NewEndpointService(mockStorage, "http://short.com", idSize)
	req := []models.BatchShortenRequest{
		{CorrelationID: "1", OriginalURL: "https://example1.com"},
		{CorrelationID: "2", OriginalURL: "https://example2.com"},
	}
	ctx := requests.SetUserIDToContext(context.Background(), "user1")
	resp, err := testService.CreateShortURLsBatch(ctx, req, "https", "short.com")
	require.NoError(t, err)
	assert.Len(t, resp, 2)
	for _, r := range resp {
		assert.Contains(t, r.ShortURL, "http://short.com/")
		assert.NotEmpty(t, r.CorrelationID)
	}
}

func BenchmarkService_CreateShortURLsBatch(b *testing.B) {
	store, _ := repository.NewMemStorage()
	svc := NewEndpointService(store, "http://short.com", 8)
	ctx := requests.SetUserIDToContext(context.Background(), "bench-user")
	const batchSize = 10

	for i := 0; b.Loop(); i++ {
		req := make([]models.BatchShortenRequest, batchSize)
		for j := range batchSize {
			req[j] = models.BatchShortenRequest{
				CorrelationID: fmt.Sprintf("%d-%d", i, j),
				OriginalURL:   fmt.Sprintf("https://example.com/%d/%d", i, j),
			}
		}
		_, _ = svc.CreateShortURLsBatch(ctx, req, "http", "short.com")
	}
}

func TestService_GetUserURLs(t *testing.T) {
	mockStorage := NewMockStorage(t)
	userURLs := []repository.UserURL{
		{ShortURL: "abc123", OriginalURL: "http://orig.com"},
	}
	mockStorage.EXPECT().GetUserURLs(mock.Anything, "user1").Return(userURLs, nil)

	testService := NewEndpointService(mockStorage, "http://short.com", 8)
	ctx := requests.SetUserIDToContext(context.Background(), "user1")
	result, err := testService.GetUserURLs(ctx, "http", "short.com")
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "http://short.com/abc123", result[0].ShortURL)
	assert.Equal(t, "http://orig.com", result[0].OriginalURL)
}

func BenchmarkService_GetUserURLs(b *testing.B) {
	store, _ := repository.NewMemStorage()
	svc := NewEndpointService(store, "http://short.com", 8)
	ctx := requests.SetUserIDToContext(context.Background(), "bench-user")

	const n = 100
	for i := range n {
		id := utils.GenerateID(8)
		_ = store.Add(ctx, repository.Row{ID: id, OriginalURL: fmt.Sprintf("https://example.com/%d", i), UserID: "bench-user"})
	}

	for b.Loop() {
		_, _ = svc.GetUserURLs(ctx, "http", "short.com")
	}
}

func TestService_DeleteUserURLs(t *testing.T) {
	mockStorage := NewMockStorage(t)
	mockStorage.EXPECT().DeleteBatch(mock.Anything, "user1", []string{"abc", "def"}).Return(nil)

	testService := NewEndpointService(mockStorage, "", 8)
	ctx := context.Background()
	err := testService.DeleteUserURLs(ctx, "user1", []string{"abc", "def"})
	assert.NoError(t, err)

	// empty slice
	err = testService.DeleteUserURLs(ctx, "user1", []string{})
	assert.NoError(t, err)
}

func TestService_GetStats(t *testing.T) {
	mockStorage := NewMockStorage(t)
	service := NewEndpointService(mockStorage, "", 8)
	expectedStats := models.Statistics{URLs: 5, Users: 2}
	mockStorage.On("GetStats", mock.Anything).Return(expectedStats, nil)

	stats, err := service.GetStats(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, expectedStats, stats)
	mockStorage.AssertExpectations(t)
}
