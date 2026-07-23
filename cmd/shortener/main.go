// Package main is the entry point for the URL shortener server.

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
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
// Expects original string
// Returns original string in case it was not empty. Else N/A
func orNA(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}

// entry point
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
		logger.Log.Error("Unable to create storage", zap.Error(err))
	}
	service := service.NewEndpointService(store, configData.ShowAddr, configData.IDSize)
	handler := handlers.NewHandler(service, configData.MaxParallelWorkers)
	auditor, err := audit.InitAudit(configData.AuditFile, configData.AuditURL, configData.MaxParallelWorkers)
	if err != nil {
		logger.Log.Error("Unable to initialize audit", zap.Error(err))
	}
	defer auditor.Stop()
	srv := server.NewServer(configData.RunAddr, handler, configData.ReadTimeout, configData.WriteTimeout, auditor, configData.CookieSecret)

	// use pprof
	go func() {
		srv := &http.Server{
			Addr:         "localhost:6060",
			ReadTimeout:  5 * configData.ReadTimeout,
			WriteTimeout: configData.WriteTimeout,
			IdleTimeout:  120 * time.Second,
		}
		log.Println(srv.ListenAndServe())
	}()
	// create separate goroutine
	serverErr := make(chan error, 1)
	go func() {
		if configData.EnableHttps {
			serverErr <- srv.ListenAndServeTLS("server.pem", "server.key")
		} else {
			serverErr <- srv.ListenAndServe()
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)

	select {
	case sig := <-stop:
		logger.Log.Info("Shutdown signal received",
			zap.String("signal", sig.String()),
		)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Log.Error("Graceful shutdown failed", zap.Error(err))
		} else {
			logger.Log.Info("Server stopped gracefully")
		}

	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Fatal("Server stopped with error", zap.Error(err))
		}
	}
}
