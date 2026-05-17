package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemStorage(t *testing.T) {
	store, err := NewMemStorage()
	require.NoError(t, err)

	ctx := context.Background()
	row := Row{ID: "id1", OriginalURL: "http://example.com", UserID: "u1"}

	// Add
	err = store.Add(ctx, row)
	assert.NoError(t, err)

	// Get
	got, err := store.Get(ctx, "id1")
	assert.NoError(t, err)
	assert.Equal(t, "http://example.com", got)

	// Get not found
	_, err = store.Get(ctx, "missing")
	assert.ErrorIs(t, err, ErrNotFound)

	// GetUserURLs
	urls, err := store.GetUserURLs(ctx, "u1")
	assert.NoError(t, err)
	assert.Len(t, urls, 1)
	assert.Equal(t, "id1", urls[0].ShortURL)
	assert.Equal(t, "http://example.com", urls[0].OriginalURL)

	// AddBatch
	items := []Row{
		{ID: "id2", OriginalURL: "http://example2.com", UserID: "u1"},
		{ID: "id3", OriginalURL: "http://example3.com", UserID: "u2"},
	}
	err = store.AddBatch(ctx, items)
	assert.NoError(t, err)

	urls, err = store.GetUserURLs(ctx, "u1")
	assert.NoError(t, err)
	assert.Len(t, urls, 2)

	// DeleteBatch
	err = store.DeleteBatch(ctx, "u1", []string{"id1", "id2"})
	assert.NoError(t, err)

	// after delete, Get should return ErrGone
	_, err = store.Get(ctx, "id1")
	assert.ErrorIs(t, err, ErrGone)

	// GetUserURLs should not return deleted
	urls, err = store.GetUserURLs(ctx, "u1")
	assert.NoError(t, err)
	assert.Len(t, urls, 0)

	// Ping
	err = store.Ping(ctx)
	assert.NoError(t, err)
}
