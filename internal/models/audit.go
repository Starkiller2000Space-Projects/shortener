// Package models defines data structures used across the application.
package models

// AuditAction represents the type of audit event.
type AuditAction string

const (
	AuditFollow  AuditAction = "follow"
	AuditShorten AuditAction = "shorten"
)

// AuditData is stored in the request context to accumulate audit information.
type AuditData struct {
	Action AuditAction
	URL    string
	UserID string
}

// AuditEvent represents an audit log entry.
type AuditEvent struct {
	Timestamp int64       `json:"ts"`
	Action    AuditAction `json:"action"`
	UserID    string      `json:"user_id"`
	URL       string      `json:"url"`
}
