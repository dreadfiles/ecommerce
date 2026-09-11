package service

import (
	"context"
	"errors"
	"testing"

	"ecommerce/internal/product/domain"
	"ecommerce/internal/product/repository"
)

type mockProductRepository struct {
	createFunc     func(context.Context, *domain.Product) error
	createManyFunc func(context.Context, []*domain.Product) error
	getByIDFunc    func(context.Context, int64) (*domain.Product, error)
	getAllFunc     func(context.Context) ([]*domain.Product, error)
	searchFunc     func(context.Context, string) ([]*domain.Product, error)
	updateFunc     func(context.Context, *domain.Product) error
	deleteFunc     func(context.Context, int64) error
}

func (m *mockProductRepository) Create(
	ctx context.Context,
	product *domain.Product,
) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, product)
	}

	return nil
}

func (m *mockProductRepository) CreateMany(
	ctx context.Context,
	products []*domain.Product,
) error {
	if m.createManyFunc != nil {
		return m.createManyFunc(ctx, products)
	}

	return nil
}

func (m *mockProductRepository) GetByID(
	ctx context.Context,
	id int64,
) (*domain.Product, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}

	return nil, nil
}

func (m *mockProductRepository) GetAll(
	ctx context.Context,
) ([]*domain.Product, error) {
	if m.getAllFunc != nil {
		return m.getAllFunc(ctx)
	}

	return nil, nil
}

func (m *mockProductRepository) Search(
	ctx context.Context,
	query string,
) ([]*domain.Product, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, query)
	}

	return nil, nil
}

func (m *mockProductRepository) Update(
	ctx context.Context,
	product *domain.Product,
) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, product)
	}

	return nil
}

func (m *mockProductRepository) Delete(
	ctx context.Context,
	id int64,
) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}

	return nil
}

func productForServiceTest() *domain.Product {
	return &domain.Product{
		ID:          1,
		Name:        "Laptop",
		SKU:         "LAP-001",
		Description: "Gaming laptop",
		Category:    "Computers",
		Price:       "2500.00",
		Stock:       10,
		WeightKg:    "2.500",
	}
}

func TestNewProductService(t *testing.T) {
	repositoryMock := &mockProductRepository{}

	service := NewProductService(repositoryMock)

	if service == nil {
		t.Fatal("expected service, got nil")
	}

	implementation, ok := service.(*productService)
	if !ok {
		t.Fatalf(
			"expected *productService, got %T",
			service,
		)
	}

	if implementation.repository != repositoryMock {
		t.Fatal("expected repository to be assigned")
	}
}

func TestProductService_Create(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		product      *domain.Product
		repositoryFn func(context.Context, *domain.Product) error
		expectedErr  error
	}{
		{
			name:    "creates product successfully",
			product: productForServiceTest(),
		},
		{
			name:        "rejects nil product",
			product:     nil,
			expectedErr: ErrInvalidProduct,
		},
		{
			name: "rejects invalid product",
			product: &domain.Product{
				Name:        "",
				SKU:         "LAP-001",
				Description: "Gaming laptop",
				Category:    "Computers",
				Price:       "2500.00",
				Stock:       10,
				WeightKg:    "2.500",
			},
			expectedErr: errors.New("product name is required"),
		},
		{
			name:    "returns repository error",
			product: productForServiceTest(),
			repositoryFn: func(
				_ context.Context,
				_ *domain.Product,
			) error {
				return repository.ErrConflict
			},
			expectedErr: repository.ErrConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repositoryMock := &mockProductRepository{
				createFunc: tt.repositoryFn,
			}

			service := NewProductService(repositoryMock)

			err := service.Create(
				ctx,
				tt.product,
			)

			if tt.expectedErr == nil {
				if err != nil {
					t.Fatalf(
						"expected no error, got %v",
						err,
					)
				}

				return
			}

			if err == nil {
				t.Fatalf(
					"expected error %v, got nil",
					tt.expectedErr,
				)
			}

			if !errors.Is(err, tt.expectedErr) &&
				err.Error() != tt.expectedErr.Error() {
				t.Fatalf(
					"expected error %v, got %v",
					tt.expectedErr,
					err,
				)
			}
		})
	}
}

