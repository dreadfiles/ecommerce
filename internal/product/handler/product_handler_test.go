package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ecommerce/internal/product/domain"
	"ecommerce/internal/product/repository"
	"ecommerce/internal/product/service"
)

type mockProductService struct {
	createFunc  func(context.Context, *domain.Product) error
	getAllFunc  func(context.Context) ([]*domain.Product, error)
	getByIDFunc func(context.Context, int64) (*domain.Product, error)
	searchFunc  func(context.Context, string) ([]*domain.Product, error)
	updateFunc  func(context.Context, *domain.Product) error
	deleteFunc  func(context.Context, int64) error
}

func (m *mockProductService) Create(
	ctx context.Context,
	product *domain.Product,
) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, product)
	}

	return nil
}

func (m *mockProductService) GetAll(
	ctx context.Context,
) ([]*domain.Product, error) {
	if m.getAllFunc != nil {
		return m.getAllFunc(ctx)
	}

	return nil, nil
}

func (m *mockProductService) GetByID(
	ctx context.Context,
	id int64,
) (*domain.Product, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}

	return nil, nil
}

func (m *mockProductService) Search(
	ctx context.Context,
	query string,
) ([]*domain.Product, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, query)
	}

	return nil, nil
}

func (m *mockProductService) Update(
	ctx context.Context,
	product *domain.Product,
) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, product)
	}

	return nil
}

func (m *mockProductService) Delete(
	ctx context.Context,
	id int64,
) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}

	return nil
}

func TestProductHandler_Create(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		serviceError   error
		expectedStatus int
		expectedError  string
	}{
		{
			name: "creates product successfully",
			body: `{
				"name": "Laptop",
				"sku": "LAP-001",
				"description": "Gaming laptop",
				"category": "Computers",
				"price": "2500.00",
				"stock": 10,
				"weight_kg": "2.500"
			}`,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "rejects invalid json",
			body:           `{"name":`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request body",
		},
		{
			name: "returns conflict when sku already exists",
			body: `{
				"name": "Laptop",
				"sku": "LAP-001",
				"description": "Gaming laptop",
				"category": "Computers",
				"price": "2500.00",
				"stock": 10,
				"weight_kg": "2.500"
			}`,
			serviceError:   repository.ErrConflict,
			expectedStatus: http.StatusConflict,
			expectedError:  repository.ErrConflict.Error(),
		},
		{
			name: "returns bad request for service validation error",
			body: `{
				"name": "Laptop",
				"sku": "LAP-001",
				"description": "Gaming laptop",
				"category": "Computers",
				"price": "2500.00",
				"stock": 10,
				"weight_kg": "2.500"
			}`,
			serviceError:   errors.New("product name is required"),
			expectedStatus: http.StatusBadRequest,
			expectedError:  "product name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockProductService{
				createFunc: func(
					_ context.Context,
					product *domain.Product,
				) error {
					return tt.serviceError
				},
			}

			handler := NewProductHandler(mock)

			request := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/products",
				strings.NewReader(tt.body),
			)

			recorder := httptest.NewRecorder()

			handler.Create(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d",
					tt.expectedStatus,
					recorder.Code,
				)
			}

			if tt.expectedError != "" {
				var response map[string]string

				if err := json.NewDecoder(
					recorder.Body,
				).Decode(&response); err != nil {
					t.Fatalf(
						"failed to decode error response: %v",
						err,
					)
				}

				if response["error"] != tt.expectedError {
					t.Fatalf(
						"expected error %q, got %q",
						tt.expectedError,
						response["error"],
					)
				}
			}
		})
	}
}

