package audit

import "github.com/max-marek-projects/shortener/internal/models"

// audit observer interface
//
//go:generate mockery --name=Observer --inpackage --filename=mock_observer_test.go --with-expecter
type Observer interface {
	Notify(event models.AuditEvent)
}

//go:generate mockery --name=Audit --output=../middlewares --outpkg=middlewares --filename=mock_audit_test.go --with-expecter --structname=MockAudit
type Audit interface {
	Register(o Observer)
	NotifyAll(event models.AuditEvent)
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

var AuditKey string = "audit_data"

// Initialize audit
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
