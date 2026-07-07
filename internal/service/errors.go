// Package service implements the core URL shortening business logic.

package service

import (
	"errors"
)

var ErrorEmptyUrl = errors.New("Url is empty or contains only whitespace characters")
var ErrorEmptyBatch = errors.New("Empty batch received")
var ErrorDuplicate = errors.New("url already exists")
