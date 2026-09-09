package service

import "errors"

var (
	ErrInvalidPurchase        = errors.New("invalid purchase")
	ErrInvalidProductID       = errors.New("invalid product id")
	ErrInvalidQuantity        = errors.New("invalid quantity")
	ErrIdempotencyKeyRequired = errors.New("idempotency key is required")
	ErrIdempotencyKeyConflict = errors.New("idempotency key was already used with a different purchase")
	ErrProductNotFound        = errors.New("product not found")
	ErrInsufficientStock      = errors.New("insufficient stock")
	ErrPaymentDeclined        = errors.New("payment declined")
)
