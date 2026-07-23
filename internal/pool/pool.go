package pool

import (
	"reflect"
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
func NewPool[T Resetter]() *pool[T] {
	return &pool[T]{
		p: sync.Pool{
			New: func() any {
				var t T
				return reflect.New(reflect.TypeOf(t).Elem()).Interface()
			},
		},
	}
}

// Get
func (p *pool[T]) Get() T {
	return p.p.Get().(T)
}

// Put
func (p *pool[T]) Put(x T) {
	x.Reset()
	p.p.Put(x)
}
