package repository

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileStorage(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "storage_test_*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := NewFileStorage(tmpFile.Name())
	require.NoError(t, err)

	ctx := context.Background()
	row := Row{ID: "id1", OriginalURL: "http://example.com", UserID: "u1"}

	err = store.Add(ctx, row)
	assert.NoError(t, err)

	got, err := store.Get(ctx, "id1")
	assert.NoError(t, err)
	assert.Equal(t, "http://example.com", got)

	store2, err := NewFileStorage(tmpFile.Name())
	require.NoError(t, err)
	got2, err := store2.Get(ctx, "id1")
	assert.NoError(t, err)
	assert.Equal(t, "http://example.com", got2)
}
