package service

import (
	"strings"
	"testing"

	"ecommerce/internal/product/domain"
)

func validProduct() *domain.Product {
	return &domain.Product{
		Name:        "Laptop",
		SKU:         "LAP-001",
		Description: "Professional laptop",
		Category:    "Computers",
		Price:       "1500.99",
		Stock:       10,
		WeightKg:    "1.250",
	}
}

func TestValidateProduct(t *testing.T) {
	tests := []struct {
		name        string
		product     *domain.Product
		wantErr     bool
		errorString string
	}{
		{
			name:    "accepts valid product",
			product: validProduct(),
		},
		{
			name:        "rejects nil product",
			product:     nil,
			wantErr:     true,
			errorString: ErrInvalidProduct.Error(),
		},
		{
			name: "rejects empty name",
			product: func() *domain.Product {
				p := validProduct()
				p.Name = ""
				return p
			}(),
			wantErr:     true,
			errorString: "product name is required",
		},
		{
			name: "rejects whitespace name",
			product: func() *domain.Product {
				p := validProduct()
				p.Name = "   "
				return p
			}(),
			wantErr:     true,
			errorString: "product name is required",
		},
		{
			name: "rejects empty sku",
			product: func() *domain.Product {
				p := validProduct()
				p.SKU = ""
				return p
			}(),
			wantErr:     true,
			errorString: "product sku is required",
		},
		{
			name: "rejects empty description",
			product: func() *domain.Product {
				p := validProduct()
				p.Description = ""
				return p
			}(),
			wantErr:     true,
			errorString: "product description is required",
		},
		{
			name: "rejects empty category",
			product: func() *domain.Product {
				p := validProduct()
				p.Category = ""
				return p
			}(),
			wantErr:     true,
			errorString: "product category is required",
		},
		{
			name: "rejects empty price",
			product: func() *domain.Product {
				p := validProduct()
				p.Price = ""
				return p
			}(),
			wantErr:     true,
			errorString: "product price is required",
		},
		{
			name: "rejects negative stock",
			product: func() *domain.Product {
				p := validProduct()
				p.Stock = -1
				return p
			}(),
			wantErr:     true,
			errorString: "product stock cannot be negative",
		},
		{
			name: "rejects empty weight",
			product: func() *domain.Product {
				p := validProduct()
				p.WeightKg = ""
				return p
			}(),
			wantErr:     true,
			errorString: "product weight is required",
		},
		{
			name: "rejects invalid price",
			product: func() *domain.Product {
				p := validProduct()
				p.Price = "10.123"
				return p
			}(),
			wantErr:     true,
			errorString: "invalid product price",
		},
		{
			name: "rejects invalid weight",
			product: func() *domain.Product {
				p := validProduct()
				p.WeightKg = "1.1234"
				return p
			}(),
			wantErr:     true,
			errorString: "invalid product weight",
		},
		{
			name: "accepts zero stock",
			product: func() *domain.Product {
				p := validProduct()
				p.Stock = 0
				return p
			}(),
		},
		{
			name: "accepts zero price",
			product: func() *domain.Product {
				p := validProduct()
				p.Price = "0.00"
				return p
			}(),
		},
		{
			name: "accepts minimum weight precision",
			product: func() *domain.Product {
				p := validProduct()
				p.WeightKg = "0.001"
				return p
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProduct(tt.product)

			if !tt.wantErr {
				if err != nil {
					t.Fatalf(
						"expected no error, got %v",
						err,
					)
				}

				return
			}

			if err == nil {
				t.Fatal("expected validation error")
			}

			if !strings.Contains(
				err.Error(),
				tt.errorString,
			) {
				t.Errorf(
					"expected error containing %q, got %q",
					tt.errorString,
					err.Error(),
				)
			}
		})
	}
}

func TestValidateProduct_DecimalRules(t *testing.T) {
	tests := []struct {
		name    string
		price   string
		weight  string
		wantErr bool
	}{
		{
			name:   "accepts two decimal places for price",
			price:  "99.99",
			weight: "1.001",
		},
		{
			name:    "rejects three decimal places for price",
			price:   "99.999",
			weight:  "1.001",
			wantErr: true,
		},
		{
			name:   "accepts three decimal places for weight",
			price:  "99.99",
			weight: "1.999",
		},
		{
			name:    "rejects four decimal places for weight",
			price:   "99.99",
			weight:  "1.9999",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			product := validProduct()
			product.Price = tt.price
			product.WeightKg = tt.weight

			err := ValidateProduct(product)

			if tt.wantErr && err == nil {
				t.Fatal("expected validation error")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf(
					"expected no error, got %v",
					err,
				)
			}
		})
	}
}
