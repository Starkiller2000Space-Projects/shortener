package repository

import (
	"context"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/max-marek-projects/shortener/internal/config/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// create test db in container
func setupTestDB(t *testing.T) (*dbStorage, *db.DBConf, func()) {
	err := godotenv.Load("../../.env")
	require.NoError(t, err)
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping integration test")
	}
	config := db.NewDbConf(dsn)
	config.MigrationsPath = "../../migrations"
	storage, err := NewDBStorage(config)
	require.NoError(t, err)
	cleanup := func() {
		_, err := storage.storage.ExecContext(context.Background(), "TRUNCATE urls")
		if err != nil {
			t.Logf("failed to truncate: %v", err)
		}
	}
	return storage, config, cleanup
}

// test addition to db storage
func TestDBStorage_Add(t *testing.T) {
	// create storage
	store, dbConf, cleanup := setupTestDB(t)
	defer cleanup()
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
	store2, err := NewDBStorage(dbConf)
	require.NoError(t, err)
	got2, err := store2.Get(ctx, testId)
	assert.NoError(t, err)
	assert.Equal(t, testUrl, got2)
}

// test ping for file storage
func TestDBStorage_Ping(t *testing.T) {
	// create storage
	store, _, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	// ping
	err := store.Ping(ctx)
	assert.NoError(t, err)
}

// test batch addition to file storage
func TestDBStorage_AddBatch(t *testing.T) {
	// create storage
	store, dbConf, cleanup := setupTestDB(t)
	defer cleanup()
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
	err := store.AddBatch(ctx, testRows)
	assert.NoError(t, err)
	// test ids were added to storage
	for _, row := range testRows {
		got, err := store.Get(ctx, row.ID)
		assert.NoError(t, err)
		assert.Equal(t, row.OriginalURL, got)
	}
	// check new storage creation leaves data in storage
	store2, err := NewDBStorage(dbConf)
	require.NoError(t, err)
	for _, row := range testRows {
		got, err := store2.Get(ctx, row.ID)
		assert.NoError(t, err)
		assert.Equal(t, row.OriginalURL, got)
	}
}

// test get user urls
func TestDBStorage_GetUserUrls(t *testing.T) {
	// create storage
	store, _, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	// check id is not present in storage
	userId1 := "u1"
	testRows := []Row{
		{ID: "id1", OriginalURL: "http://example1.com", UserID: userId1},
		{ID: "id2", OriginalURL: "http://example2.com", UserID: userId1},
		{ID: "id3", OriginalURL: "http://example3.com", UserID: userId1},
		{ID: "id4", OriginalURL: "http://example4.com", UserID: "u2"},
		{ID: "id5", OriginalURL: "http://example5.com", UserID: "u2"},
	}
	// add test ids
	err := store.AddBatch(ctx, testRows)
	require.NoError(t, err)
	// test get user urls
	expected := []UserURL{
		{ShortURL: "id1", OriginalURL: "http://example1.com"},
		{ShortURL: "id2", OriginalURL: "http://example2.com"},
		{ShortURL: "id3", OriginalURL: "http://example3.com"},
	}
	userUrls, err := store.GetUserURLs(ctx, userId1)
	assert.NoError(t, err)
	assert.Equal(t, expected, userUrls)
}

// test delete batch from storage
func TestDBStorage_DeleteBatch(t *testing.T) {
	// create storage
	store, _, cleanup := setupTestDB(t)
	defer cleanup()
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
	err := store.AddBatch(ctx, testRows)
	require.NoError(t, err)
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
}
