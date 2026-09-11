package service

import (
	"context"
	"errors"
	"testing"

	purchasedomain "ecommerce/internal/purchase/domain"
	"ecommerce/internal/purchase/repository"
)

type mockPurchaseRepository struct {
	getQuoteFunc            func(context.Context, []purchasedomain.OrderItem) (*purchasedomain.Order, error)
	createFunc              func(context.Context, *purchasedomain.Order) error
	getByIdempotencyKeyFunc func(context.Context, string) (*purchasedomain.Order, error)
	getByIDFunc             func(context.Context, int64) (*purchasedomain.Order, error)
	getAllFunc              func(context.Context) ([]*purchasedomain.Order, error)

	getQuoteCalled            bool
	createCalled              bool
	getByIdempotencyKeyCalled bool
	getByIDCalled             bool
	getAllCalled              bool
}

func (m *mockPurchaseRepository) GetQuote(
	ctx context.Context,
	items []purchasedomain.OrderItem,
) (*purchasedomain.Order, error) {
	m.getQuoteCalled = true

	if m.getQuoteFunc == nil {
		return nil, errors.New("get quote mock not configured")
	}

	return m.getQuoteFunc(ctx, items)
}

func (m *mockPurchaseRepository) Create(
	ctx context.Context,
	order *purchasedomain.Order,
) error {
	m.createCalled = true

	if m.createFunc == nil {
		return errors.New("create mock not configured")
	}

	return m.createFunc(ctx, order)
}

func (m *mockPurchaseRepository) GetByIdempotencyKey(
	ctx context.Context,
	key string,
) (*purchasedomain.Order, error) {
	m.getByIdempotencyKeyCalled = true

	if m.getByIdempotencyKeyFunc == nil {
		return nil, errors.New(
			"get idempotency key mock not configured",
		)
	}

	return m.getByIdempotencyKeyFunc(ctx, key)
}

func (m *mockPurchaseRepository) GetByID(
	ctx context.Context,
	id int64,
) (*purchasedomain.Order, error) {
	m.getByIDCalled = true

	if m.getByIDFunc == nil {
		return nil, errors.New("get by id mock not configured")
	}

	return m.getByIDFunc(ctx, id)
}

func (m *mockPurchaseRepository) GetAll(
	ctx context.Context,
) ([]*purchasedomain.Order, error) {
	m.getAllCalled = true

	if m.getAllFunc == nil {
		return nil, errors.New("get all mock not configured")
	}

	return m.getAllFunc(ctx)
}

type mockPaymentService struct {
	processFunc func(context.Context, string) error
	called      bool
}

func (m *mockPaymentService) Process(
	ctx context.Context,
	amount string,
) error {
	m.called = true

	if m.processFunc == nil {
		return errors.New("payment mock not configured")
	}

	return m.processFunc(ctx, amount)
}

func purchaseItems() []PurchaseItem {
	return []PurchaseItem{
		{
			ProductID: 1,
			Quantity:  2,
		},
		{
			ProductID: 2,
			Quantity:  1,
		},
	}
}

