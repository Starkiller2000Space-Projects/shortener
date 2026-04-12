package main

import (
	"log"

	"github.com/Starkiller2000Space-Projects/shortener/cmd/shortener/internal/handlers"
	"github.com/Starkiller2000Space-Projects/shortener/cmd/shortener/internal/server"
	"github.com/Starkiller2000Space-Projects/shortener/cmd/shortener/internal/storage"
)

// entry point
func main() {
	const (
		idSize = 8
		addr   = ":8080"
	)

	store := storage.NewStorage(idSize)
	handler := handlers.NewHandler(store)
	srv := server.NewServer(addr, handler)
	log.Fatal(srv.ListenAndServe())
}
