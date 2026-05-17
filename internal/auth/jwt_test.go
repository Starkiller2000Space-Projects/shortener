package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testSecret = "test-secret"

func TestSetAndGetUserID(t *testing.T) {
	userID := "test-user-123"
	w := httptest.NewRecorder()
	SetUserCookie(w, userID, testSecret)

	cookie := w.Result().Cookies()[0]
	assert.Equal(t, CookieName, cookie.Name)
	userID, err := extractUserIDFromToken(cookie.Value, testSecret)
	assert.NoError(t, err)
	assert.Equal(t, userID, userID)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)
	got, err := GetUserIDFromRequest(req, testSecret)
	assert.NoError(t, err)
	assert.Equal(t, userID, got)
}
