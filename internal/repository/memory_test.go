package repository

import (
	"cmp"
	"context"
	"slices"
	"testing"

	"github.com/max-marek-projects/shortener/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// test addition to memory storage
func TestMemStorage_Add(t *testing.T) {
	// create storage
	store, err := NewMemStorage()
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
	// check new storage creation deletes data
	store2, err := NewMemStorage()
	require.NoError(t, err)
	got2, err := store2.Get(ctx, testId)
	assert.Error(t, err)
	assert.Equal(t, "", got2)

	// test cancel request
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	err = store.Add(ctx, row)
	assert.ErrorIs(t, err, context.Canceled)
	got, err = store.Get(ctx, testId)
	assert.Equal(t, "", got)
	assert.ErrorIs(t, err, context.Canceled)
}

func BenchmarkMemStorageAdd(b *testing.B) {
	storage, _ := NewMemStorage()
	ctx := context.Background()
	data := Row{
		ID:          "testid",
		OriginalURL: "https://example.com/very/long/url/for/testing/performance",
		UserID:      "user123",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data.ID = utils.GenerateId(8)
		_ = storage.Add(ctx, data)
	}
}

func BenchmarkMemStorageGet(b *testing.B) {
	storage, _ := NewMemStorage()
	ctx := context.Background()
	id := "existing"
	_ = storage.Add(ctx, Row{ID: id, OriginalURL: "https://example.com", UserID: "user"})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = storage.Get(ctx, id)
	}
}

// test ping memory storage
func TestMemStorage_Ping(t *testing.T) {
	store, err := NewMemStorage()
	require.NoError(t, err)
	ctx := context.Background()
	err = store.Ping(ctx)
	assert.NoError(t, err)
}

// test batch addition to memory storage
func TestMemStorage_AddBatch(t *testing.T) {
	// create storage
	store, err := NewMemStorage()
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
	// check new storage creation deletes data from storage
	store2, err := NewMemStorage()
	require.NoError(t, err)
	for _, row := range testRows {
		got, err := store2.Get(ctx, row.ID)
		assert.Error(t, err)
		assert.Equal(t, "", got)
	}

	// test cancel request
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	err = store.AddBatch(ctx, testRows)
	assert.ErrorIs(t, err, context.Canceled)
}

func BenchmarkMemStorageAddBatch(b *testing.B) {
	storage, _ := NewMemStorage()
	ctx := context.Background()
	items := make([]Row, 10)
	for i := 0; i < 10; i++ {
		items[i] = Row{
			ID:          utils.GenerateId(8),
			OriginalURL: "https://example.com/" + string(rune(i)),
			UserID:      "user",
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := range items {
			items[j].ID = utils.GenerateId(8)
		}
		_ = storage.AddBatch(ctx, items)
	}
}

// test get user urls
func TestMemStorage_GetUserUrls(t *testing.T) {
	// create storage
	store, err := NewMemStorage()
	require.NoError(t, err)
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
	err = store.AddBatch(ctx, testRows)
	require.NoError(t, err)
	// test get user urls
	expected := []UserURL{
		{ShortURL: "id1", OriginalURL: "http://example1.com"},
		{ShortURL: "id2", OriginalURL: "http://example2.com"},
		{ShortURL: "id3", OriginalURL: "http://example3.com"},
	}
	userUrls, err := store.GetUserURLs(ctx, userId1)
	assert.NoError(t, err)
	slices.SortFunc(userUrls, func(item1, item2 UserURL) int { return cmp.Compare(item1.ShortURL, item2.ShortURL) })
	assert.Equal(
		t,
		expected,
		userUrls,
	)

	// test cancel request
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	userUrls, err = store.GetUserURLs(ctx, userId1)
	assert.Nil(t, userUrls)
	assert.ErrorIs(t, err, context.Canceled)
}

func BenchmarkMemStorageGetUserURLs(b *testing.B) {
	storage, _ := NewMemStorage()
	ctx := context.Background()
	userID := "userX"
	for i := 0; i < 100; i++ {
		_ = storage.Add(ctx, Row{
			ID:          utils.GenerateId(8),
			OriginalURL: "https://example.com/" + string(rune(i)),
			UserID:      userID,
		})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = storage.GetUserURLs(ctx, userID)
	}
}

// test delete batch from memory storage
func TestMemStorage_DeleteBatch(t *testing.T) {
	// create storage
	store, err := NewMemStorage()
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

func BenchmarkMemStorageDeleteBatch(b *testing.B) {
	storage, _ := NewMemStorage()
	ctx := context.Background()
	userID := "userDel"
	ids := make([]string, 10)
	for i := 0; i < 10; i++ {
		id := utils.GenerateId(8)
		ids[i] = id
		_ = storage.Add(ctx, Row{ID: id, OriginalURL: "https://example.com", UserID: userID})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = storage.DeleteBatch(ctx, userID, ids)
		for _, id := range ids {
			_ = storage.Add(ctx, Row{ID: id, OriginalURL: "https://example.com", UserID: userID})
		}
	}
}
