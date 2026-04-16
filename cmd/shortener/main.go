package main

import (
	"log"

	"github.com/Starkiller2000Space-Projects/shortener/internal/config"
	"github.com/Starkiller2000Space-Projects/shortener/internal/handlers"
	"github.com/Starkiller2000Space-Projects/shortener/internal/logger"
	"github.com/Starkiller2000Space-Projects/shortener/internal/server"
	"github.com/Starkiller2000Space-Projects/shortener/internal/service"
	"github.com/Starkiller2000Space-Projects/shortener/internal/storage"
	"go.uber.org/zap"
)

// entry point
func main() {
	configData := config.LoadConfig()
	err := logger.Initialize(configData.LoggerLevel)
	if err != nil {
		log.Fatalf("Unable to initialize logger: %v", err)
	}
	store := storage.NewStorage(configData.IdSize)
	service := service.NewEndpointService(store, configData.ShowAddr)
	handler := handlers.NewHandler(service)
	srv := server.NewServer(configData.RunAddr, handler, configData.ReadTimeout, configData.WriteTimeout)
	err = srv.ListenAndServe()
	if err != nil {
		logger.Log.Fatal("Server shut down", zap.Error(err))
	}
}
