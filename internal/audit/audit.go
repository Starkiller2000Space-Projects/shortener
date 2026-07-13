// Package audit implements an audit logging system with multiple observers.

package audit

import (
	"context"
	"sync"

	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/max-marek-projects/shortener/internal/models"
	"go.uber.org/zap"
)

// Observer is implemented by types that want to receive audit events.
// The Notify method is called for each event.
//
//go:generate mockery --name=Observer --inpackage --filename=mock_observer_test.gen.go --with-expecter
type Observer interface {
	Notify(event models.AuditEvent)
	Stop() error
}

// Audit defines the interface for broadcasting audit events.
// It allows registering observers and notifying them when an event occurs.
//
//go:generate mockery --name=Audit --output=../middlewares --outpkg=middlewares --filename=mock_audit_test.gen.go --with-expecter --structname=MockAudit
type Audit interface {
	Register(o Observer)
	NotifyAll(event models.AuditEvent) *sync.WaitGroup
}

// task is audit notification task.
type task struct {
	observer Observer
	event    models.AuditEvent
	wg       *sync.WaitGroup
}

// audit is the internal implementation of the Audit interface.
// It holds a list of observers and notifies them sequentially.
type audit struct {
	observers []Observer
	taskCh    chan task
	workers   int
	wgWorkers sync.WaitGroup
	once      sync.Once
}

// startWorkers starts all audit workers.
func (s *audit) startWorkers() {
	s.wgWorkers.Add(s.workers)
	for i := 0; i < s.workers; i++ {
		go s.worker()
	}
}

// worker handles audit tasks
func (s *audit) worker() {
	defer s.wgWorkers.Done()
	for t := range s.taskCh {
		func() {
			defer t.wg.Done()
			defer func() {
				if r := recover(); r != nil {
					logger.Log.Error(
						"notification panic",
						zap.Any("panic", r),
					)
				}
			}()
			t.observer.Notify(t.event)
		}()
	}
}

// Register adds a new observer to the audit list.
func (s *audit) Register(o Observer) {
	s.observers = append(s.observers, o)
}

// NotifyAll calls Notify on each registered observer in a separate goroutine.
func (s *audit) NotifyAll(event models.AuditEvent) *sync.WaitGroup {
	var wg sync.WaitGroup
	wg.Add(len(s.observers))

	for _, observer := range s.observers {
		s.taskCh <- task{
			observer: observer,
			event:    event,
			wg:       &wg,
		}
	}

	return &wg
}

// завершение работы
func (s *audit) Stop() error {
	s.once.Do(func() {
		close(s.taskCh)
		s.wgWorkers.Wait()
	})
	for _, o := range s.observers {
		err := o.Stop()
		if err != nil {
			return err
		}
	}
	return nil
}

// contextKey used for defining custom context keys
type contextKey string

// AuditKey is the context key used to store audit data in the request context.
var auditKey contextKey = "audit_data"

// SetAuditDataToContext adds the audit data to the context and returns the new context.
func SetAuditDataToContext(ctx context.Context, auditData *models.AuditData) context.Context {
	return context.WithValue(ctx, auditKey, auditData)
}

// GetAuditDataFromContext retrieves the audit data from the context.
// Returns the audit data and true if present, otherwise nil and false.
func GetAuditDataFromContext(ctx context.Context) (*models.AuditData, bool) {
	auditData, ok := ctx.Value(auditKey).(*models.AuditData)
	return auditData, ok
}

// InitAudit initializes a new audit instance and registers file and/or HTTP observers
// based on the provided file path and URL.
// If auditFile is non-empty, a FileAuditObserver is added.
// If auditURL is non-empty, an HTTPAuditObserver is added.
// Returns a pointer to the internal audit instance.
func InitAudit(auditFile, auditURL string, maxParallelWorkers int) (*audit, error) {
	auditItem := &audit{
		taskCh:  make(chan task, maxParallelWorkers),
		workers: 10,
	}
	auditItem.startWorkers()
	if auditFile != "" {
		fileObserver, err := NewFileAuditObserver(auditFile)
		if err != nil {
			return nil, err
		}
		auditItem.Register(fileObserver)
	}
	if auditURL != "" {
		auditItem.Register(NewHTTPAuditObserver(auditURL))
	}
	return auditItem, nil
}
