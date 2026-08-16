// Package server sets up the grpc server with routes and middleware.
package server

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/max-marek-projects/shortener/api"
	"github.com/max-marek-projects/shortener/internal/audit"
	"github.com/max-marek-projects/shortener/internal/handlers"
	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/middlewares"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// GRPCServer wraps an http.Server with pre-configured middleware and routes.
type GRPCServer struct {
	*grpc.Server
	Addr              string
	WaitForBackground func()
}

// NewGRPCServer creates a new Server instance with the given address, handler, timeouts, auditor, and cookie secret.
// It sets up chi router with all necessary middleware and routes:
// - Recoverer, Gzip, Logger, Audit middleware for all routes.
// - Public routes: /ping, /{id}
// - Protected routes (with AuthMiddleware): POST /, /api/shorten, /api/shorten/batch, /api/user/urls (GET/DELETE).
func NewGRPCServer(
	addr string,
	h *handlers.GRPCHandler,
	readTimeout, writeTimeout time.Duration,
	auditor audit.Audit,
	cookieSecret string,
) *GRPCServer {
	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			middlewares.GRPCAuditInterceptor(auditor),
			middlewares.GRPCAuthInterceptor(cookieSecret),
		),
	)
	api.RegisterShortenerServiceServer(server, h)
	return &GRPCServer{
		Server:            server,
		Addr:              addr,
		WaitForBackground: func() {},
	}
}

// ListenAndServe starts the HTTP server and logs the address.
// Returns an error if the server cannot start.
func (s *GRPCServer) ListenAndServe() error {
	logger.Log.Info("Starting GRPC server", zap.String("address", s.Addr))
	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return fmt.Errorf("create grpc listener: %w", err)
	}
	if err := s.Serve(listener); err != nil {
		logger.Log.Error(
			"grpc server stopped",
			zap.Error(err),
		)
		return err
	}
	return nil
}

// Shutdown stops the HTTP server.
// Expects context.
// Returns an error if the server wasn't closed properly.
func (s *GRPCServer) Shutdown(ctx context.Context) error {
	s.GracefulStop()
	s.WaitForBackground()
	return nil
}
