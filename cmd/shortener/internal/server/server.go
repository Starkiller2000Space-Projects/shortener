package server

import (
	"log"
	"net/http"

	"github.com/Starkiller2000Space-Projects/shortener/cmd/shortener/internal/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	http.Server
}

func NewServer(addr string, h *handlers.Handler) *Server {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.HandleFunc("/{id}", h.IdHandler)
	r.HandleFunc("/", h.PostUrlHandler)

	return &Server{
		Server: http.Server{
			Addr:    addr,
			Handler: r,
		},
	}
}

func (s *Server) ListenAndServe() error {
	log.Printf("Starting server on %s", s.Addr)
	return s.Server.ListenAndServe()
}
