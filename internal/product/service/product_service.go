package service

import (
	"context"
	"strings"

	"ecommerce/internal/product/domain"
	"ecommerce/internal/product/repository"
)

type ProductService interface {
	Create(ctx context.Context, product *domain.Product) error
	GetByID(ctx context.Context, id int64) (*domain.Product, error)
	GetAll(ctx context.Context) ([]*domain.Product, error)
	Search(ctx context.Context, query string) ([]*domain.Product, error)
	Update(ctx context.Context, product *domain.Product) error
	Delete(ctx context.Context, id int64) error
}

type productService struct {
	repository repository.ProductRepository
}

func NewProductService(repository repository.ProductRepository) ProductService {
	return &productService{repository: repository}
}

func (s *productService) Create(ctx context.Context, product *domain.Product) error {
	if err := ValidateProduct(product); err != nil {
		return err
	}

	return s.repository.Create(ctx, product)
}

func (s *productService) GetByID(ctx context.Context, id int64) (*domain.Product, error) {
	if id <= 0 {
		return nil, ErrInvalidProductID
	}

	return s.repository.GetByID(ctx, id)
}

func (s *productService) GetAll(ctx context.Context) ([]*domain.Product, error) {
	return s.repository.GetAll(ctx)
}

func (s *productService) Search(
	ctx context.Context,
	query string,
) ([]*domain.Product, error) {
	query = strings.TrimSpace(query)

	if query == "" {
		return nil, ErrInvalidSearchQuery
	}

	return s.repository.Search(ctx, query)
}

func (s *productService) Update(ctx context.Context, product *domain.Product) error {
	if err := ValidateProduct(product); err != nil {
		return err
	}

	if product.ID <= 0 {
		return ErrInvalidProductID
	}

	return s.repository.Update(ctx, product)
}

func (s *productService) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrInvalidProductID
	}

	return s.repository.Delete(ctx, id)
}
