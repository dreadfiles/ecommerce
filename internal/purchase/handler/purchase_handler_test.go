package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	purchasedomain "ecommerce/internal/purchase/domain"
	"ecommerce/internal/purchase/repository"
	"ecommerce/internal/purchase/service"
)

type mockPurchaseService struct {
	createFunc  func(context.Context, []service.PurchaseItem, string, service.PaymentService) (*purchasedomain.Order, error)
	getAllFunc  func(context.Context) ([]*purchasedomain.Order, error)
	getByIDFunc func(context.Context, int64) (*purchasedomain.Order, error)

	createCalled  bool
	getAllCalled  bool
	getByIDCalled bool
}

func (m *mockPurchaseService) Create(
	ctx context.Context,
	items []service.PurchaseItem,
	idempotencyKey string,
	paymentService service.PaymentService,
) (*purchasedomain.Order, error) {
	m.createCalled = true

	if m.createFunc == nil {
		return nil, errors.New("create mock not configured")
	}

	return m.createFunc(
		ctx,
		items,
		idempotencyKey,
		paymentService,
	)
}

func (m *mockPurchaseService) GetAll(
	ctx context.Context,
) ([]*purchasedomain.Order, error) {
	m.getAllCalled = true

	if m.getAllFunc == nil {
		return nil, errors.New("get all mock not configured")
	}

	return m.getAllFunc(ctx)
}

func (m *mockPurchaseService) GetByID(
	ctx context.Context,
	id int64,
) (*purchasedomain.Order, error) {
	m.getByIDCalled = true

	if m.getByIDFunc == nil {
		return nil, errors.New("get by id mock not configured")
	}

	return m.getByIDFunc(ctx, id)
}

func TestPurchaseHandler_Create(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		idempotencyKey string
		paymentHeader  string
		serviceError   error
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid json",
			body:           `{"items":`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid request body",
		},
		{
			name: "successful purchase",
			body: `{"items":[
				{"product_id":1,"quantity":2},
				{"product_id":2,"quantity":1}
			]}`,
			idempotencyKey: "key-001",
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "maps invalid purchase",
			body:           `{"items":[]}`,
			serviceError:   service.ErrInvalidPurchase,
			expectedStatus: http.StatusBadRequest,
			expectedError:  service.ErrInvalidPurchase.Error(),
		},
		{
			name:           "maps invalid product",
			body:           `{"items":[{"product_id":0,"quantity":1}]}`,
			serviceError:   service.ErrInvalidProductID,
			expectedStatus: http.StatusBadRequest,
			expectedError:  service.ErrInvalidProductID.Error(),
		},
		{
			name:           "maps invalid quantity",
			body:           `{"items":[{"product_id":1,"quantity":0}]}`,
			serviceError:   service.ErrInvalidQuantity,
			expectedStatus: http.StatusBadRequest,
			expectedError:  service.ErrInvalidQuantity.Error(),
		},
		{
			name:           "maps missing idempotency key",
			body:           `{"items":[{"product_id":1,"quantity":1}]}`,
			serviceError:   service.ErrIdempotencyKeyRequired,
			expectedStatus: http.StatusBadRequest,
			expectedError:  service.ErrIdempotencyKeyRequired.Error(),
		},
		{
			name:           "maps product not found",
			body:           `{"items":[{"product_id":999,"quantity":1}]}`,
			serviceError:   service.ErrProductNotFound,
			expectedStatus: http.StatusNotFound,
			expectedError:  service.ErrProductNotFound.Error(),
		},
		{
			name:           "maps insufficient stock",
			body:           `{"items":[{"product_id":1,"quantity":100}]}`,
			serviceError:   service.ErrInsufficientStock,
			expectedStatus: http.StatusConflict,
			expectedError:  service.ErrInsufficientStock.Error(),
		},
		{
			name:           "maps idempotency conflict",
			body:           `{"items":[{"product_id":1,"quantity":1}]}`,
			serviceError:   service.ErrIdempotencyKeyConflict,
			expectedStatus: http.StatusConflict,
			expectedError:  service.ErrIdempotencyKeyConflict.Error(),
		},
		{
			name:           "maps payment declined",
			body:           `{"items":[{"product_id":1,"quantity":1}]}`,
			serviceError:   service.ErrPaymentDeclined,
			expectedStatus: http.StatusPaymentRequired,
			expectedError:  service.ErrPaymentDeclined.Error(),
		},
		{
			name:           "maps unexpected error",
			body:           `{"items":[{"product_id":1,"quantity":1}]}`,
			serviceError:   errors.New("database failure"),
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPurchaseService{
				createFunc: func(
					_ context.Context,
					_ []service.PurchaseItem,
					_ string,
					paymentService service.PaymentService,
				) (*purchasedomain.Order, error) {
					if tt.serviceError != nil {
						return nil, tt.serviceError
					}

					if paymentService == nil {
						return nil, errors.New(
							"payment service cannot be nil",
						)
					}

					return &purchasedomain.Order{
						ID:     10,
						Status: purchasedomain.OrderStatusPaid,
						Total:  "25.50",
						Items: []purchasedomain.OrderItem{
							{
								ProductID: 1,
								Quantity:  2,
								UnitPrice: "10.00",
								Subtotal:  "20.00",
							},
							{
								ProductID: 2,
								Quantity:  1,
								UnitPrice: "5.50",
								Subtotal:  "5.50",
							},
						},
					}, nil
				},
			}

			handler := NewPurchaseHandler(mockService)

			request := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/purchases",
				strings.NewReader(tt.body),
			)

			if tt.idempotencyKey != "" {
				request.Header.Set(
					idempotencyKeyHeader,
					tt.idempotencyKey,
				)
			}

			if tt.paymentHeader != "" {
				request.Header.Set(
					fakePaymentHeader,
					tt.paymentHeader,
				)
			}

			response := httptest.NewRecorder()

			handler.Create(response, request)

			if response.Code != tt.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d; body=%q",
					tt.expectedStatus,
					response.Code,
					response.Body.String(),
				)
			}

			if tt.expectedError != "" {
				var body map[string]string

				if err := json.NewDecoder(
					response.Body,
				).Decode(&body); err != nil {
					t.Fatalf(
						"failed to decode response: %v",
						err,
					)
				}

				if body["error"] != tt.expectedError {
					t.Errorf(
						"expected error %q, got %q",
						tt.expectedError,
						body["error"],
					)
				}
			}

			if tt.name != "invalid json" && !mockService.createCalled {
				t.Error("expected Create service to be called")
			}
		})
	}
}

