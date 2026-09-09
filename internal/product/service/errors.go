package service

import "errors"

var (
	ErrInvalidProductID   = errors.New("invalid product id")
	ErrInvalidProduct     = errors.New("invalid product")
	ErrInvalidSearchQuery = errors.New("search query is required")
)
