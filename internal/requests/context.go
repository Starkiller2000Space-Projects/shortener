// Package requests provides helpers for parsing HTTP headers and context values.

package requests

import "context"

type contextKey string

const userIDKey contextKey = "userID"

// SetUserIDToContext adds the user ID to the context and returns the new context.
// This is used to propagate the authenticated user ID to downstream handlers.
func SetUserIDToContext(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserIDFromContext retrieves the user ID from the context.
// Returns the user ID and true if present, otherwise empty string and false.
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey).(string)
	return userID, ok
}