func TestProductHandler_GetAll(t *testing.T) {
	tests := []struct {
		name           string
		products       []*domain.Product
		serviceError   error
		expectedStatus int
		expectedCount  int
	}{
		{
			name: "returns products successfully",
			products: []*domain.Product{
				{
					ID:          1,
					Name:        "Laptop",
					SKU:         "LAP-001",
					Description: "Gaming laptop",
					Category:    "Computers",
					Price:       "2500.00",
					Stock:       10,
					WeightKg:    "2.500",
				},
				{
					ID:          2,
					Name:        "Mouse",
					SKU:         "MOU-001",
					Description: "Wireless mouse",
					Category:    "Accessories",
					Price:       "50.00",
					Stock:       20,
					WeightKg:    "0.200",
				},
			},
			expectedStatus: http.StatusOK,
			expectedCount:  2,
		},
		{
			name:           "returns empty list",
			products:       []*domain.Product{},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name:           "returns internal server error",
			serviceError:   errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockProductService{
				getAllFunc: func(
					_ context.Context,
				) ([]*domain.Product, error) {
					return tt.products, tt.serviceError
				},
			}

			handler := NewProductHandler(mock)

			request := httptest.NewRequest(
				http.MethodGet,
				"/api/v1/products",
				nil,
			)

			recorder := httptest.NewRecorder()

			handler.GetAll(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d",
					tt.expectedStatus,
					recorder.Code,
				)
			}

			if tt.serviceError != nil {
				return
			}

			var response []map[string]any

			if err := json.NewDecoder(
				recorder.Body,
			).Decode(&response); err != nil {
				t.Fatalf(
					"failed to decode response: %v",
					err,
				)
			}

			if len(response) != tt.expectedCount {
				t.Fatalf(
					"expected %d products, got %d",
					tt.expectedCount,
					len(response),
				)
			}
		})
	}
}

func TestProductHandler_GetByID(t *testing.T) {
	product := &domain.Product{
		ID:          1,
		Name:        "Laptop",
		SKU:         "LAP-001",
		Description: "Gaming laptop",
		Category:    "Computers",
		Price:       "2500.00",
		Stock:       10,
		WeightKg:    "2.500",
	}

	tests := []struct {
		name           string
		id             string
		serviceProduct *domain.Product
		serviceError   error
		expectedStatus int
	}{
		{
			name:           "returns product successfully",
			id:             "1",
			serviceProduct: product,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "rejects invalid id",
			id:             "abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "rejects zero id",
			id:             "0",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "returns not found",
			id:             "999",
			serviceError:   repository.ErrNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "returns internal server error",
			id:             "1",
			serviceError:   errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockProductService{
				getByIDFunc: func(
					_ context.Context,
					_ int64,
				) (*domain.Product, error) {
					return tt.serviceProduct, tt.serviceError
				},
			}

			handler := NewProductHandler(mock)

			request := httptest.NewRequest(
				http.MethodGet,
				"/api/v1/products/"+tt.id,
				nil,
			)

			request.SetPathValue("id", tt.id)

			recorder := httptest.NewRecorder()

			handler.GetByID(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d",
					tt.expectedStatus,
					recorder.Code,
				)
			}
		})
	}
}

func TestProductHandler_Update(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		body           string
		serviceError   error
		expectedStatus int
	}{
		{
			name: "updates product successfully",
			id:   "1",
			body: `{
				"name": "Updated Laptop",
				"sku": "LAP-001",
				"description": "Updated description",
				"category": "Computers",
				"price": "3000.00",
				"stock": 15,
				"weight_kg": "2.800"
			}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "rejects invalid id",
			id:             "abc",
			body:           `{}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "rejects invalid json",
			id:             "1",
			body:           `{"name":`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "returns not found",
			id:   "999",
			body: `{
				"name": "Laptop",
				"sku": "LAP-001",
				"description": "Gaming laptop",
				"category": "Computers",
				"price": "2500.00",
				"stock": 10,
				"weight_kg": "2.500"
			}`,
			serviceError:   repository.ErrNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "returns conflict",
			id:   "1",
			body: `{
				"name": "Laptop",
				"sku": "LAP-002",
				"description": "Gaming laptop",
				"category": "Computers",
				"price": "2500.00",
				"stock": 10,
				"weight_kg": "2.500"
			}`,
			serviceError:   repository.ErrConflict,
			expectedStatus: http.StatusConflict,
		},
		{
			name: "returns bad request for validation error",
			id:   "1",
			body: `{
				"name": "Laptop",
				"sku": "LAP-001",
				"description": "Gaming laptop",
				"category": "Computers",
				"price": "2500.00",
				"stock": 10,
				"weight_kg": "2.500"
			}`,
			serviceError:   errors.New("invalid product"),
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockProductService{
				updateFunc: func(
					_ context.Context,
					_ *domain.Product,
				) error {
					return tt.serviceError
				},
			}

			handler := NewProductHandler(mock)

			request := httptest.NewRequest(
				http.MethodPut,
				"/api/v1/products/"+tt.id,
				strings.NewReader(tt.body),
			)

			request.SetPathValue("id", tt.id)

			recorder := httptest.NewRecorder()

			handler.Update(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d",
					tt.expectedStatus,
					recorder.Code,
				)
			}
		})
	}
}

