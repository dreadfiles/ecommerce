package service

import (
	"context"
	"fmt"
	"io"

	"ecommerce/internal/importer/parser"
	"ecommerce/internal/importer/validator"
	"ecommerce/internal/product/domain"
	"ecommerce/internal/product/repository"
)

type ProductImportService interface {
	Import(ctx context.Context, reader io.Reader) error
}

type productImportService struct {
	productRepository repository.ProductRepository
}

func NewProductImportService(
	productRepository repository.ProductRepository,
) ProductImportService {
	return &productImportService{
		productRepository: productRepository,
	}
}

func (s *productImportService) Import(
	ctx context.Context,
	reader io.Reader,
) error {
	if reader == nil {
		return fmt.Errorf("csv reader cannot be nil")
	}

	parsedProducts, err := parser.ParseProducts(reader)
	if err != nil {
		return fmt.Errorf("parse products: %w", err)
	}

	if err := validator.ValidateProducts(parsedProducts); err != nil {
		return fmt.Errorf("validate products: %w", err)
	}

	products := make([]*domain.Product, 0, len(parsedProducts))

	for _, parsedProduct := range parsedProducts {
		products = append(products, parsedProduct.Product)
	}

	if err := s.productRepository.CreateMany(ctx, products); err != nil {
		return fmt.Errorf("persist imported products: %w", err)
	}

	return nil
}
