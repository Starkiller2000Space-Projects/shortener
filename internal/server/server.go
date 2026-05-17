package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/max-marek-projects/shortener/internal/handlers"
	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/middlewares"
	"go.uber.org/zap"
)

type Server struct {
	http.Server
}

func NewServer(addr string, h *handlers.Handler, readTimeout, writeTimeout time.Duration) *Server {
	r := chi.NewRouter()

	// middlewares
	r.Use(middlewares.GzipMiddleware)
	r.Use(middlewares.RequestsLogger)
	r.Use(middleware.Recoverer)

	// endpoints
	r.Get("/ping", h.PingHandler)
	r.Get("/{id}", h.IdHandler)
	r.Post("/", h.PostUrlHandler)

	// api router
	apiRouter := chi.NewRouter()
	apiRouter.Post("/shorten", h.ShortenJSONHandler)
	apiRouter.Post("/shorten/batch", h.PostBatchShortenHandler)

	// mount all routers
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
