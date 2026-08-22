// Package middlewares provides HTTP middleware for logging, auth, gzip, and audit.
package middlewares

import (
	"net"
	"net/http"
)

// TrustedSubnetMiddleware returns a middleware that checks if .
// If the cookie is missing, it creates a new user ID and sets a new cookie.
// If the cookie is invalid, it returns 401 Unauthorized.
// It injects the user ID into the request context for downstream handlers.
func TrustedSubnetMiddleware(trustedSubnet *net.IPNet) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ipStr := r.Header.Get("X-Real-IP")
			if ipStr == "" {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			ip := net.ParseIP(ipStr)
			if ip == nil {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			if trustedSubnet == nil {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			if !trustedSubnet.Contains(ip) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
