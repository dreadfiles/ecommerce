package repository

import (
	"context"

	"ecommerce/internal/product/domain"
)

type ProductRepository interface {
	Create(ctx context.Context, product *domain.Product) error
	CreateMany(ctx context.Context, products []*domain.Product) error
	GetByID(ctx context.Context, id int64) (*domain.Product, error)
	GetAll(ctx context.Context) ([]*domain.Product, error)
	Search(ctx context.Context, query string) ([]*domain.Product, error)
	Update(ctx context.Context, product *domain.Product) error
	Delete(ctx context.Context, id int64) error
}
