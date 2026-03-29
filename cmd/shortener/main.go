package main

import (
	"log"

	"github.com/Starkiller2000Space-Projects/shortener/internal/config"
	"github.com/Starkiller2000Space-Projects/shortener/internal/handlers"
	"github.com/Starkiller2000Space-Projects/shortener/internal/server"
	"github.com/Starkiller2000Space-Projects/shortener/internal/storage"
)

// entry point
func main() {
    configData := config.LoadConfig()
	store := storage.NewStorage(configData.IdSize)
	handler := handlers.NewHandler(store, configData.ShowAddr)
	srv := server.NewServer(configData.RunAddr, handler)
	log.Fatal(srv.ListenAndServe())
}
