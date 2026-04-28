package utils

import (
	"crypto/rand"
	"encoding/base64"
)

// generate ID  for url with desired size
func GenerateId(size int) string {
	b := make([]byte, size)                               // empty bytes array
	rand.Read(b)                                          // fill bytes array with random bytes
	return base64.RawURLEncoding.EncodeToString(b)[:size] // encode bytes array to string
}
