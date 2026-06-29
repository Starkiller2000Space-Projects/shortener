package models

type AuditAction string

const (
	AuditFollow  AuditAction = "follow"
	AuditShorten AuditAction = "shorten"
)

type AuditData struct {
	Action AuditAction
	URL    string
}

type AuditEvent struct {
	Ts     int64       `json:"ts"`
	Action AuditAction `json:"action"`
	UserID string      `json:"user_id"`
	URL    string      `json:"url"`
}