func quotedOrder() *purchasedomain.Order {
	return &purchasedomain.Order{
		Total: "25.50",
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
}

func TestNewPurchaseService(t *testing.T) {
	repositoryMock := &mockPurchaseRepository{}

	service := NewPurchaseService(repositoryMock)

	if service == nil {
		t.Fatal("expected service")
	}
}

func TestPurchaseService_Create(t *testing.T) {
	tests := []struct {
		name                string
		items               []PurchaseItem
		idempotencyKey      string
		getExisting         *purchasedomain.Order
		getExistingError    error
		quote               *purchasedomain.Order
		quoteError          error
		paymentError        error
		createError         error
		expectedError       error
		expectedErrorText   string
		expectQuote         bool
		expectPayment       bool
		expectCreate        bool
		expectExistingCheck bool
		expectNilPayment    bool
	}{
		{
			name:                "creates purchase successfully",
			items:               purchaseItems(),
			idempotencyKey:      "key-001",
			getExistingError:    repository.ErrOrderNotFound,
			quote:               quotedOrder(),
			expectExistingCheck: true,
			expectQuote:         true,
			expectPayment:       true,
			expectCreate:        true,
		},
		{
			name:           "rejects empty items",
			items:          nil,
			idempotencyKey: "key-001",
			expectedError:  ErrInvalidPurchase,
		},
		{
			name:           "rejects missing idempotency key",
			items:          purchaseItems(),
			idempotencyKey: "",
			expectedError:  ErrIdempotencyKeyRequired,
		},
		{
			name: "rejects invalid product id",
			items: []PurchaseItem{
				{
					ProductID: 0,
					Quantity:  1,
				},
			},
			idempotencyKey: "key-001",
			expectedError:  ErrInvalidProductID,
		},
		{
			name: "rejects invalid quantity",
			items: []PurchaseItem{
				{
					ProductID: 1,
					Quantity:  0,
				},
			},
			idempotencyKey: "key-001",
			expectedError:  ErrInvalidQuantity,
		},
		{
			name: "rejects duplicate products",
			items: []PurchaseItem{
				{
					ProductID: 1,
					Quantity:  1,
				},
				{
					ProductID: 1,
					Quantity:  2,
				},
			},
			idempotencyKey: "key-001",
			expectedError:  ErrInvalidPurchase,
		},
		{
			name:              "rejects nil payment service",
			items:             purchaseItems(),
			idempotencyKey:    "key-001",
			expectedErrorText: "payment service cannot be nil",
			expectNilPayment:  true,
		},
		{
			name:                "maps idempotency lookup error",
			items:               purchaseItems(),
			idempotencyKey:      "key-001",
			getExistingError:    errors.New("database unavailable"),
			expectedErrorText:   "check idempotency key: database unavailable",
			expectExistingCheck: true,
		},
		{
			name:                "maps quote generic error",
			items:               purchaseItems(),
			idempotencyKey:      "key-001",
			getExistingError:    repository.ErrOrderNotFound,
			quoteError:          errors.New("quote database error"),
			expectedErrorText:   "get purchase quote: quote database error",
			expectExistingCheck: true,
			expectQuote:         true,
		},
		{
			name:                "maps payment generic error",
			items:               purchaseItems(),
			idempotencyKey:      "key-001",
			getExistingError:    repository.ErrOrderNotFound,
			quote:               quotedOrder(),
			paymentError:        errors.New("payment provider unavailable"),
			expectedErrorText:   "process payment: payment provider unavailable",
			expectExistingCheck: true,
			expectQuote:         true,
			expectPayment:       true,
		},
		{
			name:                "maps create generic error",
			items:               purchaseItems(),
			idempotencyKey:      "key-001",
			getExistingError:    repository.ErrOrderNotFound,
			quote:               quotedOrder(),
			createError:         errors.New("insert failed"),
			expectedErrorText:   "create purchase: insert failed",
			expectExistingCheck: true,
			expectQuote:         true,
			expectPayment:       true,
			expectCreate:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repositoryMock := &mockPurchaseRepository{
				getByIdempotencyKeyFunc: func(
					context.Context,
					string,
				) (*purchasedomain.Order, error) {
					return tt.getExisting, tt.getExistingError
				},
				getQuoteFunc: func(
					context.Context,
					[]purchasedomain.OrderItem,
				) (*purchasedomain.Order, error) {
					return tt.quote, tt.quoteError
				},
				createFunc: func(
					_ context.Context,
					order *purchasedomain.Order,
				) error {
					order.ID = 100
					return tt.createError
				},
			}

			paymentMock := &mockPaymentService{
				processFunc: func(
					context.Context,
					string,
				) error {
					return tt.paymentError
				},
			}

			var paymentService PaymentService = paymentMock

			if tt.expectNilPayment {
				paymentService = nil
			}

			service := NewPurchaseService(repositoryMock)

			order, err := service.Create(
				context.Background(),
				tt.items,
				tt.idempotencyKey,
				paymentService,
			)

			if tt.expectedError != nil {
				if err == nil {
					t.Fatalf(
						"expected error %v",
						tt.expectedError,
					)
				}

				if !errors.Is(err, tt.expectedError) {
					t.Fatalf(
						"expected error %v, got %v",
						tt.expectedError,
						err,
					)
				}

				return
			}

			if tt.expectedErrorText != "" {
				if err == nil {
					t.Fatalf(
						"expected error %q",
						tt.expectedErrorText,
					)
				}

				if err.Error() != tt.expectedErrorText {
					t.Fatalf(
						"expected error %q, got %q",
						tt.expectedErrorText,
						err.Error(),
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

			if order == nil {
				t.Fatal("expected order")
			}

			if repositoryMock.getByIdempotencyKeyCalled != tt.expectExistingCheck {
				t.Errorf(
					"expected idempotency lookup=%v, got %v",
					tt.expectExistingCheck,
					repositoryMock.getByIdempotencyKeyCalled,
				)
			}

			if repositoryMock.getQuoteCalled != tt.expectQuote {
				t.Errorf(
					"expected quote=%v, got %v",
					tt.expectQuote,
					repositoryMock.getQuoteCalled,
				)
			}

			if paymentMock.called != tt.expectPayment {
				t.Errorf(
					"expected payment=%v, got %v",
					tt.expectPayment,
					paymentMock.called,
				)
			}

			if repositoryMock.createCalled != tt.expectCreate {
				t.Errorf(
					"expected create=%v, got %v",
					tt.expectCreate,
					repositoryMock.createCalled,
				)
			}
		})
	}
}

func TestPurchaseService_Create_Idempotency(t *testing.T) {
	t.Run("returns existing order for same purchase", func(t *testing.T) {
		existing := &purchasedomain.Order{
			ID:             50,
			Status:         purchasedomain.OrderStatusPaid,
			Total:          "25.50",
			IdempotencyKey: "key-001",
			Items: []purchasedomain.OrderItem{
				{
					ProductID: 1,
					Quantity:  2,
				},
				{
					ProductID: 2,
					Quantity:  1,
				},
			},
		}

		repositoryMock := &mockPurchaseRepository{
			getByIdempotencyKeyFunc: func(
				context.Context,
				string,
			) (*purchasedomain.Order, error) {
				return existing, nil
			},
		}

		paymentMock := &mockPaymentService{
			processFunc: func(
				context.Context,
				string,
			) error {
				t.Fatal("payment should not be processed")
				return nil
			},
		}

		service := NewPurchaseService(repositoryMock)

		order, err := service.Create(
			context.Background(),
			purchaseItems(),
			"key-001",
			paymentMock,
		)

		if err != nil {
			t.Fatalf(
				"expected no error, got %v",
				err,
			)
		}

		if order != existing {
			t.Fatal("expected existing order")
		}

		if !repositoryMock.getByIdempotencyKeyCalled {
			t.Fatal("expected idempotency lookup")
		}

		if repositoryMock.getQuoteCalled {
			t.Fatal("quote should not be requested")
		}

		if repositoryMock.createCalled {
			t.Fatal("order should not be created again")
		}

		if paymentMock.called {
			t.Fatal("payment should not be processed again")
		}
	})

	t.Run("rejects same key with different purchase", func(t *testing.T) {
		existing := &purchasedomain.Order{
			ID: 50,
			Items: []purchasedomain.OrderItem{
				{
					ProductID: 1,
					Quantity:  5,
				},
			},
		}

		repositoryMock := &mockPurchaseRepository{
			getByIdempotencyKeyFunc: func(
				context.Context,
				string,
			) (*purchasedomain.Order, error) {
				return existing, nil
			},
		}

		service := NewPurchaseService(repositoryMock)

		_, err := service.Create(
			context.Background(),
			purchaseItems(),
			"key-001",
			&mockPaymentService{},
		)

		if !errors.Is(
			err,
			ErrIdempotencyKeyConflict,
		) {
			t.Fatalf(
				"expected ErrIdempotencyKeyConflict, got %v",
				err,
			)
		}

		if repositoryMock.getQuoteCalled {
			t.Fatal("quote should not be requested")
		}

		if repositoryMock.createCalled {
			t.Fatal("order should not be created")
		}
	})

	t.Run("normalizes idempotency key", func(t *testing.T) {
		var receivedKey string

		repositoryMock := &mockPurchaseRepository{
			getByIdempotencyKeyFunc: func(
				_ context.Context,
				key string,
			) (*purchasedomain.Order, error) {
				receivedKey = key
				return nil, repository.ErrOrderNotFound
			},
			getQuoteFunc: func(
				_ context.Context,
				_ []purchasedomain.OrderItem,
			) (*purchasedomain.Order, error) {
				return quotedOrder(), nil
			},
			createFunc: func(
				_ context.Context,
				order *purchasedomain.Order,
			) error {
				if order.IdempotencyKey != "key-001" {
					t.Fatalf(
						"expected normalized key key-001, got %q",
						order.IdempotencyKey,
					)
				}

				order.ID = 100

				return nil
			},
		}

		paymentMock := &mockPaymentService{
			processFunc: func(
				_ context.Context,
				_ string,
			) error {
				return nil
			},
		}

		service := NewPurchaseService(repositoryMock)

		_, err := service.Create(
			context.Background(),
			purchaseItems(),
			"  key-001  ",
			paymentMock,
		)

		if err != nil {
			t.Fatalf(
				"expected no error, got %v",
				err,
			)
		}

		if receivedKey != "key-001" {
			t.Fatalf(
				"expected normalized lookup key key-001, got %q",
				receivedKey,
			)
		}
	})

	t.Run("handles idempotency conflict and returns existing order", func(t *testing.T) {
		existing := &purchasedomain.Order{
			ID:             200,
			Status:         purchasedomain.OrderStatusPaid,
			Total:          "25.50",
			IdempotencyKey: "key-001",
			Items: []purchasedomain.OrderItem{
				{
					ProductID: 1,
					Quantity:  2,
				},
				{
					ProductID: 2,
					Quantity:  1,
				},
			},
		}

		lookupCalls := 0

		repositoryMock := &mockPurchaseRepository{
			getByIdempotencyKeyFunc: func(
				_ context.Context,
				_ string,
			) (*purchasedomain.Order, error) {
				lookupCalls++

				if lookupCalls == 1 {
					return nil, repository.ErrOrderNotFound
				}

				return existing, nil
			},
			getQuoteFunc: func(
				_ context.Context,
				_ []purchasedomain.OrderItem,
			) (*purchasedomain.Order, error) {
				return quotedOrder(), nil
			},
			createFunc: func(
				_ context.Context,
				_ *purchasedomain.Order,
			) error {
				return repository.ErrIdempotencyKeyExists
			},
		}

		paymentMock := &mockPaymentService{
			processFunc: func(
				_ context.Context,
				_ string,
			) error {
				return nil
			},
		}

		service := NewPurchaseService(repositoryMock)

		order, err := service.Create(
			context.Background(),
			purchaseItems(),
			"key-001",
			paymentMock,
		)

		if err != nil {
			t.Fatalf(
				"expected no error, got %v",
				err,
			)
		}

		if order != existing {
			t.Fatal("expected existing order")
		}

		if lookupCalls != 2 {
			t.Fatalf(
				"expected 2 idempotency lookups, got %d",
				lookupCalls,
			)
		}

		if !repositoryMock.createCalled {
			t.Fatal("expected create to be called")
		}
	})

	t.Run("maps idempotency conflict lookup error", func(t *testing.T) {
		lookupCalls := 0
		expectedError := errors.New("database unavailable")

		repositoryMock := &mockPurchaseRepository{
			getByIdempotencyKeyFunc: func(
				_ context.Context,
				_ string,
			) (*purchasedomain.Order, error) {
				lookupCalls++

				if lookupCalls == 1 {
					return nil, repository.ErrOrderNotFound
				}

				return nil, expectedError
			},
			getQuoteFunc: func(
				_ context.Context,
				_ []purchasedomain.OrderItem,
			) (*purchasedomain.Order, error) {
				return quotedOrder(), nil
			},
			createFunc: func(
				_ context.Context,
				_ *purchasedomain.Order,
			) error {
				return repository.ErrIdempotencyKeyExists
			},
		}

		paymentMock := &mockPaymentService{
			processFunc: func(
				_ context.Context,
				_ string,
			) error {
				return nil
			},
		}

		service := NewPurchaseService(repositoryMock)

		_, err := service.Create(
			context.Background(),
			purchaseItems(),
			"key-001",
			paymentMock,
		)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, expectedError) {
			t.Fatalf(
				"expected wrapped database error, got %v",
				err,
			)
		}

		if err.Error() != "get existing purchase: database unavailable" {
			t.Fatalf(
				"expected get existing purchase error, got %v",
				err,
			)
		}
	})

	t.Run("rejects idempotency conflict with different purchase", func(t *testing.T) {
		existing := &purchasedomain.Order{
			ID: 300,
			Items: []purchasedomain.OrderItem{
				{
					ProductID: 1,
					Quantity:  99,
				},
			},
		}

		lookupCalls := 0

		repositoryMock := &mockPurchaseRepository{
			getByIdempotencyKeyFunc: func(
				_ context.Context,
				_ string,
			) (*purchasedomain.Order, error) {
				lookupCalls++

				if lookupCalls == 1 {
					return nil, repository.ErrOrderNotFound
				}

				return existing, nil
			},
			getQuoteFunc: func(
				_ context.Context,
				_ []purchasedomain.OrderItem,
			) (*purchasedomain.Order, error) {
				return quotedOrder(), nil
			},
			createFunc: func(
				_ context.Context,
				_ *purchasedomain.Order,
			) error {
				return repository.ErrIdempotencyKeyExists
			},
		}

		paymentMock := &mockPaymentService{
			processFunc: func(
				_ context.Context,
				_ string,
			) error {
				return nil
			},
		}

		service := NewPurchaseService(repositoryMock)

		_, err := service.Create(
			context.Background(),
			purchaseItems(),
			"key-001",
			paymentMock,
		)

		if !errors.Is(
			err,
			ErrIdempotencyKeyConflict,
		) {
			t.Fatalf(
				"expected ErrIdempotencyKeyConflict, got %v",
				err,
			)
		}
	})
}

func TestPurchaseService_Create_ErrorMapping(t *testing.T) {
	tests := []struct {
		name          string
		quoteError    error
		paymentError  error
		createError   error
		expectedError error
	}{
		{
			name:          "maps product not found",
			quoteError:    repository.ErrProductNotFound,
			expectedError: ErrProductNotFound,
		},
		{
			name:          "maps insufficient stock",
			quoteError:    repository.ErrInsufficientStock,
			expectedError: ErrInsufficientStock,
		},
		{
			name:          "maps payment declined",
			paymentError:  ErrPaymentDeclined,
			expectedError: ErrPaymentDeclined,
		},
		{
			name:          "maps create product not found",
			createError:   repository.ErrProductNotFound,
			expectedError: ErrProductNotFound,
		},
		{
			name:          "maps create insufficient stock",
			createError:   repository.ErrInsufficientStock,
			expectedError: ErrInsufficientStock,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repositoryMock := &mockPurchaseRepository{
				getByIdempotencyKeyFunc: func(
					context.Context,
					string,
				) (*purchasedomain.Order, error) {
					return nil, repository.ErrOrderNotFound
				},
				getQuoteFunc: func(
					context.Context,
					[]purchasedomain.OrderItem,
				) (*purchasedomain.Order, error) {
					if tt.quoteError != nil {
						return nil, tt.quoteError
					}

					return quotedOrder(), nil
				},
				createFunc: func(
					context.Context,
					*purchasedomain.Order,
				) error {
					return tt.createError
				},
			}

			paymentMock := &mockPaymentService{
				processFunc: func(
					context.Context,
					string,
				) error {
					return tt.paymentError
				},
			}

			service := NewPurchaseService(repositoryMock)

			_, err := service.Create(
				context.Background(),
				purchaseItems(),
				"key-001",
				paymentMock,
			)

			if !errors.Is(
				err,
				tt.expectedError,
			) {
				t.Fatalf(
					"expected %v, got %v",
					tt.expectedError,
					err,
				)
			}
		})
	}
}

func TestPurchaseService_Create_PaymentDeclined(t *testing.T) {
	repositoryMock := &mockPurchaseRepository{
		getByIdempotencyKeyFunc: func(
			_ context.Context,
			_ string,
		) (*purchasedomain.Order, error) {
			return nil, repository.ErrOrderNotFound
		},
		getQuoteFunc: func(
			_ context.Context,
			_ []purchasedomain.OrderItem,
		) (*purchasedomain.Order, error) {
			return quotedOrder(), nil
		},
		createFunc: func(
			_ context.Context,
			_ *purchasedomain.Order,
		) error {
			t.Fatal("create should not be called when payment is declined")
			return nil
		},
	}

	paymentMock := &mockPaymentService{
		processFunc: func(
			_ context.Context,
			_ string,
		) error {
			return ErrPaymentDeclined
		},
	}

	service := NewPurchaseService(repositoryMock)

	_, err := service.Create(
		context.Background(),
		purchaseItems(),
		"key-001",
		paymentMock,
	)

	if !errors.Is(err, ErrPaymentDeclined) {
		t.Fatalf(
			"expected ErrPaymentDeclined, got %v",
			err,
		)
	}

	if !paymentMock.called {
		t.Fatal("expected payment to be called")
	}

	if repositoryMock.createCalled {
		t.Fatal("create should not be called")
	}
}

func TestPurchaseService_GetByID(t *testing.T) {
	t.Run("rejects invalid id", func(t *testing.T) {
		repositoryMock := &mockPurchaseRepository{
			getByIDFunc: func(
				context.Context,
				int64,
			) (*purchasedomain.Order, error) {
				return nil, nil
			},
		}

		service := NewPurchaseService(repositoryMock)

		_, err := service.GetByID(
			context.Background(),
			0,
		)

		if !errors.Is(
			err,
			ErrInvalidPurchase,
		) {
			t.Fatalf(
				"expected ErrInvalidPurchase, got %v",
				err,
			)
		}

		if repositoryMock.getByIDCalled {
			t.Fatal("repository should not be called")
		}
	})

	t.Run("rejects negative id", func(t *testing.T) {
		repositoryMock := &mockPurchaseRepository{}

		service := NewPurchaseService(repositoryMock)

		_, err := service.GetByID(
			context.Background(),
			-1,
		)

		if !errors.Is(
			err,
			ErrInvalidPurchase,
		) {
			t.Fatalf(
				"expected ErrInvalidPurchase, got %v",
				err,
			)
		}

		if repositoryMock.getByIDCalled {
			t.Fatal("repository should not be called")
		}
	})

	t.Run("returns order", func(t *testing.T) {
		expected := &purchasedomain.Order{
			ID:    10,
			Total: "20.00",
		}

		repositoryMock := &mockPurchaseRepository{
			getByIDFunc: func(
				context.Context,
				int64,
			) (*purchasedomain.Order, error) {
				return expected, nil
			},
		}

		service := NewPurchaseService(repositoryMock)

		order, err := service.GetByID(
			context.Background(),
			10,
		)

		if err != nil {
			t.Fatalf(
				"expected no error, got %v",
				err,
			)
		}

		if order != expected {
			t.Fatal("expected repository order")
		}

		if !repositoryMock.getByIDCalled {
			t.Fatal("expected GetByID to be called")
		}
	})

	t.Run("maps not found", func(t *testing.T) {
		repositoryMock := &mockPurchaseRepository{
			getByIDFunc: func(
				context.Context,
				int64,
			) (*purchasedomain.Order, error) {
				return nil, repository.ErrOrderNotFound
			},
		}

		service := NewPurchaseService(repositoryMock)

		_, err := service.GetByID(
			context.Background(),
			10,
		)

		if !errors.Is(
			err,
			repository.ErrOrderNotFound,
		) {
			t.Fatalf(
				"expected ErrOrderNotFound, got %v",
				err,
			)
		}
	})

	t.Run("maps generic repository error", func(t *testing.T) {
		repositoryError := errors.New("database unavailable")

		repositoryMock := &mockPurchaseRepository{
			getByIDFunc: func(
				context.Context,
				int64,
			) (*purchasedomain.Order, error) {
				return nil, repositoryError
			},
		}

		service := NewPurchaseService(repositoryMock)

		_, err := service.GetByID(
			context.Background(),
			10,
		)

		if err == nil {
			t.Fatal("expected error")
		}

		if !errors.Is(err, repositoryError) {
			t.Fatalf(
				"expected wrapped repository error, got %v",
				err,
			)
		}

		if err.Error() != "get purchase: database unavailable" {
			t.Fatalf(
				"expected get purchase error, got %v",
				err,
			)
		}
	})
}

func TestPurchaseService_GetAll(t *testing.T) {
	t.Run("returns orders", func(t *testing.T) {
		expected := []*purchasedomain.Order{
			{
				ID:    1,
				Total: "10.00",
			},
		}

		repositoryMock := &mockPurchaseRepository{
			getAllFunc: func(
				context.Context,
			) ([]*purchasedomain.Order, error) {
				return expected, nil
			},
		}

		service := NewPurchaseService(repositoryMock)

		orders, err := service.GetAll(
			context.Background(),
		)

		if err != nil {
			t.Fatalf(
				"expected no error, got %v",
				err,
			)
		}

		if len(orders) != 1 {
			t.Fatalf(
				"expected 1 order, got %d",
				len(orders),
			)
		}

		if orders[0] != expected[0] {
			t.Fatal("expected repository order")
		}

		if !repositoryMock.getAllCalled {
			t.Fatal("expected GetAll to be called")
		}
	})

	t.Run("propagates repository error", func(t *testing.T) {
		expectedError := errors.New("database unavailable")

		repositoryMock := &mockPurchaseRepository{
			getAllFunc: func(
				context.Context,
			) ([]*purchasedomain.Order, error) {
				return nil, expectedError
			},
		}

		service := NewPurchaseService(repositoryMock)

		orders, err := service.GetAll(
			context.Background(),
		)

		if orders != nil {
			t.Fatal("expected nil orders")
		}

		if !errors.Is(err, expectedError) {
			t.Fatalf(
				"expected repository error, got %v",
				err,
			)
		}
	})
}

func TestValidatePurchase(t *testing.T) {
	tests := []struct {
		name           string
		items          []PurchaseItem
		idempotencyKey string
		expectedError  error
	}{
		{
			name:           "valid purchase",
			items:          purchaseItems(),
			idempotencyKey: "key-001",
		},
		{
			name:           "empty items",
			items:          nil,
			idempotencyKey: "key-001",
			expectedError:  ErrInvalidPurchase,
		},
		{
			name: "invalid zero product id",
			items: []PurchaseItem{
				{
					ProductID: 0,
					Quantity:  1,
				},
			},
			idempotencyKey: "key-001",
			expectedError:  ErrInvalidProductID,
		},
		{
			name: "invalid negative product id",
			items: []PurchaseItem{
				{
					ProductID: -1,
					Quantity:  1,
				},
			},
			idempotencyKey: "key-001",
			expectedError:  ErrInvalidProductID,
		},
		{
			name: "invalid zero quantity",
			items: []PurchaseItem{
				{
					ProductID: 1,
					Quantity:  0,
				},
			},
			idempotencyKey: "key-001",
			expectedError:  ErrInvalidQuantity,
		},
		{
			name: "invalid negative quantity",
			items: []PurchaseItem{
				{
					ProductID: 1,
					Quantity:  -1,
				},
			},
			idempotencyKey: "key-001",
			expectedError:  ErrInvalidQuantity,
		},
		{
			name: "duplicate products",
			items: []PurchaseItem{
				{
					ProductID: 1,
					Quantity:  1,
				},
				{
					ProductID: 1,
					Quantity:  2,
				},
			},
			idempotencyKey: "key-001",
			expectedError:  ErrInvalidPurchase,
		},
		{
			name:           "empty idempotency key",
			items:          purchaseItems(),
			idempotencyKey: "",
			expectedError:  ErrIdempotencyKeyRequired,
		},
		{
			name:           "whitespace idempotency key",
			items:          purchaseItems(),
			idempotencyKey: "   ",
			expectedError:  ErrIdempotencyKeyRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePurchase(
				tt.items,
				tt.idempotencyKey,
			)

			if tt.expectedError == nil {
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
					"expected error %v",
					tt.expectedError,
				)
			}

			if !errors.Is(err, tt.expectedError) {
				t.Fatalf(
					"expected %v, got %v",
					tt.expectedError,
					err,
				)
			}
		})
	}
}

