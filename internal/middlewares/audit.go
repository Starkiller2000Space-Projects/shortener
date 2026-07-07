// Package middlewares provides HTTP middleware for logging, auth, gzip, and audit.

package middlewares

import (
	"context"
	"net/http"
	"time"

	"github.com/max-marek-projects/shortener/internal/audit"
	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/models"
	"github.com/max-marek-projects/shortener/internal/requests"
)

// AuditMiddleware returns a middleware that captures audit data from the request context
// and notifies all registered audit observers after the request is handled.
// It injects a *models.AuditData into the request context under the audit.AuditKey.
// The middleware logs a debug message when audit data is received.
func AuditMiddleware(auditor audit.Audit) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		auditFn := func(w http.ResponseWriter, r *http.Request) {
			userID, _ := requests.GetUserIDFromContext(r.Context())
			auditData := &models.AuditData{}
			ctx := context.WithValue(r.Context(), audit.AuditKey, auditData)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
			if auditData.Action != "" {
				logger.Log.Debug("Received audit data")
				event := models.AuditEvent{
					Timestamp: time.Now().Unix(),
					Action:    auditData.Action,
					UserID:    userID,
					URL:       auditData.URL,
				}
				auditor.NotifyAll(event)
			}
		}
		return http.HandlerFunc(auditFn)
	}
}
