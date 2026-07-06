package repository

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// test addition to file storage
func TestFileStorage_AddAndGet(t *testing.T) {
	// create temp tile
	tmpFile, err := os.CreateTemp("", "storage_test_*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()
	// create storage
	store, err := NewFileStorage(tmpFile.Name())
	require.NoError(t, err)
	ctx := context.Background()
	// check id is not present in storage
	testId := "id1"
	testUrl := "http://example.com"
	got, err := store.Get(ctx, testId)
	assert.Error(t, err)
	assert.Equal(t, "", got)
	// add test id
	row := Row{ID: testId, OriginalURL: testUrl, UserID: "u1"}
	err = store.Add(ctx, row)
	assert.NoError(t, err)
	// test id was added to storage
	got, err = store.Get(ctx, testId)
	assert.NoError(t, err)
	assert.Equal(t, testUrl, got)
	// check new storage creation leaves data in storage
	store2, err := NewFileStorage(tmpFile.Name())
	require.NoError(t, err)
	got2, err := store2.Get(ctx, testId)
	assert.NoError(t, err)
	assert.Equal(t, testUrl, got2)

	// test cancel request
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	err = store.Add(ctx, row)
	assert.ErrorIs(t, err, context.Canceled)
	got, err = store.Get(ctx, testId)
	assert.Equal(t, "", got)
	assert.ErrorIs(t, err, context.Canceled)
}

// test ping for file storage
func TestFileStorage_Ping(t *testing.T) {
	// ping non existing file
	nonExistingFileName := "non_existing_storage.json"
	store, err := NewFileStorage(nonExistingFileName)
	defer os.Remove(nonExistingFileName)
	require.NoError(t, err)
	ctx := context.Background()
	err = store.Ping(ctx)
	assert.NoError(t, err)
	// create temp tile
	tmpFileClosed, err := os.CreateTemp("", "storage_test_*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFileClosed.Name())
	tmpFileClosed.Close()
	// ping with existing file
	store, err = NewFileStorage(tmpFileClosed.Name())
	require.NoError(t, err)
	err = store.Ping(ctx)
	assert.NoError(t, err)
}

// test batch addition to file storage
func TestFileStorage_AddBatch(t *testing.T) {
	// create temp tile
	tmpFile, err := os.CreateTemp("", "storage_test_*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()
	// create storage
	store, err := NewFileStorage(tmpFile.Name())
	require.NoError(t, err)
	ctx := context.Background()
	// check id is not present in storage
	testRows := []Row{
		{ID: "id1", OriginalURL: "http://example1.com", UserID: "u1"},
		{ID: "id2", OriginalURL: "http://example2.com", UserID: "u2"},
		{ID: "id3", OriginalURL: "http://example3.com", UserID: "u3"},
		{ID: "id4", OriginalURL: "http://example4.com", UserID: "u4"},
		{ID: "id5", OriginalURL: "http://example5.com", UserID: "u5"},
	}
	for _, row := range testRows {
		got, err := store.Get(ctx, row.ID)
		assert.Error(t, err)
		assert.Equal(t, "", got)
	}
	// add test ids
	err = store.AddBatch(ctx, testRows)
	assert.NoError(t, err)
	// test ids were added to storage
	for _, row := range testRows {
		got, err := store.Get(ctx, row.ID)
		assert.NoError(t, err)
		assert.Equal(t, row.OriginalURL, got)
	}
	// check new storage creation leaves data in storage
	store2, err := NewFileStorage(tmpFile.Name())
	require.NoError(t, err)
	for _, row := range testRows {
		got, err := store2.Get(ctx, row.ID)
		assert.NoError(t, err)
		assert.Equal(t, row.OriginalURL, got)
	}

	// test cancel request
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	err = store.AddBatch(ctx, testRows)
	assert.ErrorIs(t, err, context.Canceled)
}

// test delete batch from storage
func TestFileStorage_DeleteBatch(t *testing.T) {
	// create temp tile
	tmpFile, err := os.CreateTemp("", "storage_test_*.json")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()
	// create storage
	store, err := NewFileStorage(tmpFile.Name())
	require.NoError(t, err)
	ctx := context.Background()
	// check id is not present in storage
	testRows := []Row{
		{ID: "id1", OriginalURL: "http://example1.com", UserID: "u1"},
		{ID: "id2", OriginalURL: "http://example2.com", UserID: "u2"},
		{ID: "id3", OriginalURL: "http://example3.com", UserID: "u3"},
		{ID: "id4", OriginalURL: "http://example4.com", UserID: "u4"},
		{ID: "id5", OriginalURL: "http://example5.com", UserID: "u5"},
	}
	// add test ids
	err = store.AddBatch(ctx, testRows)
	require.NoError(t, err)
	// test deleting nothing
	assert.NoError(t, store.DeleteBatch(ctx, "u1", []string{}))
	// delete batch
	err = store.DeleteBatch(ctx, "u1", []string{"id1", "id2"})
	assert.NoError(t, err)
	// check if current user id was deleted
	got, err := store.Get(ctx, "id1")
	assert.Error(t, err)
	assert.Equal(t, "", got)
	// check other user id was untouched
	got, err = store.Get(ctx, "id2")
	assert.NoError(t, err)
	assert.Equal(t, "http://example2.com", got)

	// test cancel request
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	err = store.DeleteBatch(ctx, "u1", []string{"id1", "id2"})
	assert.ErrorIs(t, err, context.Canceled)
}
