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
	RunAddr      string        `env:"SERVER_ADDRESS"` // address and port to run server
	ShowAddr     string        `env:"BASE_URL"`       // address and port to show for short urls
	IdSize       int           `env:"ID_SIZE"`        // short link id length
	ReadTimeout  time.Duration `env:"READ_TIMEOUT"`   // server read timeout in seconds
	WriteTimeout time.Duration `env:"WRITE_TIMEOUT"`  // server write timeout in seconds
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

	flag.StringVar(&config.RunAddr, "a", ":8080", "address and port to run server")
	flag.StringVar(&config.ShowAddr, "b", "", "address and port to show for short urls")
	flag.IntVar(&config.IdSize, "i", 8, "address and port to show for short urls")
	var readSec, writeSec int
	flag.IntVar(&readSec, "r", 30, "server read timeout in seconds")
	flag.IntVar(&writeSec, "w", 30, "server write timeout in seconds")
	flag.Parse()
	// parse time duration params
	config.ReadTimeout = time.Duration(readSec) * time.Second
	config.WriteTimeout = time.Duration(writeSec) * time.Second
	if err := env.Parse(&config); err != nil {
		log.Printf("warning: failed to parse env: %v", err)
	}
	return &config
}