func TestPurchaseHandler_Create_FakePaymentFailure(t *testing.T) {
	mockService := &mockPurchaseService{
		createFunc: func(
			_ context.Context,
			_ []service.PurchaseItem,
			_ string,
			paymentService service.PaymentService,
		) (*purchasedomain.Order, error) {
			fakePayment, ok := paymentService.(*service.FakePaymentService)
			if !ok {
				return nil, errors.New("expected FakePaymentService")
			}

			if fakePayment == nil {
				return nil, errors.New("expected non-nil payment service")
			}

			return nil, service.ErrPaymentDeclined
		},
	}

	handler := NewPurchaseHandler(mockService)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/purchases",
		strings.NewReader(
			`{"items":[{"product_id":1,"quantity":1}]}`,
		),
	)

	request.Header.Set(
		idempotencyKeyHeader,
		"key-payment-failure",
	)

	request.Header.Set(
		fakePaymentHeader,
		"failure",
	)

	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusPaymentRequired {
		t.Fatalf(
			"expected status %d, got %d; body=%q",
			http.StatusPaymentRequired,
			response.Code,
			response.Body.String(),
		)
	}

	var body map[string]string

	if err := json.NewDecoder(
		response.Body,
	).Decode(&body); err != nil {
		t.Fatalf(
			"failed to decode response: %v",
			err,
		)
	}

	if body["error"] != service.ErrPaymentDeclined.Error() {
		t.Errorf(
			"expected error %q, got %q",
			service.ErrPaymentDeclined.Error(),
			body["error"],
		)
	}
}

