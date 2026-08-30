package server

import (
	"testing"
	"time"

	"github.com/max-marek-projects/shortener/internal/handlers"
	"github.com/stretchr/testify/assert"
)

func TestGRPCServer_New(t *testing.T) {
	mockSvc := NewMockService(t)
	grpcHandler := handlers.NewGRPCHandler(mockSvc)
	auditor := NewMockAudit(t)

	srv := NewGRPCServer(":3200", grpcHandler, 5*time.Second, 5*time.Second, auditor, "secret", nil)
	assert.NotNil(t, srv)
	assert.Equal(t, ":3200", srv.Addr)
	assert.NotNil(t, srv.Server)
	assert.NotNil(t, srv.WaitForBackground)
}
