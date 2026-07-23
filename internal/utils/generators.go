// Package utils provides helper functions for ID generation and user IDs.
package utils

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/google/uuid"
)

// GenerateID generates a random alphanumeric ID of the specified length.
// It uses crypto/rand and base64 URL-safe encoding without padding.
// The result is truncated to the given size.
func GenerateID(size int) string {
	b := make([]byte, size)                               // empty bytes array
	rand.Read(b)                                          // fill bytes array with random bytes
	return base64.RawURLEncoding.EncodeToString(b)[:size] // encode bytes array to string
}

// GenerateUserID generates a new UUID v4 as a string.
// It uses google/uuid package.
func GenerateUserID() string {
	return uuid.New().String()
}
