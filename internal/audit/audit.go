package audit

import "github.com/max-marek-projects/shortener/internal/models"

// audit observer interface
type Observer interface {
	Notify(event models.AuditEvent)
}

// audit for all observers notification
type audit struct {
	observers []Observer
}

// register new observer
func (s *audit) Register(o Observer) {
	s.observers = append(s.observers, o)
}

// notify all observers
func (s *audit) NotifyAll(event models.AuditEvent) {
	for _, o := range s.observers {
		go o.Notify(event)
	}
}

// singleton audit variable
var Audit *audit

// Initialize audit
func InitAudit(auditFile, auditURL string) {
	Audit = &audit{}
	if auditFile != "" {
		Audit.Register(NewFileAuditObserver(auditFile))
	}
	if auditURL != "" {
		Audit.Register(NewHTTPAuditObserver(auditURL))
	}
}
