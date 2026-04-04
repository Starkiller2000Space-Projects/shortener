package main

import (
	"log"

	"github.com/Starkiller2000Space-Projects/shortener/internal/config"
	"github.com/Starkiller2000Space-Projects/shortener/internal/handler"
	"github.com/Starkiller2000Space-Projects/shortener/internal/server"
	"github.com/Starkiller2000Space-Projects/shortener/internal/service"
	"github.com/Starkiller2000Space-Projects/shortener/internal/storage"
)

// entry point
func main() {
    configData := config.LoadConfig()
	store := storage.NewStorage(configData.IdSize)
	service := service.NewEndpointService(store, configData.ShowAddr)
	handler := handler.NewHandler(service)
	srv := server.NewServer(configData.RunAddr, handler, configData.ReadTimeout, configData.WriteTimeout)
	log.Fatal(srv.ListenAndServe())
}
