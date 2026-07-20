package repository

import (
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/max-marek-projects/shortener/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetStorage(t *testing.T) {
	err := godotenv.Load("../../.env")
	if err != nil && !os.IsNotExist(err) {
		require.NoError(t, err)
	}
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping integration test")
	}
	cfg := &config.Config{DatabaseURL: dsn, FileStoragePath: "", MigrationsPath: "../../migrations"}
	s, err := GetStorage(cfg)
	assert.NoError(t, err)
	_, ok := s.(*dbStorage)
	assert.True(t, ok)

	cfg = &config.Config{DatabaseURL: "", FileStoragePath: "/tmp/file"}
	s, err = GetStorage(cfg)
	assert.NoError(t, err)
	_, ok = s.(*fileStorage)
	assert.True(t, ok)

	cfg = &config.Config{DatabaseURL: "", FileStoragePath: ""}
	s, err = GetStorage(cfg)
	assert.NoError(t, err)
	_, ok = s.(*memStorage)
	assert.True(t, ok)

	cfg = &config.Config{DatabaseURL: dsn, FileStoragePath: "/tmp/file"}
	_, err = GetStorage(cfg)
	assert.ErrorIs(t, err, ErrMutuallyExclusiveFlags)
}
