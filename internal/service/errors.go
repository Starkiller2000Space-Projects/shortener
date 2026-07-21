// Package service implements the core URL shortening business logic.
package service

import (
	"errors"
)

var ErrEmptyURL = errors.New("url is empty or contains only whitespace characters")
var ErrEmptyBatch = errors.New("empty batch received")
var ErrDuplicate = errors.New("url already exists")
