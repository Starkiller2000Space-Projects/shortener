package config

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	RunAddr            string        `env:"SERVER_ADDRESS"`       // address and port to run server
	ShowAddr           string        `env:"BASE_URL"`             // address and port to show for short urls
	IdSize             int           `env:"ID_SIZE"`              // short link id length
	ReadTimeout        time.Duration `env:"READ_TIMEOUT"`         // server read timeout in seconds
	WriteTimeout       time.Duration `env:"WRITE_TIMEOUT"`        // server write timeout in seconds
	LoggerLevel        string        `env:"LOGGER_LEVEL"`         // logger level DEBUG / INFO / WARNING / ERROR / FATAL
	FileStoragePath    string        `env:"FILE_STORAGE_PATH"`    // file path to save shortened urls to
	DatabaseUrl        string        `env:"DATABASE_DSN"`         // database connection url
	CookieSecret       string        `env:"COOKIE_SECRET"`        // secret for cookie signature
	MaxParallelWorkers int           `env:"MAX_PARALLEL_WORKERS"` // max amount of parallel workers
}

// parse all flags from command line
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
	flag.StringVar(&config.RunAddr, "a", ":8080", "address and port to run server")
	flag.StringVar(&config.ShowAddr, "b", "", "address and port to show for short urls")
	flag.IntVar(&config.IdSize, "i", 8, "address and port to show for short urls")
	flag.StringVar(&config.LoggerLevel, "l", "INFO", "logger level")
	flag.StringVar(&config.FileStoragePath, "f", "", "file path to save shortened urls to")
	flag.StringVar(&config.DatabaseUrl, "d", "", "database connection url")
	flag.StringVar(&config.CookieSecret, "s", "", "cookie signing secret")
	flag.IntVar(&config.MaxParallelWorkers, "max-parallel-workers", 100, "maximum concurrent parallel operations")
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
	return &config
}
