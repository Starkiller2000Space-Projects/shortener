// Package server sets up the HTTP server with routes and middleware.

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

// Server wraps an http.Server with pre-configured middleware and routes.
type Server struct {
	http.Server
}

// NewServer creates a new Server instance with the given address, handler, timeouts, auditor, and cookie secret.
// It sets up chi router with all necessary middleware and routes:
// - Recoverer, Gzip, Logger, Audit middleware for all routes.
// - Public routes: /ping, /{id}
// - Protected routes (with AuthMiddleware): POST /, /api/shorten, /api/shorten/batch, /api/user/urls (GET/DELETE).
func NewServer(addr string, h *handlers.Handler, readTimeout, writeTimeout time.Duration, auditor audit.Audit, cookieSecret string) *Server {
	r := chi.NewRouter()

	//middlewares
	r.Use(middleware.Recoverer)
	r.Use(middlewares.GzipMiddleware)
	r.Use(middlewares.LoggerMiddleware)
	r.Use(middlewares.AuditMiddleware(auditor))

	// public endpoints
	r.Get("/ping", h.PingHandler)
	r.Get("/{id}", h.IDHandler)

	// protected endpoints
	r.Group(func(protected chi.Router) {
		protected.Use(middlewares.AuthMiddleware(cookieSecret))
		protected.Post("/", h.PostURLHandler)
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

// ListenAndServe starts the HTTP server and logs the address.
// Returns an error if the server cannot start.
func (s *Server) ListenAndServe() error {
	logger.Log.Info("Starting server", zap.String("address", s.Addr))
	return s.Server.ListenAndServe()
}