func TestProductService_GetByID(t *testing.T) {
	ctx := context.Background()
	expectedProduct := productForServiceTest()

	tests := []struct {
		name         string
		id           int64
		repositoryFn func(context.Context, int64) (*domain.Product, error)
		expectedErr  error
	}{
		{
			name: "returns product successfully",
			id:   1,
			repositoryFn: func(
				_ context.Context,
				id int64,
			) (*domain.Product, error) {
				if id != 1 {
					return nil, errors.New("unexpected id")
				}

				return expectedProduct, nil
			},
		},
		{
			name:        "rejects zero id",
			id:          0,
			expectedErr: ErrInvalidProductID,
		},
		{
			name:        "rejects negative id",
			id:          -1,
			expectedErr: ErrInvalidProductID,
		},
		{
			name: "returns repository not found",
			id:   999,
			repositoryFn: func(
				_ context.Context,
				_ int64,
			) (*domain.Product, error) {
				return nil, repository.ErrNotFound
			},
			expectedErr: repository.ErrNotFound,
		},
		{
			name: "returns repository error",
			id:   2,
			repositoryFn: func(
				_ context.Context,
				_ int64,
			) (*domain.Product, error) {
				return nil, errors.New("database error")
			},
			expectedErr: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repositoryMock := &mockProductRepository{
				getByIDFunc: tt.repositoryFn,
			}

			service := NewProductService(repositoryMock)

			result, err := service.GetByID(
				ctx,
				tt.id,
			)

			if tt.expectedErr != nil {
				if err == nil {
					t.Fatalf(
						"expected error %v, got nil",
						tt.expectedErr,
					)
				}

				if !errors.Is(err, tt.expectedErr) &&
					err.Error() != tt.expectedErr.Error() {
					t.Fatalf(
						"expected error %v, got %v",
						tt.expectedErr,
						err,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf(
					"expected no error, got %v",
					err,
				)
			}

			if result == nil {
				t.Fatal("expected product, got nil")
			}

			if result.ID != expectedProduct.ID {
				t.Fatalf(
					"expected product id %d, got %d",
					expectedProduct.ID,
					result.ID,
				)
			}
		})
	}
}

func TestProductService_GetAll(t *testing.T) {
	ctx := context.Background()

	expectedProducts := []*domain.Product{
		productForServiceTest(),
		productForServiceTest(),
	}

	expectedProducts[0].ID = 1
	expectedProducts[1].ID = 2

	tests := []struct {
		name         string
		repositoryFn func(context.Context) ([]*domain.Product, error)
		expectedErr  error
		expectedLen  int
	}{
		{
			name: "returns products successfully",
			repositoryFn: func(
				_ context.Context,
			) ([]*domain.Product, error) {
				return expectedProducts, nil
			},
			expectedLen: 2,
		},
		{
			name: "returns empty list",
			repositoryFn: func(
				_ context.Context,
			) ([]*domain.Product, error) {
				return []*domain.Product{}, nil
			},
			expectedLen: 0,
		},
		{
			name: "returns nil list",
			repositoryFn: func(
				_ context.Context,
			) ([]*domain.Product, error) {
				return nil, nil
			},
			expectedLen: 0,
		},
		{
			name: "returns repository error",
			repositoryFn: func(
				_ context.Context,
			) ([]*domain.Product, error) {
				return nil, errors.New("database error")
			},
			expectedErr: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repositoryMock := &mockProductRepository{
				getAllFunc: tt.repositoryFn,
			}

			service := NewProductService(repositoryMock)

			result, err := service.GetAll(ctx)

			if tt.expectedErr != nil {
				if err == nil {
					t.Fatalf(
						"expected error %v, got nil",
						tt.expectedErr,
					)
				}

				if !errors.Is(err, tt.expectedErr) &&
					err.Error() != tt.expectedErr.Error() {
					t.Fatalf(
						"expected error %v, got %v",
						tt.expectedErr,
						err,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf(
					"expected no error, got %v",
					err,
				)
			}

			if len(result) != tt.expectedLen {
				t.Fatalf(
					"expected %d products, got %d",
					tt.expectedLen,
					len(result),
				)
			}
		})
	}
}

func TestProductService_Search(t *testing.T) {
	ctx := context.Background()

	expectedProducts := []*domain.Product{
		productForServiceTest(),
	}

	tests := []struct {
		name         string
		query        string
		repositoryFn func(context.Context, string) ([]*domain.Product, error)
		expectedErr  error
		expectedLen  int
	}{
		{
			name:  "returns matching products",
			query: "laptop",
			repositoryFn: func(
				_ context.Context,
				query string,
			) ([]*domain.Product, error) {
				if query != "laptop" {
					return nil, errors.New("unexpected query")
				}

				return expectedProducts, nil
			},
			expectedLen: 1,
		},
		{
			name:  "trims query",
			query: "  laptop  ",
			repositoryFn: func(
				_ context.Context,
				query string,
			) ([]*domain.Product, error) {
				if query != "laptop" {
					return nil, errors.New("query was not trimmed")
				}

				return expectedProducts, nil
			},
			expectedLen: 1,
		},
		{
			name:        "rejects empty query",
			query:       "",
			expectedErr: ErrInvalidSearchQuery,
		},
		{
			name:        "rejects whitespace query",
			query:       "   ",
			expectedErr: ErrInvalidSearchQuery,
		},
		{
			name:  "returns repository error",
			query: "laptop",
			repositoryFn: func(
				_ context.Context,
				_ string,
			) ([]*domain.Product, error) {
				return nil, errors.New("database error")
			},
			expectedErr: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repositoryMock := &mockProductRepository{
				searchFunc: tt.repositoryFn,
			}

			service := NewProductService(repositoryMock)

			result, err := service.Search(
				ctx,
				tt.query,
			)

			if tt.expectedErr != nil {
				if err == nil {
					t.Fatalf(
						"expected error %v, got nil",
						tt.expectedErr,
					)
				}

				if !errors.Is(err, tt.expectedErr) &&
					err.Error() != tt.expectedErr.Error() {
					t.Fatalf(
						"expected error %v, got %v",
						tt.expectedErr,
						err,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf(
					"expected no error, got %v",
					err,
				)
			}

			if len(result) != tt.expectedLen {
				t.Fatalf(
					"expected %d products, got %d",
					tt.expectedLen,
					len(result),
				)
			}
		})
	}
}

func TestProductService_Update(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		product      *domain.Product
		repositoryFn func(context.Context, *domain.Product) error
		expectedErr  error
	}{
		{
			name:    "updates product successfully",
			product: productForServiceTest(),
		},
		{
			name:        "rejects nil product",
			product:     nil,
			expectedErr: ErrInvalidProduct,
		},
		{
			name: "rejects invalid product before id validation",
			product: &domain.Product{
				ID:          1,
				Name:        "",
				SKU:         "LAP-001",
				Description: "Gaming laptop",
				Category:    "Computers",
				Price:       "2500.00",
				Stock:       10,
				WeightKg:    "2.500",
			},
			expectedErr: errors.New("product name is required"),
		},
		{
			name: "rejects zero id",
			product: &domain.Product{
				ID:          0,
				Name:        "Laptop",
				SKU:         "LAP-001",
				Description: "Gaming laptop",
				Category:    "Computers",
				Price:       "2500.00",
				Stock:       10,
				WeightKg:    "2.500",
			},
			expectedErr: ErrInvalidProductID,
		},
		{
			name: "rejects negative id",
			product: &domain.Product{
				ID:          -1,
				Name:        "Laptop",
				SKU:         "LAP-001",
				Description: "Gaming laptop",
				Category:    "Computers",
				Price:       "2500.00",
				Stock:       10,
				WeightKg:    "2.500",
			},
			expectedErr: ErrInvalidProductID,
		},
		{
			name:    "returns not found",
			product: productForServiceTest(),
			repositoryFn: func(
				_ context.Context,
				_ *domain.Product,
			) error {
				return repository.ErrNotFound
			},
			expectedErr: repository.ErrNotFound,
		},
		{
			name:    "returns conflict",
			product: productForServiceTest(),
			repositoryFn: func(
				_ context.Context,
				_ *domain.Product,
			) error {
				return repository.ErrConflict
			},
			expectedErr: repository.ErrConflict,
		},
		{
			name:    "returns repository error",
			product: productForServiceTest(),
			repositoryFn: func(
				_ context.Context,
				_ *domain.Product,
			) error {
				return errors.New("database error")
			},
			expectedErr: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repositoryMock := &mockProductRepository{
				updateFunc: tt.repositoryFn,
			}

			service := NewProductService(repositoryMock)

			err := service.Update(
				ctx,
				tt.product,
			)

			if tt.expectedErr == nil {
				if err != nil {
					t.Fatalf(
						"expected no error, got %v",
						err,
					)
				}

				return
			}

			if err == nil {
				t.Fatalf(
					"expected error %v, got nil",
					tt.expectedErr,
				)
			}

			if !errors.Is(err, tt.expectedErr) &&
				err.Error() != tt.expectedErr.Error() {
				t.Fatalf(
					"expected error %v, got %v",
					tt.expectedErr,
					err,
				)
			}
		})
	}
}

func TestProductService_Delete(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name         string
		id           int64
		repositoryFn func(context.Context, int64) error
		expectedErr  error
	}{
		{
			name: "deletes product successfully",
			id:   1,
		},
		{
			name:        "rejects zero id",
			id:          0,
			expectedErr: ErrInvalidProductID,
		},
		{
			name:        "rejects negative id",
			id:          -1,
			expectedErr: ErrInvalidProductID,
		},
		{
			name: "returns not found",
			id:   999,
			repositoryFn: func(
				_ context.Context,
				_ int64,
			) error {
				return repository.ErrNotFound
			},
			expectedErr: repository.ErrNotFound,
		},
		{
			name: "returns conflict",
			id:   2,
			repositoryFn: func(
				_ context.Context,
				_ int64,
			) error {
				return repository.ErrConflict
			},
			expectedErr: repository.ErrConflict,
		},
		{
			name: "returns repository error",
			id:   3,
			repositoryFn: func(
				_ context.Context,
				_ int64,
			) error {
				return errors.New("database error")
			},
			expectedErr: errors.New("database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repositoryMock := &mockProductRepository{
				deleteFunc: tt.repositoryFn,
			}

			service := NewProductService(repositoryMock)

			err := service.Delete(
				ctx,
				tt.id,
			)

			if tt.expectedErr == nil {
				if err != nil {
					t.Fatalf(
						"expected no error, got %v",
						err,
					)
				}

				return
			}

			if err == nil {
				t.Fatalf(
					"expected error %v, got nil",
					tt.expectedErr,
				)
			}

			if !errors.Is(err, tt.expectedErr) &&
				err.Error() != tt.expectedErr.Error() {
				t.Fatalf(
					"expected error %v, got %v",
					tt.expectedErr,
					err,
				)
			}
		})
	}
}