func TestProductHandler_Delete(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		serviceError   error
		expectedStatus int
	}{
		{
			name:           "deletes product successfully",
			id:             "1",
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "rejects invalid id",
			id:             "abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "rejects zero id",
			id:             "0",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "returns not found",
			id:             "999",
			serviceError:   repository.ErrNotFound,
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "returns internal server error",
			id:             "1",
			serviceError:   errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockProductService{
				deleteFunc: func(
					_ context.Context,
					_ int64,
				) error {
					return tt.serviceError
				},
			}

			handler := NewProductHandler(mock)

			request := httptest.NewRequest(
				http.MethodDelete,
				"/api/v1/products/"+tt.id,
				nil,
			)

			request.SetPathValue("id", tt.id)

			recorder := httptest.NewRecorder()

			handler.Delete(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d",
					tt.expectedStatus,
					recorder.Code,
				)
			}
		})
	}
}

func TestProductHandler_Search(t *testing.T) {
	products := []*domain.Product{
		{
			ID:          1,
			Name:        "Laptop",
			SKU:         "LAP-001",
			Description: "Gaming laptop",
			Category:    "Computers",
			Price:       "2500.00",
			Stock:       10,
			WeightKg:    "2.500",
		},
	}

	tests := []struct {
		name           string
		query          string
		products       []*domain.Product
		serviceError   error
		expectedStatus int
		expectedCount  int
	}{
		{
			name:           "returns matching products",
			query:          "laptop",
			products:       products,
			expectedStatus: http.StatusOK,
			expectedCount:  1,
		},
		{
			name:           "returns empty result",
			query:          "keyboard",
			products:       []*domain.Product{},
			expectedStatus: http.StatusOK,
			expectedCount:  0,
		},
		{
			name:           "returns bad request for empty query",
			query:          "",
			serviceError:   service.ErrInvalidSearchQuery,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "returns internal server error",
			query:          "laptop",
			serviceError:   errors.New("database error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockProductService{
				searchFunc: func(
					_ context.Context,
					_ string,
				) ([]*domain.Product, error) {
					return tt.products, tt.serviceError
				},
			}

			handler := NewProductHandler(mock)

			request := httptest.NewRequest(
				http.MethodGet,
				"/api/v1/products/search?q="+tt.query,
				nil,
			)

			recorder := httptest.NewRecorder()

			handler.Search(recorder, request)

			if recorder.Code != tt.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d",
					tt.expectedStatus,
					recorder.Code,
				)
			}

			if tt.serviceError != nil {
				return
			}

			var response []map[string]any

			if err := json.NewDecoder(
				recorder.Body,
			).Decode(&response); err != nil {
				t.Fatalf(
					"failed to decode response: %v",
					err,
				)
			}

			if len(response) != tt.expectedCount {
				t.Fatalf(
					"expected %d products, got %d",
					tt.expectedCount,
					len(response),
				)
			}
		})
	}
}
