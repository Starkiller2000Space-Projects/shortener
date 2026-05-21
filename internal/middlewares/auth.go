package middlewares

import (
	"errors"
	"net/http"

	"github.com/max-marek-projects/shortener/internal/auth"
	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/utils"
	"go.uber.org/zap"
)

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
				} else {
					// invalid cookie -> 401 Unauthorized
					http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
					return
				}
			}
			if err != nil || userID == "" {
				// cookie is not valid
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			// cookie is valid

			ctx := utils.SetUserIDToContext(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
