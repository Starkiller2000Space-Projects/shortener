// Package config handles application configuration from flags, env, and .env.
package config

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config holds all configuration parameters for the application.
// Values are populated from environment variables, .env file, and command-line flags.
// Flags take precedence over environment variables.
type Config struct {
	RunAddr            string        `env:"SERVER_ADDRESS"`       // address and port to run server
	ShowAddr           string        `env:"BASE_URL"`             // address and port to show for short urls
	IDSize             int           `env:"ID_SIZE"`              // short link id length
	ReadTimeout        time.Duration `env:"READ_TIMEOUT"`         // server read timeout in seconds
	WriteTimeout       time.Duration `env:"WRITE_TIMEOUT"`        // server write timeout in seconds
	LoggerLevel        string        `env:"LOGGER_LEVEL"`         // logger level DEBUG / INFO / WARNING / ERROR / FATAL
	FileStoragePath    string        `env:"FILE_STORAGE_PATH"`    // file path to save shortened urls to
	DatabaseURL        string        `env:"DATABASE_DSN"`         // database connection url
	CookieSecret       string        `env:"COOKIE_SECRET"`        // secret for cookie signature
	MaxParallelWorkers int           `env:"MAX_PARALLEL_WORKERS"` // max amount of parallel workers
	AuditFile          string        `env:"AUDIT_FILE"`           // path to audit file
	AuditURL           string        `env:"AUDIT_URL"`            // audit service url
	MigrationsPath     string        `env:"MIGRATIONS"`           // path to migrations
	EnableHttps        bool          `env:"ENABLE_HTTPS"`         // enable https protocol

}

// LoadConfig parses configuration from .env file, environment variables,
// and command-line flags. Flags take precedence over environment variables.
// Returns a pointer to the populated Config struct.
// If .env is missing, it continues with environment variables and flags.
// If parsing fails, it logs a fatal error.
func LoadConfig() *Config {
	var config Config

	err := godotenv.Load()
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("No .env file found, using environment variables and flags")
		} else {
			log.Fatalf("Failed to load .env file: %v", err)
		}
	}
	// read flags directly to config
	flag.StringVar(&config.ShowAddr, "b", "", "address and port to show for short urls")
	flag.IntVar(&config.IDSize, "i", 8, "address and port to show for short urls")
	flag.StringVar(&config.LoggerLevel, "l", "INFO", "logger level")
	flag.StringVar(&config.FileStoragePath, "f", "", "file path to save shortened urls to")
	flag.StringVar(&config.DatabaseURL, "d", "", "database connection url")
	flag.StringVar(&config.CookieSecret, "c", "", "cookie signing secret")
	flag.IntVar(&config.MaxParallelWorkers, "max-parallel-workers", 100, "maximum concurrent parallel operations")
	flag.StringVar(&config.AuditFile, "audit-file", "", "path to audit file")
	flag.StringVar(&config.AuditURL, "audit-url", "", "audit service url")
	flag.StringVar(&config.MigrationsPath, "migrations", "./migrations", "path to database migrations")
	flag.BoolVar(&config.EnableHttps, "s", false, "enable https protocol")
	flag.StringVar(&config.RunAddr, "a", ":8080", "address and port to run server")
	// read flags to temp vars
	var readSec, writeSec int
	flag.IntVar(&readSec, "r", 30, "server read timeout in seconds")
	flag.IntVar(&writeSec, "w", 30, "server write timeout in seconds")
	// parse flags
	flag.Parse()
	// parse temp vars to config struct
	config.ReadTimeout = time.Duration(readSec) * time.Second
	config.WriteTimeout = time.Duration(writeSec) * time.Second
	if err := env.Parse(&config); err != nil {
		log.Printf("warning: failed to parse env: %v", err)
	}
	if config.EnableHttps && config.RunAddr == ":8080" {
		config.RunAddr = ":443"
	}
	return &config
}
