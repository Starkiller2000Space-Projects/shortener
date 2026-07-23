// Package middlewares provides HTTP middleware for logging, auth, gzip, and audit.
package middlewares

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/max-marek-projects/shortener/internal/logger"
	"go.uber.org/zap"
)

// responseDataPool reuses responseData objects to reduce allocations.
var responseDataPool = sync.Pool{
	New: func() any { return &responseData{} },
}

// LoggerMiddleware returns a middleware that logs each HTTP request with details:
// URI, method, status code, duration, and response size.
// It uses the global logger (logger.Log) at Info level.
// The middleware stores response data using a custom loggingResponseWriter.
func LoggerMiddleware(h http.Handler) http.Handler {
	logFn := func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		responseData := responseDataPool.Get().(*responseData)
		responseData.status = 0
		responseData.size = 0
		defer responseDataPool.Put(responseData)
		lw := loggingResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
		}
		h.ServeHTTP(&lw, r)

		duration := time.Since(start)

		logger.Log.Info(
			"Processed request",
			zap.String("uri", r.RequestURI),
			zap.String("method", r.Method),
			zap.Int("status", responseData.status),
			zap.Duration("duration", duration),
			zap.Int("size", responseData.size),
		)
	}
	return http.HandlerFunc(logFn)
}

type (
	// generate:reset
	//
	// responseData stores the status code and response size for logging purposes.
	responseData struct {
		status int
		size   int
	}

	// loggingResponseWriter wraps an http.ResponseWriter and records
	// the status code and number of bytes written.
	loggingResponseWriter struct {
		http.ResponseWriter
		responseData *responseData
	}
)

// Write writes the response data and records the number of bytes written.
func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	if err != nil {
		return size, fmt.Errorf("failed to write response: %w", err)
	}
	return size, err
}

// WriteHeader records the status code and then writes it to the underlying ResponseWriter.
func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}
