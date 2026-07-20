package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDBConf(t *testing.T) {
	tests := []struct {
		name     string
		dbURL    string
		migPath  string
		expected *DBConf
	}{
		{
			name:    "typical",
			dbURL:   "postgres://user:pass@localhost:5432/db",
			migPath: "./migrations",
			expected: &DBConf{
				URL:             "postgres://user:pass@localhost:5432/db",
				MaxOpenConns:    10,
				MaxIdleConns:    5,
				ConnMaxLifetime: 5 * time.Minute,
				MigrationsPath:  "./migrations",
			},
		},
		{
			name:    "empty migrations path",
			dbURL:   "postgres://user@localhost/db",
			migPath: "",
			expected: &DBConf{
				URL:             "postgres://user@localhost/db",
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
