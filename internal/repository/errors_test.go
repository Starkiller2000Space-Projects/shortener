package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrAlreadyExists(t *testing.T) {
	testId := "12345"
	err := &ErrAlreadyExists{ExistingID: testId}
	assert.Contains(t, err.Error(), testId)
}
