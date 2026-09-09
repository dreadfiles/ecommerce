package repository

import (
	"context"

	"ecommerce/internal/purchase/domain"
)

type PurchaseRepository interface {
	GetQuote(ctx context.Context, items []domain.OrderItem) (*domain.Order, error)
	Create(ctx context.Context, order *domain.Order) error
	GetByIdempotencyKey(ctx context.Context, key string) (*domain.Order, error)
	GetByID(ctx context.Context, id int64) (*domain.Order, error)
	GetAll(ctx context.Context) ([]*domain.Order, error)
}
