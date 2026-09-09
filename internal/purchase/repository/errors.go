package repository

import "errors"

var (
	ErrProductNotFound      = errors.New("product not found")
	ErrInsufficientStock    = errors.New("insufficient stock")
	ErrIdempotencyKeyExists = errors.New("idempotency key already exists")
	ErrOrderNotFound        = errors.New("order not found")
)
