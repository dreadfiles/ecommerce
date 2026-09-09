package parser

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"ecommerce/internal/product/domain"
)

const (
	columnName = iota
	columnSKU
	columnDescription
	columnCategory
	columnPrice
	columnStock
	columnWeightKg

	expectedColumns = columnWeightKg + 1
)

var expectedHeader = [expectedColumns]string{
	"name",
	"sku",
	"description",
	"category",
	"price",
	"stock",
	"weight_kg",
}

type ParsedProduct struct {
	Product *domain.Product
	Row     int
}

func ParseProducts(reader io.Reader) ([]ParsedProduct, error) {
	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true
	csvReader.FieldsPerRecord = expectedColumns

	products := make([]ParsedProduct, 0)

	header, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("read csv header: %w", err)
	}

	if err := validateHeader(header); err != nil {
		return nil, err
	}

	for {
		record, err := csvReader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}

			return nil, fmt.Errorf(
				"read csv line %d: %w",
				csvReader.InputOffset(),
				err,
			)
		}

		row, _ := csvReader.FieldPos(0)

		if isEmptyRecord(record) {
			continue
		}

		product, err := parseRecord(record)
		if err != nil {
			return nil, fmt.Errorf(
				"parse csv line %d: %w",
				row,
				err,
			)
		}

		products = append(products, ParsedProduct{
			Product: product,
			Row:     row,
		})
	}

	return products, nil
}

func validateHeader(header []string) error {
	if len(header) != expectedColumns {
		return fmt.Errorf(
			"invalid csv header: expected %d columns, got %d",
			expectedColumns,
			len(header),
		)
	}

	for i, expectedColumn := range expectedHeader {
		actualColumn := strings.TrimSpace(header[i])

		if i == 0 {
			actualColumn = strings.TrimPrefix(actualColumn, "\uFEFF")
		}

		if actualColumn != expectedColumn {
			return fmt.Errorf(
				"invalid csv header: expected column %d to be %q, got %q",
				i+1,
				expectedColumn,
				header[i],
			)
		}
	}

	return nil
}

func parseRecord(record []string) (*domain.Product, error) {
	stock, err := strconv.Atoi(strings.TrimSpace(record[columnStock]))
	if err != nil {
		return nil, fmt.Errorf("invalid stock: %w", err)
	}

	return &domain.Product{
		Name:        strings.TrimSpace(record[columnName]),
		SKU:         strings.TrimSpace(record[columnSKU]),
		Description: strings.TrimSpace(record[columnDescription]),
		Category:    strings.TrimSpace(record[columnCategory]),
		Price:       strings.TrimSpace(record[columnPrice]),
		Stock:       stock,
		WeightKg:    strings.TrimSpace(record[columnWeightKg]),
	}, nil
}

func isEmptyRecord(record []string) bool {
	for _, field := range record {
		if strings.TrimSpace(field) != "" {
			return false
		}
	}

	return true
}
