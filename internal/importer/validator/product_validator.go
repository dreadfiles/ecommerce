package validator

import (
	"errors"
	"fmt"
	"strings"

	"ecommerce/internal/importer/parser"
	"ecommerce/internal/product/service"
)

func ValidateProducts(products []parser.ParsedProduct) error {
	if len(products) == 0 {
		return errors.New("csv contains no products")
	}

	skus := make(map[string]struct{}, len(products))
	var validationErrors []string

	for _, parsedProduct := range products {
		if err := service.ValidateProduct(parsedProduct.Product); err != nil {
			validationErrors = append(
				validationErrors,
				fmt.Sprintf("row %d: %v", parsedProduct.Row, err),
			)
			continue
		}

		sku := strings.TrimSpace(parsedProduct.Product.SKU)

		if _, exists := skus[sku]; exists {
			validationErrors = append(
				validationErrors,
				fmt.Sprintf(
					"row %d: duplicate sku %q",
					parsedProduct.Row,
					sku,
				),
			)
			continue
		}

		skus[sku] = struct{}{}
	}

	if len(validationErrors) > 0 {
		return fmt.Errorf(
			"validation failed:\n%s",
			strings.Join(validationErrors, "\n"),
		)
	}

	return nil
}
