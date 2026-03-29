package server

import (
	"net/http"

	"github.com/Starkiller2000Space-Projects/shortener/cmd/shortener/internal/handlers"
)

type Server struct {
	http.Server
}

func NewServer(addr string, h *handlers.Handler) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/{id}", h.IdHandler)
	mux.HandleFunc("/", h.PostUrlHandler)

	return &Server{
		Server: http.Server{
			Addr:    addr,
			Handler: mux,
		},
	}
}