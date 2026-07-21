package pool

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testResetter struct{ val int }

func (t *testResetter) Reset() { t.val = 0 }

func TestPool(t *testing.T) {
	p := NewPool(func() *testResetter { return &testResetter{} })
	obj := p.Get()
	obj.val = 42
	p.Put(obj)
	obj2 := p.Get()
	assert.Equal(t, 0, obj2.val)
}
