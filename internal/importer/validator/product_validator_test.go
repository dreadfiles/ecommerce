package validator

import (
	"strings"
	"testing"

	"ecommerce/internal/importer/parser"
	"ecommerce/internal/product/domain"
)

func parsedProduct(
	row int,
	sku string,
) parser.ParsedProduct {
	return parser.ParsedProduct{
		Row: row,
		Product: &domain.Product{
			Name:        "Product",
			SKU:         sku,
			Description: "Description",
			Category:    "Category",
			Price:       "10.00",
			Stock:       10,
			WeightKg:    "1.000",
		},
	}
}

func TestValidateProducts(t *testing.T) {
	tests := []struct {
		name       string
		products   []parser.ParsedProduct
		wantError  bool
		errorParts []string
	}{
		{
			name: "accepts valid products",
			products: []parser.ParsedProduct{
				parsedProduct(2, "SKU-001"),
				parsedProduct(3, "SKU-002"),
			},
		},
		{
			name:      "rejects empty product list",
			products:  nil,
			wantError: true,
			errorParts: []string{
				"csv contains no products",
			},
		},
		{
			name: "rejects duplicate sku",
			products: []parser.ParsedProduct{
				parsedProduct(2, "SKU-001"),
				parsedProduct(3, "SKU-001"),
			},
			wantError: true,
			errorParts: []string{
				"row 3",
				"duplicate sku",
				"SKU-001",
			},
		},
		{
			name: "reports validation row",
			products: []parser.ParsedProduct{
				parsedProduct(2, "SKU-001"),
				{
					Row: 3,
					Product: &domain.Product{
						SKU:         "SKU-002",
						Description: "Description",
						Category:    "Category",
						Price:       "10.00",
						Stock:       10,
						WeightKg:    "1.000",
					},
				},
			},
			wantError: true,
			errorParts: []string{
				"row 3",
				"product name is required",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProducts(tt.products)

			if !tt.wantError {
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

			for _, part := range tt.errorParts {
				if !strings.Contains(
					err.Error(),
					part,
				) {
					t.Errorf(
						"expected error containing %q, got %q",
						part,
						err.Error(),
					)
				}
			}
		})
	}
}

func TestValidateProducts_DuplicateDetectionIsCaseSensitive(t *testing.T) {
	products := []parser.ParsedProduct{
		parsedProduct(2, "SKU-001"),
		parsedProduct(3, "sku-001"),
	}

	if err := ValidateProducts(products); err != nil {
		t.Fatalf(
			"expected different SKUs to be accepted, got %v",
			err,
		)
	}
}
