// Package audit implements an audit logging system with multiple observers.

package audit

import "github.com/max-marek-projects/shortener/internal/models"

// Observer is implemented by types that want to receive audit events.
// The Notify method is called for each event.
//
//go:generate mockery --name=Observer --inpackage --filename=mock_observer_test.go --with-expecter
type Observer interface {
	Notify(event models.AuditEvent)
}

// Audit defines the interface for broadcasting audit events.
// It allows registering observers and notifying them when an event occurs.
//
//go:generate mockery --name=Audit --output=../middlewares --outpkg=middlewares --filename=mock_audit_test.go --with-expecter --structname=MockAudit
type Audit interface {
	Register(o Observer)
	NotifyAll(event models.AuditEvent)
}

// audit is the internal implementation of the Audit interface.
// It holds a list of observers and notifies them sequentially.
type audit struct {
	observers []Observer
}

// Register adds a new observer to the audit list.
func (s *audit) Register(o Observer) {
	s.observers = append(s.observers, o)
}

// NotifyAll calls Notify on each registered observer in a separate goroutine.
func (s *audit) NotifyAll(event models.AuditEvent) {
	for _, o := range s.observers {
		go o.Notify(event)
	}
}

// contextKey used for defining custom context keys
type contextKey string

// AuditKey is the context key used to store audit data in the request context.
var AuditKey contextKey = "audit_data"

// InitAudit initializes a new audit instance and registers file and/or HTTP observers
// based on the provided file path and URL.
// If auditFile is non-empty, a FileAuditObserver is added.
// If auditURL is non-empty, an HTTPAuditObserver is added.
// Returns a pointer to the internal audit instance.
func InitAudit(auditFile, auditURL string) *audit {
	auditItem := &audit{}
	if auditFile != "" {
		auditItem.Register(NewFileAuditObserver(auditFile))
	}
	if auditURL != "" {
		auditItem.Register(NewHTTPAuditObserver(auditURL))
	}
	return auditItem
}
