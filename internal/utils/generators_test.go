package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateID(t *testing.T) {
	sizes := []int{1, 5, 8, 10, 16}
	for _, size := range sizes {
		id := GenerateID(size)
		assert.Len(t, id, size, "ID length should be %d", size)
	}
}

func BenchmarkGenerateID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GenerateID(8)
	}
}

func TestGenerateUserID(t *testing.T) {
	id1 := GenerateUserID()
	id2 := GenerateUserID()
	assert.NotEmpty(t, id1)
	assert.NotEqual(t, id1, id2, "user IDs should be unique")
}

func BenchmarkGenerateUserID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateUserID()
	}
}
