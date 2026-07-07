package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrAlreadyExists(t *testing.T) {
	testID := "12345"
	err := &ErrAlreadyExists{ExistingID: testID}
	assert.Contains(t, err.Error(), testID)
}