func TestSamePurchaseItems(t *testing.T) {
	tests := []struct {
		name     string
		request  []PurchaseItem
		order    []purchasedomain.OrderItem
		expected bool
	}{
		{
			name: "same items",
			request: []PurchaseItem{
				{ProductID: 1, Quantity: 2},
				{ProductID: 2, Quantity: 1},
			},
			order: []purchasedomain.OrderItem{
				{ProductID: 1, Quantity: 2},
				{ProductID: 2, Quantity: 1},
			},
			expected: true,
		},
		{
			name: "same items different order",
			request: []PurchaseItem{
				{ProductID: 1, Quantity: 2},
				{ProductID: 2, Quantity: 1},
			},
			order: []purchasedomain.OrderItem{
				{ProductID: 2, Quantity: 1},
				{ProductID: 1, Quantity: 2},
			},
			expected: true,
		},
		{
			name: "different quantity",
			request: []PurchaseItem{
				{ProductID: 1, Quantity: 3},
			},
			order: []purchasedomain.OrderItem{
				{ProductID: 1, Quantity: 2},
			},
			expected: false,
		},
		{
			name: "different product",
			request: []PurchaseItem{
				{ProductID: 1, Quantity: 2},
			},
			order: []purchasedomain.OrderItem{
				{ProductID: 2, Quantity: 2},
			},
			expected: false,
		},
		{
			name: "different length",
			request: []PurchaseItem{
				{ProductID: 1, Quantity: 2},
			},
			order: []purchasedomain.OrderItem{
				{ProductID: 1, Quantity: 2},
				{ProductID: 2, Quantity: 1},
			},
			expected: false,
		},
		{
			name: "request contains missing product",
			request: []PurchaseItem{
				{ProductID: 1, Quantity: 2},
				{ProductID: 3, Quantity: 1},
			},
			order: []purchasedomain.OrderItem{
				{ProductID: 1, Quantity: 2},
				{ProductID: 2, Quantity: 1},
			},
			expected: false,
		},
		{
			name:     "empty items",
			request:  nil,
			order:    nil,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := samePurchaseItems(
				tt.request,
				tt.order,
			)

			if got != tt.expected {
				t.Errorf(
					"expected %v, got %v",
					tt.expected,
					got,
				)
			}
		})
	}
}
