package repository

import "errors"

var (
	ErrNotFound = errors.New("product not found")
	ErrConflict = errors.New("product already exists")
)
