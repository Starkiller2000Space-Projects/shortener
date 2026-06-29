package middlewares

import (
	"net/http"
	"time"

	"github.com/max-marek-projects/shortener/internal/audit"
	"github.com/max-marek-projects/shortener/internal/auth"
	"github.com/max-marek-projects/shortener/internal/models"
)

func AuditMiddleware(secretKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		auditFn := func(w http.ResponseWriter, r *http.Request) {
			userID, _ := auth.GetUserIDFromRequest(r, secretKey)
			next.ServeHTTP(w, r)
			if data, ok := r.Context().Value("audit_data").(models.AuditData); ok {
				event := models.AuditEvent{
					Ts:     time.Now().Unix(),
					Action: data.Action,
					UserID: userID,
					URL:    data.URL,
				}
				audit.Audit.NotifyAll(event)
			}
		}
		return http.HandlerFunc(auditFn)
	}
}