func TestPurchaseHandler_GetAll(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockService := &mockPurchaseService{
			getAllFunc: func(
				context.Context,
			) ([]*purchasedomain.Order, error) {
				return []*purchasedomain.Order{
					{
						ID:     1,
						Status: purchasedomain.OrderStatusPaid,
						Total:  "25.50",
						Items: []purchasedomain.OrderItem{
							{
								ProductID: 1,
								Quantity:  2,
								UnitPrice: "10.00",
								Subtotal:  "20.00",
							},
						},
					},
				}, nil
			},
		}

		handler := NewPurchaseHandler(mockService)

		request := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/purchases",
			nil,
		)

		response := httptest.NewRecorder()

		handler.GetAll(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf(
				"expected status 200, got %d",
				response.Code,
			)
		}

		if !mockService.getAllCalled {
			t.Fatal("expected GetAll service to be called")
		}

		var purchases []map[string]interface{}

		if err := json.NewDecoder(
			response.Body,
		).Decode(&purchases); err != nil {
			t.Fatalf(
				"failed to decode response: %v",
				err,
			)
		}

		if len(purchases) != 1 {
			t.Fatalf(
				"expected 1 purchase, got %d",
				len(purchases),
			)
		}

		if purchases[0]["id"] != float64(1) {
			t.Errorf(
				"expected id=1, got %v",
				purchases[0]["id"],
			)
		}

		if purchases[0]["status"] != "PAID" {
			t.Errorf(
				"expected status=PAID, got %v",
				purchases[0]["status"],
			)
		}

		if purchases[0]["total"] != "25.50" {
			t.Errorf(
				"expected total=25.50, got %v",
				purchases[0]["total"],
			)
		}

		items, ok := purchases[0]["items"].([]interface{})
		if !ok {
			t.Fatal("expected items to be an array")
		}

		if len(items) != 1 {
			t.Fatalf(
				"expected 1 item, got %d",
				len(items),
			)
		}
	})

	t.Run("service error", func(t *testing.T) {
		mockService := &mockPurchaseService{
			getAllFunc: func(
				context.Context,
			) ([]*purchasedomain.Order, error) {
				return nil, errors.New("database failure")
			},
		}

		handler := NewPurchaseHandler(mockService)

		request := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/purchases",
			nil,
		)

		response := httptest.NewRecorder()

		handler.GetAll(response, request)

		if response.Code != http.StatusInternalServerError {
			t.Fatalf(
				"expected status %d, got %d; body=%q",
				http.StatusInternalServerError,
				response.Code,
				response.Body.String(),
			)
		}

		var body map[string]string

		if err := json.NewDecoder(
			response.Body,
		).Decode(&body); err != nil {
			t.Fatalf(
				"failed to decode response: %v",
				err,
			)
		}

		if body["error"] != "internal server error" {
			t.Errorf(
				"expected error %q, got %q",
				"internal server error",
				body["error"],
			)
		}
	})
}

