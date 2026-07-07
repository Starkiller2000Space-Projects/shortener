package repository

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/max-marek-projects/shortener/internal/config/db"
	"github.com/max-marek-projects/shortener/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// create test db in container
func setupTestDB(t interface {
	mock.TestingT
	Skip(...any)
}) (*dbStorage, *db.DBConf, func()) {
	err := godotenv.Load("../../.env")
	if err != nil && !os.IsNotExist(err) {
		require.NoError(t, err)
	}
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping integration test")
	}
	config := db.NewDBConf(dsn)
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
	testID := "id1"
	testURL := "http://example.com"
	got, err := store.Get(ctx, testID)
	assert.Error(t, err)
	assert.Equal(t, "", got)
	// add test id
	row := Row{ID: testID, OriginalURL: testURL, UserID: "u1"}
	err = store.Add(ctx, row)
	assert.NoError(t, err)
	// test id was added to storage
	got, err = store.Get(ctx, testID)
	assert.NoError(t, err)
	assert.Equal(t, testURL, got)
	// check new storage creation leaves data in storage
	store2, err := NewDBStorage(dbConf)
	require.NoError(t, err)
	got2, err := store2.Get(ctx, testID)
	assert.NoError(t, err)
	assert.Equal(t, testURL, got2)
}

func BenchmarkDBStorageAdd(b *testing.B) {
	storage, _, cleanup := setupTestDB(b)
	defer cleanup()

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		id := utils.GenerateID(8)
		row := Row{
			ID:          id,
			OriginalURL: "https://example.com/" + id,
			UserID:      "user",
		}
		err := storage.Add(ctx, row)
		if err != nil {
			b.Fatalf("Add failed: %v", err)
		}
	}
}

func BenchmarkDBStorageGet(b *testing.B) {
	storage, _, cleanup := setupTestDB(b)
	defer cleanup()

	ctx := context.Background()
	id := "fixedID"
	row := Row{ID: id, OriginalURL: "https://example.com/fixed", UserID: "user"}
	err := storage.Add(ctx, row)
	if err != nil {
		b.Fatalf("failed to add initial row: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := storage.Get(ctx, id)
		if err != nil {
			b.Fatalf("Get failed: %v", err)
		}
	}
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

func BenchmarkDBStorageAddBatch(b *testing.B) {
	storage, _, cleanup := setupTestDB(b)
	defer cleanup()

	ctx := context.Background()

	for i := 0; b.Loop(); i++ {
		items := make([]Row, 10)
		for j := range 10 {
			items[j] = Row{
				ID:          utils.GenerateID(8),
				OriginalURL: "https://example.com/" + fmt.Sprint(i) + "/" + fmt.Sprint(j),
				UserID:      "user",
			}
		}
		err := storage.AddBatch(ctx, items)
		if err != nil {
			b.Fatalf("AddBatch failed: %v", err)
		}
	}
}

// test get user urls
func TestDBStorage_GetUserUrls(t *testing.T) {
	// create storage
	store, _, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	// check id is not present in storage
	userID1 := "u1"
	testRows := []Row{
		{ID: "id1", OriginalURL: "http://example1.com", UserID: userID1},
		{ID: "id2", OriginalURL: "http://example2.com", UserID: userID1},
		{ID: "id3", OriginalURL: "http://example3.com", UserID: userID1},
		{ID: "id4", OriginalURL: "http://example4.com", UserID: "u2"},
		{ID: "id5", OriginalURL: "http://example5.com", UserID: "u2"},
	}
	err := store.AddBatch(ctx, testRows)
	require.NoError(t, err)
	// test get user urls
	expected := []UserURL{
		{ShortURL: "id1", OriginalURL: "http://example1.com"},
		{ShortURL: "id2", OriginalURL: "http://example2.com"},
		{ShortURL: "id3", OriginalURL: "http://example3.com"},
	}
	userUrls, err := store.GetUserURLs(ctx, userID1)
	assert.NoError(t, err)
	assert.Equal(t, expected, userUrls)
}

func BenchmarkDBStorageGetUserURLs(b *testing.B) {
	storage, _, cleanup := setupTestDB(b)
	defer cleanup()

	ctx := context.Background()
	userID := "benchUser"
	for i := 0; i < 100; i++ {
		row := Row{
			ID:          utils.GenerateID(8),
			OriginalURL: "https://example.com/" + fmt.Sprint(i),
			UserID:      userID,
		}
		err := storage.Add(ctx, row)
		if err != nil {
			b.Fatalf("failed to add initial data: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := storage.GetUserURLs(ctx, userID)
		if err != nil {
			b.Fatalf("GetUserURLs failed: %v", err)
		}
	}
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

func BenchmarkDBStorageDeleteBatch(b *testing.B) {
	storage, _, cleanup := setupTestDB(b)
	defer cleanup()

	ctx := context.Background()
	userID := "delUser"
	ids := make([]string, 10)
	for i := 0; i < 10; i++ {
		id := utils.GenerateID(8)
		ids[i] = id
		row := Row{ID: id, OriginalURL: "https://example.com/" + id, UserID: userID}
		err := storage.Add(ctx, row)
		if err != nil {
			b.Fatalf("failed to add initial data: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := storage.DeleteBatch(ctx, userID, ids)
		if err != nil {
			b.Fatalf("DeleteBatch failed: %v", err)
		}
		for _, id := range ids {
			row := Row{ID: id, OriginalURL: "https://example.com/" + id, UserID: userID}
			err := storage.Add(ctx, row)
			if err != nil {
				b.Fatalf("failed to re-add after delete: %v", err)
			}
		}
	}
}
