// Package middlewares provides HTTP middleware for logging, auth, gzip, and audit.
package middlewares

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/max-marek-projects/shortener/internal/audit"
	"github.com/max-marek-projects/shortener/internal/auth"
	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/requests"
	"github.com/max-marek-projects/shortener/internal/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthMiddleware returns a middleware that authenticates users via JWT cookies.
// If the cookie is missing, it creates a new user ID and sets a new cookie.
// If the cookie is invalid, it returns 401 Unauthorized.
// It injects the user ID into the request context for downstream handlers.
func AuthMiddleware(secretKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// check if userID is exists and is valid
			userID, err := auth.GetUserIDFromRequest(r, secretKey)
			if err != nil {
				if errors.Is(err, http.ErrNoCookie) {
					// no cookie -> create new user
					userID = utils.GenerateUserID()
					if err := auth.SetUserCookie(w, userID, secretKey); err != nil {
						logger.Log.Error("Failed to set auth cookie", zap.Error(err))
						http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
						return
					}
					// user created successfully, proceed
				} else {
					// invalid cookie -> 401 Unauthorized
					http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
					return
				}
			}
			// now err is nil, userID is valid
			auditData, ok := audit.GetAuditDataFromContext(r.Context())
			if ok {
				auditData.UserID = userID
			}
			ctx := requests.SetUserIDToContext(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GRPCAuthInterceptor(
	secretKey string,
) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if info.FullMethod ==
			"/shortener.ShortenerService/ExpandURL" {
			return handler(ctx, req)
		}
		userID, err := getUserIDFromMetadata(ctx, secretKey)
		if err != nil {
			return nil, status.Error(
				codes.Unauthenticated,
				"invalid authorization",
			)
		}
		auditData, ok := audit.GetAuditDataFromContext(ctx)
		if ok {
			auditData.UserID = userID
		}
		ctx = requests.SetUserIDToContext(ctx, userID)
		return handler(ctx, req)
	}
}

func getUserIDFromMetadata(
	ctx context.Context,
	secretKey string,
) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(
			codes.Unauthenticated,
			"metadata is missing",
		)
	}
	values := md.Get("authorization")
	if len(values) == 0 {
		return "", status.Error(
			codes.Unauthenticated,
			"authorization is missing",
		)
	}
	value := strings.TrimSpace(values[0])
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(value, bearerPrefix) {
		return "", status.Error(
			codes.Unauthenticated,
			"invalid authorization scheme",
		)
	}
	token := strings.TrimSpace(
		strings.TrimPrefix(value, bearerPrefix),
	)
	if token == "" {
		return "", status.Error(
			codes.Unauthenticated,
			"token is empty",
		)
	}
	return auth.GetUserIDFromToken(token, secretKey)
}
