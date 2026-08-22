// Package main is the entry point for the URL shortener server.

package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/max-marek-projects/shortener/internal/audit"
	"github.com/max-marek-projects/shortener/internal/config"
	"github.com/max-marek-projects/shortener/internal/handlers"
	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/repository"
	"github.com/max-marek-projects/shortener/internal/server"
	"github.com/max-marek-projects/shortener/internal/service"
	"go.uber.org/zap"

	_ "net/http/pprof" // #nosec G108
)

// global variables that can be rewritten by flags
var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

// orNA replaces empty string value with N/A
func orNA(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}

func main() {
	fmt.Println("Build version:", orNA(buildVersion))
	fmt.Println("Build date:", orNA(buildDate))
	fmt.Println("Build commit:", orNA(buildCommit))
	configData := config.LoadConfig()
	err := logger.Initialize(configData.LoggerLevel)
	if err != nil {
		log.Fatalf("Unable to initialize logger: %v", err)
	}
	store, err := repository.GetStorage(configData)
	if err != nil {
		logger.Log.Fatal("Unable to create storage", zap.Error(err))
	}
	service := service.NewEndpointService(store, configData.ShowAddr, configData.IDSize)
	httpHandler := handlers.NewHandler(service, configData.MaxParallelWorkers, &sync.WaitGroup{})
	grpcHandler := handlers.NewGRPCHandler(service)
	auditor, err := audit.InitAudit(configData.AuditFile, configData.AuditURL, configData.MaxParallelWorkers)
	if err != nil {
		logger.Log.Fatal("Unable to initialize audit", zap.Error(err))
	}
	defer auditor.Stop()

	// HTTP-server
	httpSrv, err := server.NewServer(configData.RunAddr, httpHandler, configData.ReadTimeout, configData.WriteTimeout, auditor, configData.CookieSecret, configData.TrustedSubnet)
	if err != nil {
		logger.Log.Fatal("Unable to create requests handler", zap.Error(err))
	}

	// gRPC-server
	var tlsConfig *tls.Config
	if configData.EnableHTTPS {
		cert, err := tls.LoadX509KeyPair("server.pem", "server.key")
		if err != nil {
			logger.Log.Fatal("failed to load TLS certificates", zap.Error(err))
		}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
	}
	grpcSrv := server.NewGRPCServer(
		configData.GRPCAddr,
		grpcHandler,
		configData.ReadTimeout,
		configData.WriteTimeout,
		auditor,
		configData.CookieSecret,
		tlsConfig,
	)

	// pprof
	go func() {
		srv := &http.Server{
			Addr:         "localhost:6060",
			ReadTimeout:  5 * configData.ReadTimeout,
			WriteTimeout: configData.WriteTimeout,
			IdleTimeout:  120 * time.Second,
		}
		if err := srv.ListenAndServe(); err != nil {
			logger.Log.Info("error running server", zap.Error(err))
		}
		defer srv.Shutdown(context.Background())
	}()

	// run http in separate goroutine
	httpErr := make(chan error, 1)
	go func() {
		if configData.EnableHTTPS {
			httpErr <- httpSrv.ListenAndServeTLS("server.pem", "server.key")
		} else {
			httpErr <- httpSrv.ListenAndServe()
		}
	}()

	// run gRPC in separate goroutine
	grpcErr := make(chan error, 1)
	go func() {
		grpcErr <- grpcSrv.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer signal.Stop(stop)

	// wait for shutdown signal or error
	select {
	case sig := <-stop:
		logger.Log.Info("Shutdown signal received", zap.String("signal", sig.String()))

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Graceful shutdown HTTP
		if err := httpSrv.Shutdown(ctx); err != nil {
			logger.Log.Error("HTTP graceful shutdown failed", zap.Error(err))
		} else {
			logger.Log.Info("HTTP server stopped gracefully")
		}

		// Graceful shutdown gRPC
		if err := grpcSrv.Shutdown(ctx); err != nil {
			logger.Log.Error("gRPC graceful shutdown failed", zap.Error(err))
		} else {
			logger.Log.Info("gRPC server stopped gracefully")
		}

	case err := <-httpErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Fatal("HTTP server stopped with error", zap.Error(err))
		}
	case err := <-grpcErr:
		if err != nil {
			logger.Log.Fatal("gRPC server stopped with error", zap.Error(err))
		}
	}
}
