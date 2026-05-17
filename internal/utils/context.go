package utils

import (
	"context"
)

type contextKey string

const UserIDKey contextKey = "userID"

// get user id from context
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(UserIDKey).(string)
	return id, ok
}
