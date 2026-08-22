// Package server sets up the HTTP server with routes and middleware.
package server

import (
	"context"
	"net"
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
	WaitForBackground func()
}

// NewServer creates a new Server instance with the given address, handler, timeouts, auditor, and cookie secret.
// It sets up chi router with all necessary middleware and routes:
// - Recoverer, Gzip, Logger, Audit middleware for all routes.
// - Public routes: /ping, /{id}
// - Protected routes (with AuthMiddleware): POST /, /api/shorten, /api/shorten/batch, /api/user/urls (GET/DELETE).
func NewServer(addr string, h *handlers.Handler, readTimeout, writeTimeout time.Duration, auditor audit.Audit, cookieSecret, trustedSubnet string) (*Server, error) {
	r := chi.NewRouter()

	//middlewares
	r.Use(middleware.Recoverer)
	r.Use(middlewares.GzipMiddleware)
	r.Use(middlewares.LoggerMiddleware)
	r.Use(middlewares.AuditMiddleware(auditor))

	var subnet *net.IPNet
	var err error
	if trustedSubnet != "" {
		_, subnet, err = net.ParseCIDR(trustedSubnet)
		if err != nil {
			return nil, err
		}
	}

	// public endpoints
	r.Get("/ping", h.PingHandler)
	r.Get("/{id}", h.ExpandURLHandler)
	r.Group(func(trustedSubnet chi.Router) {
		trustedSubnet.Use(middlewares.TrustedSubnetMiddleware(subnet))
		trustedSubnet.Get("/api/internal/stats", h.StatsHandler)
	})

	// protected endpoints
	r.Group(func(protected chi.Router) {
		protected.Use(middlewares.AuthMiddleware(cookieSecret))
		protected.Post("/", h.ShortenURLHandler)
		protected.Route("/api", func(api chi.Router) {
			api.Post("/shorten", h.ShortenJSONHandler)
			api.Post("/shorten/batch", h.PostBatchShortenHandler)
			api.Get("/user/urls", h.ListUserURLsHandler)
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
		WaitForBackground: func() {
			h.WaitForBackground()
		},
	}, nil
}

// ListenAndServeTLS starts the HTTP server and logs the address.
// Returns an error if the server cannot start.
func (s *Server) ListenAndServeTLS(certFile, keyFile string) error {
	logger.Log.Info("Starting HTTP server with certs", zap.String("address", s.Addr), zap.String("certificate", certFile), zap.String("key", certFile))
	return s.Server.ListenAndServeTLS(certFile, keyFile)
}

// ListenAndServe starts the HTTP server and logs the address.
// Returns an error if the server cannot start.
func (s *Server) ListenAndServe() error {
	logger.Log.Info("Starting HTTP server", zap.String("address", s.Addr))
	return s.Server.ListenAndServe()
}

// Shutdown stops the HTTP server.
// Expects context.
// Returns an error if the server wasn't closed properly.
func (s *Server) Shutdown(ctx context.Context) error {
	err := s.Server.Shutdown(ctx)
	s.WaitForBackground()
	return err
}
