// Package config handles application configuration from flags, env, and .env.
package config

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config holds all configuration parameters for the application.
// Values are populated from environment variables, .env file, and command-line flags.
// Flags take precedence over environment variables.
type Config struct {
	RunAddr            string        `env:"SERVER_ADDRESS" json:"server_address"`             // address and port to run server
	ShowAddr           string        `env:"BASE_URL" json:"base_url"`                         // address and port to show for short urls
	IDSize             int           `env:"ID_SIZE" json:"id_size"`                           // short link id length
	ReadTimeout        time.Duration `env:"READ_TIMEOUT" json:"read_timeout"`                 // server read timeout in seconds
	WriteTimeout       time.Duration `env:"WRITE_TIMEOUT" json:"write_timeout"`               // server write timeout in seconds
	LoggerLevel        string        `env:"LOGGER_LEVEL" json:"logger_level"`                 // logger level DEBUG / INFO / WARNING / ERROR / FATAL
	FileStoragePath    string        `env:"FILE_STORAGE_PATH" json:"file_storage_path"`       // file path to save shortened urls to
	DatabaseURL        string        `env:"DATABASE_DSN" json:"database_dsn"`                 // database connection url
	CookieSecret       string        `env:"COOKIE_SECRET" json:"cookie_secret"`               // secret for cookie signature
	MaxParallelWorkers int           `env:"MAX_PARALLEL_WORKERS" json:"max_parallel_workers"` // max amount of parallel workers
	AuditFile          string        `env:"AUDIT_FILE" json:"audit_file"`                     // path to audit file
	AuditURL           string        `env:"AUDIT_URL" json:"audit_url"`                       // audit service url
	MigrationsPath     string        `env:"MIGRATIONS" json:"migrations_path"`                // path to migrations
	EnableHTTPS        bool          `env:"ENABLE_HTTPS" json:"enable_https"`                 // enable https protocol
	ConfigFilePath     string        `env:"CONFIG" json:"-"`                                  // path to config file (ignored in JSON)
}

// LoadConfig parses configuration from .env file, environment variables,
// and command-line flags. Flags take precedence over environment variables.
// Returns a pointer to the populated Config struct.
// If .env is missing, it continues with environment variables and flags.
// If parsing fails, it logs a fatal error.
func LoadConfig() *Config {
	// default configuration
	config := &Config{
		RunAddr:            ":8080",
		IDSize:             8,
		LoggerLevel:        "INFO",
		MaxParallelWorkers: 100,
		MigrationsPath:     "./migrations",
		ReadTimeout:        time.Duration(30) * time.Second,
		WriteTimeout:       time.Duration(30) * time.Second,
	}

	// parse .env file
	err := godotenv.Load()
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("No .env file found, using environment variables and flags")
		} else {
			log.Fatalf("Failed to load .env file: %v", err)
		}
	}

	// parse config file path
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "-c" || args[i] == "-config":
			if i+1 < len(args) {
				config.ConfigFilePath = args[i+1]
			}
		case strings.HasPrefix(args[i], "-c="):
			config.ConfigFilePath = strings.TrimPrefix(args[i], "-c=")
		case strings.HasPrefix(args[i], "-config="):
			config.ConfigFilePath = strings.TrimPrefix(args[i], "-config=")
		}
	}
	if config.ConfigFilePath == "" {
		config.ConfigFilePath = os.Getenv("CONFIG")
	}

	if config.ConfigFilePath != "" {
		// #nosec G703 -- configuration file path is intentionally provided by the user.
		data, err := os.ReadFile(config.ConfigFilePath)
		if err != nil {
			log.Fatalf("failed to read config file: %v", err)
		}
		if err := json.Unmarshal(data, config); err != nil {
			log.Fatalf("failed to parse config file: %v", err)
		}
	}

	// read flags directly to config
	flag.StringVar(&config.RunAddr, "a", config.RunAddr, "address and port to run server")
	flag.StringVar(&config.ShowAddr, "b", config.ShowAddr, "address and port to show for short urls")
	flag.IntVar(&config.IDSize, "i", config.IDSize, "address and port to show for short urls")
	flag.StringVar(&config.LoggerLevel, "l", config.LoggerLevel, "logger level")
	flag.StringVar(&config.FileStoragePath, "f", config.FileStoragePath, "file path to save shortened urls to")
	flag.StringVar(&config.DatabaseURL, "d", config.DatabaseURL, "database connection url")
	flag.StringVar(&config.CookieSecret, "cookie-secret", config.CookieSecret, "cookie signing secret")
	flag.IntVar(&config.MaxParallelWorkers, "max-parallel-workers", config.MaxParallelWorkers, "maximum concurrent parallel operations")
	flag.StringVar(&config.AuditFile, "audit-file", config.AuditFile, "path to audit file")
	flag.StringVar(&config.AuditURL, "audit-url", config.AuditURL, "audit service url")
	flag.StringVar(&config.MigrationsPath, "migrations", config.MigrationsPath, "path to database migrations")
	flag.BoolVar(&config.EnableHTTPS, "s", config.EnableHTTPS, "enable https protocol")
	flag.StringVar(&config.ConfigFilePath, "c", config.ConfigFilePath, "config file path")
	flag.StringVar(&config.ConfigFilePath, "config", config.ConfigFilePath, "config file path")
	// read flags to temp vars
	var readSec, writeSec float64
	flag.Float64Var(&readSec, "r", config.ReadTimeout.Seconds(), "server read timeout in seconds")
	flag.Float64Var(&writeSec, "w", config.WriteTimeout.Seconds(), "server write timeout in seconds")
	// parse flags
	flag.Parse()
	// parse temp vars to config struct
	config.ReadTimeout = time.Duration(readSec) * time.Second
	config.WriteTimeout = time.Duration(writeSec) * time.Second
	if err := env.Parse(config); err != nil {
		log.Printf("warning: failed to parse env: %v", err)
	}
	if config.EnableHTTPS && config.RunAddr == ":8080" {
		config.RunAddr = ":443"
	}
	return config
}
