// Package repository defines storage interfaces and implementations (memory, file, DB).

package repository

import (
	"errors"
	"fmt"
)

// ErrNotFound is returned when a requested URL ID does not exist.
var ErrNotFound = errors.New("url not found")

// ErrMutuallyExclusiveFlags is returned when both database and file storage flags are provided.
var ErrMutuallyExclusiveFlags = errors.New("Mutually exclusive flags received")

// ErrGone is returned when a requested URL has been deleted (is_deleted=true).
var ErrGone = errors.New("url has been deleted")

// ErrAlreadyExists is returned when trying to insert a URL that already exists.
// It contains the existing short ID.
type ErrAlreadyExists struct {
	ExistingID string
}

// Error implements the error interface.
func (e *ErrAlreadyExists) Error() string {
	return fmt.Sprintf("url already exists with id %s", e.ExistingID)
}
