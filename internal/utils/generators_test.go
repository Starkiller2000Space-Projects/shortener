package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateId(t *testing.T) {
	sizes := []int{1, 5, 8, 10, 16}
	for _, size := range sizes {
		id := GenerateId(size)
		assert.Len(t, id, size, "ID length should be %d", size)
	}
}

func BenchmarkGenerateId(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GenerateId(8)
	}
}

func TestGenerateUserID(t *testing.T) {
	id1 := GenerateUserID()
	id2 := GenerateUserID()
	assert.NotEmpty(t, id1)
	assert.NotEqual(t, id1, id2, "user IDs should be unique")
}
