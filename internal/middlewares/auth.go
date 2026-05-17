package middlewares

import (
	"context"
	"net/http"

	"github.com/max-marek-projects/shortener/internal/auth"
	"github.com/max-marek-projects/shortener/internal/utils"
)

func AuthMiddleware(secretKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := r.Cookie(auth.CookieName) // check cookie
			if err != nil {
				userID := utils.GenerateUserID()
				auth.SetUserCookie(w, userID, secretKey)
				ctx := context.WithValue(r.Context(), utils.UserIDKey, userID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
			// check if userID is valid
			userID, err := auth.GetUserIDFromRequest(r, secretKey)
			if err != nil || userID == "" {
				// cookie is not valid
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			// cookie is valid
			ctx := context.WithValue(r.Context(), utils.UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
