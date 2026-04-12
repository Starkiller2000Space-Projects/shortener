package server

import (
	"log"
	"net/http"
	"time"

	"github.com/Starkiller2000Space-Projects/shortener/internal/handler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	http.Server
}

func NewServer(addr string, h *handler.Handler, readTimeout, writeTimeout int) *Server {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.HandleFunc("/{id}", h.IdHandler)
	r.HandleFunc("/", h.PostUrlHandler)

	return &Server{
		Server: http.Server{
			Addr:    addr,
			Handler: r,
			ReadTimeout: time.Duration(readTimeout) * time.Second,
			WriteTimeout: time.Duration(writeTimeout) * time.Second,
		},
	}
}

func (s *Server) ListenAndServe() error {
	log.Printf("Starting server on %s", s.Addr)
    return s.Server.ListenAndServe()
}