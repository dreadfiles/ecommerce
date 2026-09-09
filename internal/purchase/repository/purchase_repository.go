package repository

import (
	"context"

	"ecommerce/internal/purchase/domain"
)

type PurchaseRepository interface {
	Create(ctx context.Context, order *domain.Order) error
	GetByIdempotencyKey(ctx context.Context, key string) (*domain.Order, error)
}
