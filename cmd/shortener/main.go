package main

import (
	"log"

	"github.com/max-marek-projects/shortener/internal/config"
	"github.com/max-marek-projects/shortener/internal/handlers"
	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/server"
	"github.com/max-marek-projects/shortener/internal/service"
	"github.com/max-marek-projects/shortener/internal/storage"
	"go.uber.org/zap"
)

// entry point
func main() {
	configData := config.LoadConfig()
	err := logger.Initialize(configData.LoggerLevel)
	if err != nil {
		log.Fatalf("Unable to initialize logger: %v", err)
	}
	store, err := storage.NewStorage(configData.FileStoragePath, configData.IdSize)
	if err != nil {
		log.Fatalf("Unable to create storage: %v", err)
	}
	service := service.NewEndpointService(store, configData.ShowAddr)
	handler := handlers.NewHandler(service)
	srv := server.NewServer(configData.RunAddr, handler, configData.ReadTimeout, configData.WriteTimeout)
	err = srv.ListenAndServe()
	if err != nil {
		logger.Log.Fatal("Server shut down", zap.Error(err))
	}
}
