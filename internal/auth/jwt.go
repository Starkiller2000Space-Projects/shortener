// Package auth provides JWT-based authentication and cookie management.

package auth

import (
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// cookieName is the key used for storing the JWT in HTTP cookies.
const cookieName = "token"

// Claims represents the JWT claims containing a user ID and standard registered claims.
type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

// SetUserCookie creates a JWT token with the given user ID,
// signs it with the secret key, and sets it as an HTTP cookie.
// The cookie is HttpOnly, SameSite=Lax, and expires in 24 hours.
// Returns an error if token signing fails.
func SetUserCookie(w http.ResponseWriter, userID, secretKey string) error {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
		UserID: userID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{ // #nosec G124
		Name:     cookieName,
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

// extractUserIDFromToken parses and validates a JWT token string.
// It returns the user ID from the claims or an error if the token is invalid.
func extractUserIDFromToken(token string, secretKey string) (string, error) {
	claims := &Claims{}
	tokenData, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})
	if err != nil {
		return "", err
	}
	if !tokenData.Valid {
		return "", errors.New("invalid token")
	}
	return claims.UserID, nil
}

// GetUserIDFromRequest extracts the user ID from the JWT stored in the request's cookie.
// Returns the user ID or an error if the cookie is missing or the token is invalid.
func GetUserIDFromRequest(r *http.Request, secretKey string) (string, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return "", err
	}
	return extractUserIDFromToken(cookie.Value, secretKey)
}
