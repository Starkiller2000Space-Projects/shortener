package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/max-marek-projects/shortener/internal/audit"
	"github.com/max-marek-projects/shortener/internal/handlers"
	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/middlewares"
	"go.uber.org/zap"
)

type Server struct {
	http.Server
}

func NewServer(addr string, h *handlers.Handler, readTimeout, writeTimeout time.Duration, auditor audit.Audit, cookieSecret string) *Server {
	r := chi.NewRouter()

	//middlewares
	r.Use(middleware.Recoverer)
	r.Use(middlewares.GzipMiddleware)
	r.Use(middlewares.LoggerMiddleware)
	r.Use(middlewares.AuditMiddleware(auditor))

	// public endpoints
	r.Get("/ping", h.PingHandler)
	r.Get("/{id}", h.IdHandler)

	// protected endpoints
	r.Group(func(protected chi.Router) {
		protected.Use(middlewares.AuthMiddleware(cookieSecret))
		protected.Post("/", h.PostUrlHandler)
		protected.Route("/api", func(api chi.Router) {
			api.Post("/shorten", h.ShortenJSONHandler)
			api.Post("/shorten/batch", h.PostBatchShortenHandler)
			api.Get("/user/urls", h.GetUserURLsHandler)
			api.Delete("/user/urls", h.DeleteUserURLsHandler)
		})
	})

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
