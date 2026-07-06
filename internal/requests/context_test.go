package requests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetUserIDToContext_GetUserIDFromContext(t *testing.T) {
	ctx := context.Background()
	ctx = SetUserIDToContext(ctx, "user123")
	id, ok := GetUserIDFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, "user123", id)
}
