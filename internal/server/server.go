package server

import (
	"net/http"
	"time"

	"github.com/Starkiller2000Space-Projects/shortener/internal/handlers"
	"github.com/Starkiller2000Space-Projects/shortener/internal/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

type Server struct {
	http.Server
}

func NewServer(addr string, h *handlers.Handler, readTimeout, writeTimeout time.Duration) *Server {
	r := chi.NewRouter()
	r.Use(logger.RequestsLogger)
	r.Use(middleware.Recoverer)
	r.HandleFunc("/{id}", h.IdHandler)
	r.HandleFunc("/", h.PostUrlHandler)

	apiRouter := chi.NewRouter()
	apiRouter.Post("/shorten", h.ShortenJSONHandler)
	r.Mount("/api", apiRouter)

	return &Server{
		Server: http.Server{
			Addr:         addr,
			Handler:      r,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
		},
	}
}

func (s *Server) ListenAndServe() error {
	logger.Log.Info("Starting server", zap.String("address", s.Addr))
	return s.Server.ListenAndServe()
}
