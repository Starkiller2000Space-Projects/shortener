package main

import (
	"context"
	"errors"
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

	_ "net/http/pprof"
)

// entry point
func main() {
	configData := config.LoadConfig()
	err := logger.Initialize(configData.LoggerLevel)
	if err != nil {
		log.Fatalf("Unable to initialize logger: %v", err)
	}
	store, err := repository.GetStorage(configData)
	if err != nil {
		log.Fatalf("Unable to create storage: %v", err)
	}
	service := service.NewEndpointService(store, configData.ShowAddr, configData.IdSize)
	handler := handlers.NewHandler(service, configData.MaxParallelWorkers)
	auditor := audit.InitAudit(configData.AuditFile, configData.AuditURL)
	srv := server.NewServer(configData.RunAddr, handler, configData.ReadTimeout, configData.WriteTimeout, auditor, configData.CookieSecret)

	// use pprof
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	// create separate goroutine
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.ListenAndServe()
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
