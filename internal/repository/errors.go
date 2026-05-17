package repository

import (
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("url not found")
var ErrMutuallyExclusiveFlags = errors.New("Mutually exclusive flags received")
var ErrGone = errors.New("url has been deleted")

type ErrAlreadyExists struct {
	ExistingID string
}

func (e *ErrAlreadyExists) Error() string {
	return fmt.Sprintf("url already exists with id %s", e.ExistingID)
}
