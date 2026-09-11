package parser

import (
	"errors"
	"strings"
	"testing"
)

const validCSV = `name,sku,description,category,price,stock,weight_kg
Laptop,LAP-001,Professional laptop,Computers,1500.99,10,1.250
Mouse,MOU-001,Wireless mouse,Accessories,25.50,20,0.150
`

type errorReader struct {
	data      []byte
	failAfter int
	readCount int
	err       error
}

func (r *errorReader) Read(p []byte) (int, error) {
	if r.readCount >= r.failAfter {
		return 0, r.err
	}

	r.readCount++

	n := copy(p, r.data)
	r.data = r.data[n:]

	if len(r.data) == 0 {
		return n, r.err
	}

	return n, nil
}

type oneByteErrorReader struct {
	data []byte
	err  error
}

func (r *oneByteErrorReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, r.err
	}

	p[0] = r.data[0]
	r.data = r.data[1:]

	return 1, nil
}

func TestParseProducts(t *testing.T) {
	tests := []struct {
		name         string
		csv          string
		wantProducts int
		wantError    bool
	}{
		{
			name:         "parses valid csv",
			csv:          validCSV,
			wantProducts: 2,
		},
		{
			name: "supports quoted fields",
			csv: `name,sku,description,category,price,stock,weight_kg
Laptop,LAP-001,"Laptop, professional",Computers,1500.99,10,1.250
`,
			wantProducts: 1,
		},
		{
			name: "ignores empty records",
			csv: `name,sku,description,category,price,stock,weight_kg
Laptop,LAP-001,Professional laptop,Computers,1500.99,10,1.250
,,,,,,
`,
			wantProducts: 1,
		},
		{
			name: "rejects invalid header",
			csv: `name,sku,wrong,category,price,stock,weight_kg
Laptop,LAP-001,Description,Computers,1500.99,10,1.250
`,
			wantError: true,
		},
		{
			name: "rejects invalid stock",
			csv: `name,sku,description,category,price,stock,weight_kg
Laptop,LAP-001,Description,Computers,1500.99,abc,1.250
`,
			wantError: true,
		},
		{
			name: "rejects wrong number of columns",
			csv: `name,sku,description,category,price,stock,weight_kg
Laptop,LAP-001,Description,Computers,1500.99,10
`,
			wantError: true,
		},
		{
			name: "accepts bom in first header field",
			csv: "\uFEFFname,sku,description,category,price,stock,weight_kg\n" +
				"Laptop,LAP-001,Description,Computers,1500.99,10,1.250\n",
			wantProducts: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			products, err := ParseProducts(
				strings.NewReader(tt.csv),
			)

			if tt.wantError {
				if err == nil {
					t.Fatal("expected error")
				}

				return
			}

			if err != nil {
				t.Fatalf(
					"expected no error, got %v",
					err,
				)
			}

			if len(products) != tt.wantProducts {
				t.Fatalf(
					"expected %d products, got %d",
					tt.wantProducts,
					len(products),
				)
			}
		})
	}
}

func TestParseProducts_MapsFields(t *testing.T) {
	products, err := ParseProducts(
		strings.NewReader(validCSV),
	)

	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	product := products[0].Product

	if product.Name != "Laptop" {
		t.Errorf("unexpected name: %q", product.Name)
	}

	if product.SKU != "LAP-001" {
		t.Errorf("unexpected sku: %q", product.SKU)
	}

	if product.Description != "Professional laptop" {
		t.Errorf(
			"unexpected description: %q",
			product.Description,
		)
	}

	if product.Category != "Computers" {
		t.Errorf(
			"unexpected category: %q",
			product.Category,
		)
	}

	if product.Price != "1500.99" {
		t.Errorf(
			"unexpected price: %q",
			product.Price,
		)
	}

	if product.Stock != 10 {
		t.Errorf(
			"unexpected stock: %d",
			product.Stock,
		)
	}

	if product.WeightKg != "1.250" {
		t.Errorf(
			"unexpected weight: %q",
			product.WeightKg,
		)
	}
}

func TestParseProducts_RowNumbers(t *testing.T) {
	products, err := ParseProducts(
		strings.NewReader(validCSV),
	)

	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if products[0].Row != 2 {
		t.Errorf(
			"expected first product row=2, got %d",
			products[0].Row,
		)
	}

	if products[1].Row != 3 {
		t.Errorf(
			"expected second product row=3, got %d",
			products[1].Row,
		)
	}
}

func TestParseProducts_HeaderReadError(t *testing.T) {
	reader := &errorReader{
		data:      []byte{},
		failAfter: 0,
		err:       errors.New("read error"),
	}

	_, err := ParseProducts(reader)

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "read csv header") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseProducts_RecordReadError(t *testing.T) {
	reader := &oneByteErrorReader{
		data: []byte(
			"name,sku,description,category,price,stock,weight_kg\n",
		),
		err: errors.New("read error"),
	}

	_, err := ParseProducts(reader)

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "read csv line") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateHeader_InvalidColumnCount(t *testing.T) {
	header := []string{
		"name",
		"sku",
		"description",
		"category",
		"price",
		"stock",
	}

	err := validateHeader(header)

	if err == nil {
		t.Fatal("expected error")
	}

	expectedMessage := "invalid csv header: expected 7 columns, got 6"

	if err.Error() != expectedMessage {
		t.Fatalf(
			"expected error %q, got %q",
			expectedMessage,
			err.Error(),
		)
	}
}