func TestPurchaseHandler_GetByID(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		serviceError   error
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid id",
			id:             "abc",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid purchase id",
		},
		{
			name:           "zero id",
			id:             "0",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid purchase id",
		},
		{
			name:           "negative id",
			id:             "-1",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "invalid purchase id",
		},
		{
			name:           "not found",
			id:             "10",
			serviceError:   repository.ErrOrderNotFound,
			expectedStatus: http.StatusNotFound,
			expectedError:  "purchase not found",
		},
		{
			name:           "invalid purchase",
			id:             "10",
			serviceError:   service.ErrInvalidPurchase,
			expectedStatus: http.StatusBadRequest,
			expectedError:  service.ErrInvalidPurchase.Error(),
		},
		{
			name:           "unexpected error",
			id:             "10",
			serviceError:   errors.New("database failure"),
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockPurchaseService{
				getByIDFunc: func(
					context.Context,
					int64,
				) (*purchasedomain.Order, error) {
					if tt.serviceError != nil {
						return nil, tt.serviceError
					}

					return &purchasedomain.Order{
						ID:     10,
						Status: purchasedomain.OrderStatusPaid,
						Total:  "25.50",
					}, nil
				},
			}

			handler := NewPurchaseHandler(mockService)

			request := httptest.NewRequest(
				http.MethodGet,
				"/api/v1/purchases/"+tt.id,
				nil,
			)

			request.SetPathValue("id", tt.id)

			response := httptest.NewRecorder()

			handler.GetByID(response, request)

			if response.Code != tt.expectedStatus {
				t.Fatalf(
					"expected status %d, got %d",
					tt.expectedStatus,
					response.Code,
				)
			}

			if tt.id != "abc" &&
				tt.id != "0" &&
				tt.id != "-1" &&
				!mockService.getByIDCalled {
				t.Error("expected GetByID service to be called")
			}

			if tt.expectedError != "" {
				var body map[string]string

				if err := json.NewDecoder(
					response.Body,
				).Decode(&body); err != nil {
					t.Fatalf(
						"failed to decode response: %v",
						err,
					)
				}

				if body["error"] != tt.expectedError {
					t.Errorf(
						"expected error %q, got %q",
						tt.expectedError,
						body["error"],
					)
				}
			}
		})
	}

	t.Run("success", func(t *testing.T) {
		mockService := &mockPurchaseService{
			getByIDFunc: func(
				_ context.Context,
				id int64,
			) (*purchasedomain.Order, error) {
				if id != 10 {
					return nil, errors.New("unexpected id")
				}

				return &purchasedomain.Order{
					ID:     10,
					Status: purchasedomain.OrderStatusPaid,
					Total:  "25.50",
					Items: []purchasedomain.OrderItem{
						{
							ProductID: 1,
							Quantity:  2,
							UnitPrice: "10.00",
							Subtotal:  "20.00",
						},
					},
				}, nil
			},
		}

		handler := NewPurchaseHandler(mockService)

		request := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/purchases/10",
			nil,
		)

		request.SetPathValue("id", "10")

		response := httptest.NewRecorder()

		handler.GetByID(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf(
				"expected status 200, got %d; body=%q",
				response.Code,
				response.Body.String(),
			)
		}

		if !mockService.getByIDCalled {
			t.Fatal("expected GetByID service to be called")
		}

		var purchase map[string]interface{}

		if err := json.NewDecoder(
			response.Body,
		).Decode(&purchase); err != nil {
			t.Fatalf(
				"failed to decode response: %v",
				err,
			)
		}

		if purchase["id"] != float64(10) {
			t.Errorf(
				"expected id=10, got %v",
				purchase["id"],
			)
		}

		if purchase["status"] != "PAID" {
			t.Errorf(
				"expected status=PAID, got %v",
				purchase["status"],
			)
		}

		if purchase["total"] != "25.50" {
			t.Errorf(
				"expected total=25.50, got %v",
				purchase["total"],
			)
		}

		items, ok := purchase["items"].([]interface{})
		if !ok {
			t.Fatal("expected items to be an array")
		}

		if len(items) != 1 {
			t.Fatalf(
				"expected 1 item, got %d",
				len(items),
			)
		}
	})
}

func TestPurchaseHandler_ToPurchaseResponse(t *testing.T) {
	order := &purchasedomain.Order{
		ID:     10,
		Status: purchasedomain.OrderStatusPaid,
		Total:  "25.50",
		Items: []purchasedomain.OrderItem{
			{
				ProductID: 1,
				Quantity:  2,
				UnitPrice: "10.00",
				Subtotal:  "20.00",
			},
			{
				ProductID: 2,
				Quantity:  1,
				UnitPrice: "5.50",
				Subtotal:  "5.50",
			},
		},
	}

	response := toPurchaseResponse(order)

	if response.ID != 10 {
		t.Errorf(
			"expected id=10, got %d",
			response.ID,
		)
	}

	if response.Status != "PAID" {
		t.Errorf(
			"expected status=PAID, got %q",
			response.Status,
		)
	}

	if response.Total != "25.50" {
		t.Errorf(
			"expected total=25.50, got %q",
			response.Total,
		)
	}

	if len(response.Items) != 2 {
		t.Fatalf(
			"expected 2 items, got %d",
			len(response.Items),
		)
	}

	if response.Items[0].ProductID != 1 {
		t.Errorf(
			"expected first product_id=1, got %d",
			response.Items[0].ProductID,
		)
	}

	if response.Items[0].Quantity != 2 {
		t.Errorf(
			"expected first quantity=2, got %d",
			response.Items[0].Quantity,
		)
	}

	if response.Items[0].UnitPrice != "10.00" {
		t.Errorf(
			"expected first unit_price=10.00, got %q",
			response.Items[0].UnitPrice,
		)
	}

	if response.Items[0].Subtotal != "20.00" {
		t.Errorf(
			"expected first subtotal=20.00, got %q",
			response.Items[0].Subtotal,
		)
	}

	if response.Items[1].ProductID != 2 {
		t.Errorf(
			"expected second product_id=2, got %d",
			response.Items[1].ProductID,
		)
	}
}
