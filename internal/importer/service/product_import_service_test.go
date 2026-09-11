package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"ecommerce/internal/product/domain"
)

type mockProductRepository struct {
	createManyFunc func(
		context.Context,
		[]*domain.Product,
	) error

	createManyCalled bool
}

func (m *mockProductRepository) Create(
	_ context.Context,
	_ *domain.Product,
) error {
	return errors.New("not expected")
}

func (m *mockProductRepository) CreateMany(
	ctx context.Context,
	products []*domain.Product,
) error {
	m.createManyCalled = true

	if m.createManyFunc == nil {
		return errors.New("create many mock not configured")
	}

	return m.createManyFunc(ctx, products)
}

func (m *mockProductRepository) GetByID(
	_ context.Context,
	_ int64,
) (*domain.Product, error) {
	return nil, errors.New("not expected")
}

func (m *mockProductRepository) GetAll(
	_ context.Context,
) ([]*domain.Product, error) {
	return nil, errors.New("not expected")
}

func (m *mockProductRepository) Search(
	_ context.Context,
	_ string,
) ([]*domain.Product, error) {
	return nil, errors.New("not expected")
}

func (m *mockProductRepository) Update(
	_ context.Context,
	_ *domain.Product,
) error {
	return errors.New("not expected")
}

func (m *mockProductRepository) Delete(
	_ context.Context,
	_ int64,
) error {
	return errors.New("not expected")
}

func TestProductImportService_Import(t *testing.T) {
	tests := []struct {
		name            string
		csv             string
		repositoryError error
		wantError       bool
		errorParts      []string
		wantCreateMany  bool
	}{
		{
			name: "imports valid csv",
			csv: `name,sku,description,category,price,stock,weight_kg
Laptop,LAP-001,Professional laptop,Computers,1500.99,10,1.250
Mouse,MOU-001,Wireless mouse,Accessories,25.50,20,0.150
`,
			wantCreateMany: true,
		},
		{
			name: "rejects malformed csv",
			csv: `name,sku,description,category,price,stock,weight_kg
Laptop,LAP-001,Description,Computers,10.00,10
`,
			wantError: true,
			errorParts: []string{
				"parse products",
			},
			wantCreateMany: false,
		},
		{
			name: "rejects invalid product",
			csv: `name,sku,description,category,price,stock,weight_kg
,LAP-001,Description,Computers,10.00,10,1.250
`,
			wantError: true,
			errorParts: []string{
				"validate products",
				"product name is required",
			},
			wantCreateMany: false,
		},
		{
			name: "propagates repository error",
			csv: `name,sku,description,category,price,stock,weight_kg
Laptop,LAP-001,Description,Computers,10.00,10,1.250
`,
			repositoryError: errors.New("database failure"),
			wantError:       true,
			errorParts: []string{
				"persist imported products",
				"database failure",
			},
			wantCreateMany: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repositoryMock := &mockProductRepository{
				createManyFunc: func(
					_ context.Context,
					_ []*domain.Product,
				) error {
					return tt.repositoryError
				},
			}

			importService := NewProductImportService(
				repositoryMock,
			)

			err := importService.Import(
				context.Background(),
				strings.NewReader(tt.csv),
			)

			if !tt.wantError {
				if err != nil {
					t.Fatalf(
						"expected no error, got %v",
						err,
					)
				}
			} else {
				if err == nil {
					t.Fatal("expected error")
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
			}

			if repositoryMock.createManyCalled != tt.wantCreateMany {
				t.Errorf(
					"expected CreateMany called=%v, got %v",
					tt.wantCreateMany,
					repositoryMock.createManyCalled,
				)
			}
		})
	}
}

func TestProductImportService_Import_NilReader(t *testing.T) {
	repositoryMock := &mockProductRepository{}

	importService := NewProductImportService(
		repositoryMock,
	)

	err := importService.Import(
		context.Background(),
		nil,
	)

	if err == nil {
		t.Fatal("expected error for nil reader")
	}

	if !strings.Contains(
		err.Error(),
		"csv reader cannot be nil",
	) {
		t.Errorf(
			"unexpected error: %v",
			err,
		)
	}

	if repositoryMock.createManyCalled {
		t.Fatal("repository should not be called")
	}
}
