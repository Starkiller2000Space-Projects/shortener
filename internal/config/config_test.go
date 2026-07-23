package config

import (
	"flag"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// test parsing variables from flags
func TestLoadConfig_Flags(t *testing.T) {
	// save original values
	oldArgs := os.Args
	oldFlagCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldFlagCommandLine
	}()
	// set arguments
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{
		"cmd",
		"-a=:9090",
		"-b=https://flag-example.com",
		"-i=25",
		"-l=ERROR",
		"-f=./storage_flag.json",
		"-d=postgres://flag:pass@localhost:5432/shortener?sslmode=disable",
		"-c=flag_secret",
		"-max-parallel-workers=374",
		"-audit-file=./audit_flag.log",
		"-audit-url=https://audit-flag.com",
		"-r=1",
		"-w=2",
		"-s=true",
	}

	cfg := LoadConfig()

	expected := &Config{ // #nosec G101
		RunAddr:            ":9090",
		ShowAddr:           "https://flag-example.com",
		IDSize:             25,
		LoggerLevel:        "ERROR",
		FileStoragePath:    "./storage_flag.json",
		DatabaseURL:        "postgres://flag:pass@localhost:5432/shortener?sslmode=disable",
		CookieSecret:       "flag_secret",
		MaxParallelWorkers: 374,
		AuditFile:          "./audit_flag.log",
		AuditURL:           "https://audit-flag.com",
		ReadTimeout:        1 * time.Second,
		WriteTimeout:       2 * time.Second,
		EnableHTTPS:        true,
	}

	for expectedValue, actualValue := range map[any]any{
		expected.RunAddr:            cfg.RunAddr,
		expected.ShowAddr:           cfg.ShowAddr,
		expected.IDSize:             cfg.IDSize,
		expected.LoggerLevel:        cfg.LoggerLevel,
		expected.FileStoragePath:    cfg.FileStoragePath,
		expected.DatabaseURL:        cfg.DatabaseURL,
		expected.CookieSecret:       cfg.CookieSecret,
		expected.MaxParallelWorkers: cfg.MaxParallelWorkers,
		expected.AuditFile:          cfg.AuditFile,
		expected.AuditURL:           cfg.AuditURL,
		expected.ReadTimeout:        cfg.ReadTimeout,
		expected.WriteTimeout:       cfg.WriteTimeout,
	} {
		assert.Equal(t, expectedValue, actualValue)
	}

}

// test parsing variables from environment
func TestLoadConfig_Env(t *testing.T) {
	oldArgs := os.Args
	oldFlagCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldFlagCommandLine
		// clear variables after test
		for _, envVar := range []string{
			"SERVER_ADDRESS",
			"BASE_URL",
			"ID_SIZE",
			"LOGGER_LEVEL",
			"FILE_STORAGE_PATH",
			"DATABASE_DSN",
			"COOKIE_SECRET",
			"MAX_PARALLEL_WORKERS",
			"AUDIT_FILE",
			"AUDIT_URL",
			"READ_TIMEOUT",
			"WRITE_TIMEOUT",
			"ENABLE_HTTPS",
		} {
			os.Unsetenv(envVar)
		}

	}()
	// clear flags and set env vars
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{"cmd"}
	for envName, envVal := range map[string]string{ // #nosec G101
		"SERVER_ADDRESS":       ":9999",
		"BASE_URL":             "https://env-example.com",
		"ID_SIZE":              "12",
		"LOGGER_LEVEL":         "DEBUG",
		"FILE_STORAGE_PATH":    "./storage_env.json",
		"DATABASE_DSN":         "postgres://env:pass@localhost:5432/shortener?sslmode=disable",
		"COOKIE_SECRET":        "env_secret",
		"MAX_PARALLEL_WORKERS": "50",
		"AUDIT_FILE":           "./audit_env.log",
		"AUDIT_URL":            "https://audit-env.com",
		"READ_TIMEOUT":         "5s",
		"WRITE_TIMEOUT":        "10s",
		"ENABLE_HTTPS":         "true",
	} {
		err := os.Setenv(envName, envVal)
		require.NoError(t, err)
	}

	cfg := LoadConfig()

	expected := &Config{ // #nosec G101
		RunAddr:            ":9999",
		ShowAddr:           "https://env-example.com",
		IDSize:             12,
		LoggerLevel:        "DEBUG",
		FileStoragePath:    "./storage_env.json",
		DatabaseURL:        "postgres://env:pass@localhost:5432/shortener?sslmode=disable",
		CookieSecret:       "env_secret",
		MaxParallelWorkers: 50,
		AuditFile:          "./audit_env.log",
		AuditURL:           "https://audit-env.com",
		ReadTimeout:        5 * time.Second,
		WriteTimeout:       10 * time.Second,
	}

	for expectedValue, actualValue := range map[any]any{
		expected.RunAddr:            cfg.RunAddr,
		expected.ShowAddr:           cfg.ShowAddr,
		expected.IDSize:             cfg.IDSize,
		expected.LoggerLevel:        cfg.LoggerLevel,
		expected.FileStoragePath:    cfg.FileStoragePath,
		expected.DatabaseURL:        cfg.DatabaseURL,
		expected.CookieSecret:       cfg.CookieSecret,
		expected.MaxParallelWorkers: cfg.MaxParallelWorkers,
		expected.AuditFile:          cfg.AuditFile,
		expected.AuditURL:           cfg.AuditURL,
		expected.ReadTimeout:        cfg.ReadTimeout,
		expected.WriteTimeout:       cfg.WriteTimeout,
	} {
		assert.Equal(t, expectedValue, actualValue)
	}
}

func TestLoadConfig_EnvOverridesFlags(t *testing.T) {
	oldArgs := os.Args
	oldFlagCommandLine := flag.CommandLine
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldFlagCommandLine
		for _, EnvVarName := range []string{
			"SERVER_ADDRESS",
			"BASE_URL",
			"LOGGER_LEVEL",
			"FILE_STORAGE_PATH",
			"DATABASE_DSN",
			"COOKIE_SECRET",
			"MAX_PARALLEL_WORKERS",
			"AUDIT_FILE",
			"AUDIT_URL",
			"READ_TIMEOUT",
			"WRITE_TIMEOUT",
		} {
			err := os.Unsetenv(EnvVarName)
			require.NoError(t, err)
		}
	}()
	// set flags and env vars
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	os.Args = []string{
		"cmd",
		"-a=:8080",
		"-b=https://flag-only.com",
	}
	err := os.Setenv("SERVER_ADDRESS", ":9999")
	require.NoError(t, err)
	err = os.Setenv("BASE_URL", "https://env-wins.com")
	require.NoError(t, err)
	// load config
	cfg := LoadConfig()
	// validate values
	expected := &Config{
		RunAddr:  ":9999",
		ShowAddr: "https://env-wins.com",
	}
	for expectedValue, actualValue := range map[any]any{
		expected.RunAddr:  cfg.RunAddr,
		expected.ShowAddr: cfg.ShowAddr,
	} {
		assert.Equal(t, expectedValue, actualValue)
	}
}
