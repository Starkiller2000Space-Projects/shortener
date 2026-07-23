// Package pool provides generic pool initialization and usage utilities.
package pool

import (
	"sync"
)

// Resetter describes types that can be handled by Pool
type Resetter interface {
	Reset()
}

// pool implements generic-pool, that can store T-type structures
type pool[T Resetter] struct {
	p sync.Pool
}

// NewPool creates new pool pointer
func NewPool[T Resetter](newFunc func() T) *pool[T] {
	return &pool[T]{
		p: sync.Pool{
			New: func() any {
				return newFunc()
			},
		},
	}
}

// Get gets item from cool
func (p *pool[T]) Get() T {
	return p.p.Get().(T)
}

// Put puts single item in pool
func (p *pool[T]) Put(x T) {
	x.Reset()
	p.p.Put(x)
}
