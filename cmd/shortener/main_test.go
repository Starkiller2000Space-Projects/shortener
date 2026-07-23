package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOrNA(t *testing.T) {
	assert.Equal(t, "N/A", orNA(""))
	assert.Equal(t, "value", orNA("value"))
}
