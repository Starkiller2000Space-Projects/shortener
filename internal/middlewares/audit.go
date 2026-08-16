// Package middlewares provides HTTP middleware for logging, auth, gzip, and audit.
package middlewares

import (
	"context"
	"net/http"
	"time"

	"github.com/max-marek-projects/shortener/internal/audit"
	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/models"
	"google.golang.org/grpc"
)

func notifyAudit(auditor audit.Audit, auditData *models.AuditData) {
	if auditData == nil || auditData.Action == "" {
		return
	}

	logger.Log.Debug("Received audit data")

	event := models.AuditEvent{
		Timestamp: time.Now().Unix(),
		Action:    auditData.Action,
		UserID:    auditData.UserID,
		URL:       auditData.URL,
	}

	auditor.NotifyAll(event)
}

// AuditMiddleware returns a middleware that captures audit data from the request context
// and notifies all registered audit observers after the request is handled.
// It injects a *models.AuditData into the request context under the audit.AuditKey.
// The middleware logs a debug message when audit data is received.
func AuditMiddleware(auditor audit.Audit) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auditData := &models.AuditData{}
			ctx := audit.SetAuditDataToContext(r.Context(), auditData)
			next.ServeHTTP(w, r.WithContext(ctx))
			notifyAudit(auditor, auditData)
		})
	}
}

func GRPCAuditInterceptor(auditor audit.Audit) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		auditData := &models.AuditData{}
		ctx = audit.SetAuditDataToContext(ctx, auditData)
		resp, err := handler(ctx, req)
		notifyAudit(auditor, auditData)
		return resp, err
	}
}
