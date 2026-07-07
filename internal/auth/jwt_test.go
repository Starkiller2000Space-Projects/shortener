package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret"

func TestSetAndGetUserID(t *testing.T) {
	testUserID := "test-user-123"
	w := httptest.NewRecorder()
	err := SetUserCookie(w, testUserID, testSecret)
	require.NoError(t, err)

	response := w.Result()
	defer response.Body.Close()
	cookie := response.Cookies()[0]
	assert.Equal(t, cookieName, cookie.Name)
	userID, err := extractUserIDFromToken(cookie.Value, testSecret)
	assert.NoError(t, err)
	assert.Equal(t, testUserID, userID)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)
	got, err := GetUserIDFromRequest(req, testSecret)
	assert.NoError(t, err)
	assert.Equal(t, testUserID, got)
}
