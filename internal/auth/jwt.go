package auth

import (
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const CookieName = "token"

type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

// create jwt and set it to cookie
func SetUserCookie(w http.ResponseWriter, userID, secretKey string) {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
		UserID: userID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(secretKey))

	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func extractUserIDFromToken(token string, secretKey string) (string, error) {
	claims := &Claims{}
	tokenData, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})
	if err != nil || !tokenData.Valid {
		return "", err
	}
	return claims.UserID, nil
}

// get user id from request cookie
func GetUserIDFromRequest(r *http.Request, secretKey string) (string, error) {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return "", err
	}
	return extractUserIDFromToken(cookie.Value, secretKey)
}
