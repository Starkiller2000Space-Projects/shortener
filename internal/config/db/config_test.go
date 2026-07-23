package db

import (
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDBConf(t *testing.T) {
	err := godotenv.Load("../../.env")
	if err != nil && !os.IsNotExist(err) {
		require.NoError(t, err)
	}
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping integration test")
	}
	tests := []struct {
		name     string
		dbURL    string
		migPath  string
		expected *DBConf
	}{
		{
			name:    "typical",
			dbURL:   dsn,
			migPath: "./migrations",
			expected: &DBConf{
				URL:             dsn,
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: 5 * time.Minute,
				MigrationsPath:  "./migrations",
			},
		},
		{
			name:    "empty migrations path",
			dbURL:   dsn,
			migPath: "",
			expected: &DBConf{
				URL:             dsn,
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: 5 * time.Minute,
				MigrationsPath:  "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewDBConf(tt.dbURL, tt.migPath)
			assert.Equal(t, tt.expected, got)
		})
	}
}
