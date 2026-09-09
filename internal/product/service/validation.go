package service

import (
	"errors"
	"fmt"
	"strings"

	"ecommerce/internal/product/domain"
)

func ValidateProduct(product *domain.Product) error {
	if product == nil {
		return ErrInvalidProduct
	}

	if strings.TrimSpace(product.Name) == "" {
		return errors.New("product name is required")
	}

	if strings.TrimSpace(product.SKU) == "" {
		return errors.New("product sku is required")
	}

	if strings.TrimSpace(product.Description) == "" {
		return errors.New("product description is required")
	}

	if strings.TrimSpace(product.Category) == "" {
		return errors.New("product category is required")
	}

	if strings.TrimSpace(product.Price) == "" {
		return errors.New("product price is required")
	}

	if product.Stock < 0 {
		return errors.New("product stock cannot be negative")
	}

	if strings.TrimSpace(product.WeightKg) == "" {
		return errors.New("product weight is required")
	}

	return validateDecimalFields(product)
}

func validateDecimalFields(product *domain.Product) error {
	if _, err := parseDecimal(product.Price, 2); err != nil {
		return fmt.Errorf("invalid product price: %w", err)
	}

	if _, err := parseDecimal(product.WeightKg, 3); err != nil {
		return fmt.Errorf("invalid product weight: %w", err)
	}

	return nil
}
