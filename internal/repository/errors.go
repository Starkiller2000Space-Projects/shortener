package repository

import "errors"

var ErrNotFound = errors.New("url not found")
var ErrMutuallyExclusiveFlags = errors.New("Mutually exclusive flags received")
