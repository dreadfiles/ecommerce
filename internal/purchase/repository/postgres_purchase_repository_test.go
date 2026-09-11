package repository

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"testing"

	"ecommerce/internal/purchase/domain"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
)

func newPurchaseRepositoryTest(t *testing.T) (
	*PostgresPurchaseRepository,
	sqlmock.Sqlmock,
	func(),
) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	repo := NewPostgresPurchaseRepository(db)

	cleanup := func() {
		_ = db.Close()
	}

	return repo, mock, cleanup
}

func queryRegex(query string) string {
	return regexp.QuoteMeta(
		strings.Join(strings.Fields(query), " "),
	)
}

func TestNewPostgresPurchaseRepository(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPostgresPurchaseRepository(db)

	if repo == nil {
		t.Fatal("expected repository, got nil")
	}

	if repo.db != db {
		t.Fatal("expected repository to use provided database")
	}
}

func TestPostgresPurchaseRepository_GetQuote(t *testing.T) {
	const query = `
		SELECT price, stock
		FROM products
		WHERE id = $1
	`

	t.Run("empty items", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		order, err := repo.GetQuote(
			context.Background(),
			nil,
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if order != nil {
			t.Fatal("expected nil order")
		}

		if err.Error() != "order must contain at least one item" {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		items := []domain.OrderItem{
			{
				ProductID: 1,
				Quantity:  2,
			},
			{
				ProductID: 2,
				Quantity:  3,
			},
		}

		mock.ExpectQuery(queryRegex(query)).
			WithArgs(int64(1)).
			WillReturnRows(
				sqlmock.NewRows([]string{"price", "stock"}).
					AddRow("100.00", 10),
			)

		mock.ExpectQuery(queryRegex(query)).
			WithArgs(int64(2)).
			WillReturnRows(
				sqlmock.NewRows([]string{"price", "stock"}).
					AddRow("25.50", 10),
			)

		order, err := repo.GetQuote(
			context.Background(),
			items,
		)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if order == nil {
			t.Fatal("expected order, got nil")
		}

		if order.Status != domain.OrderStatusPending {
			t.Errorf(
				"expected status %s, got %s",
				domain.OrderStatusPending,
				order.Status,
			)
		}

		if order.Total != "276.50" {
			t.Errorf(
				"expected total 276.50, got %s",
				order.Total,
			)
		}

		if len(order.Items) != 2 {
			t.Fatalf(
				"expected 2 items, got %d",
				len(order.Items),
			)
		}

		if order.Items[0].UnitPrice != "100.00" {
			t.Errorf(
				"expected unit price 100.00, got %s",
				order.Items[0].UnitPrice,
			)
		}

		if order.Items[0].Subtotal != "200.00" {
			t.Errorf(
				"expected subtotal 200.00, got %s",
				order.Items[0].Subtotal,
			)
		}

		if order.Items[1].Subtotal != "76.50" {
			t.Errorf(
				"expected subtotal 76.50, got %s",
				order.Items[1].Subtotal,
			)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("product not found", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		mock.ExpectQuery(queryRegex(query)).
			WithArgs(int64(999)).
			WillReturnError(sql.ErrNoRows)

		order, err := repo.GetQuote(
			context.Background(),
			[]domain.OrderItem{
				{
					ProductID: 999,
					Quantity:  1,
				},
			},
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if !errors.Is(err, ErrProductNotFound) {
			t.Fatalf(
				"expected ErrProductNotFound, got %v",
				err,
			)
		}

		if order != nil {
			t.Fatal("expected nil order")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("database error", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		mock.ExpectQuery(queryRegex(query)).
			WithArgs(int64(1)).
			WillReturnError(errors.New("database error"))

		order, err := repo.GetQuote(
			context.Background(),
			[]domain.OrderItem{
				{
					ProductID: 1,
					Quantity:  1,
				},
			},
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if order != nil {
			t.Fatal("expected nil order")
		}

		if err.Error() != "get product 1: database error" {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("insufficient stock", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		mock.ExpectQuery(queryRegex(query)).
			WithArgs(int64(1)).
			WillReturnRows(
				sqlmock.NewRows([]string{"price", "stock"}).
					AddRow("100.00", 2),
			)

		order, err := repo.GetQuote(
			context.Background(),
			[]domain.OrderItem{
				{
					ProductID: 1,
					Quantity:  5,
				},
			},
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if !errors.Is(err, ErrInsufficientStock) {
			t.Fatalf(
				"expected ErrInsufficientStock, got %v",
				err,
			)
		}

		if order != nil {
			t.Fatal("expected nil order")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("invalid price", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		mock.ExpectQuery(queryRegex(query)).
			WithArgs(int64(1)).
			WillReturnRows(
				sqlmock.NewRows([]string{"price", "stock"}).
					AddRow("invalid", 10),
			)

		order, err := repo.GetQuote(
			context.Background(),
			[]domain.OrderItem{
				{
					ProductID: 1,
					Quantity:  1,
				},
			},
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if order != nil {
			t.Fatal("expected nil order")
		}

		if err.Error() == "" {
			t.Fatal("expected invalid price error")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestPostgresPurchaseRepository_Create(t *testing.T) {
	getProductQuery := `
		SELECT price, stock
		FROM products
		WHERE id = $1
		FOR UPDATE
	`

	updateStockQuery := `
		UPDATE products
		SET
			stock = stock - $1,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`

	createOrderQuery := `
		INSERT INTO orders (
			status,
			total,
			idempotency_key
		)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	createOrderItemQuery := `
		INSERT INTO order_items (
			order_id,
			product_id,
			quantity,
			unit_price,
			subtotal
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`

	t.Run("nil order", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		err := repo.Create(
			context.Background(),
			nil,
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if err.Error() != "order cannot be nil" {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("empty items", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		err := repo.Create(
			context.Background(),
			&domain.Order{},
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if err.Error() != "order must contain at least one item" {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("begin transaction error", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		mock.ExpectBegin().
			WillReturnError(errors.New("begin error"))

		order := &domain.Order{
			Items: []domain.OrderItem{
				{
					ProductID: 1,
					Quantity:  1,
				},
			},
		}

		err := repo.Create(
			context.Background(),
			order,
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if err.Error() != "begin purchase transaction: begin error" {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("success", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		order := &domain.Order{
			Status:         domain.OrderStatusPaid,
			IdempotencyKey: "key-001",
			Items: []domain.OrderItem{
				{
					ProductID: 1,
					Quantity:  2,
				},
			},
		}

		mock.ExpectBegin()

		mock.ExpectQuery(queryRegex(getProductQuery)).
			WithArgs(int64(1)).
			WillReturnRows(
				sqlmock.NewRows([]string{"price", "stock"}).
					AddRow("100.00", 10),
			)

		mock.ExpectExec(queryRegex(updateStockQuery)).
			WithArgs(2, int64(1)).
			WillReturnResult(
				sqlmock.NewResult(0, 1),
			)

		mock.ExpectQuery(queryRegex(createOrderQuery)).
			WithArgs(
				domain.OrderStatusPaid,
				"200.00",
				"key-001",
			).
			WillReturnRows(
				sqlmock.NewRows([]string{"id"}).
					AddRow(int64(100)),
			)

		mock.ExpectQuery(queryRegex(createOrderItemQuery)).
			WithArgs(
				int64(100),
				int64(1),
				2,
				"100.00",
				"200.00",
			).
			WillReturnRows(
				sqlmock.NewRows([]string{"id"}).
					AddRow(int64(500)),
			)

		mock.ExpectCommit()

		err := repo.Create(
			context.Background(),
			order,
		)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if order.ID != 100 {
			t.Errorf("expected order ID 100, got %d", order.ID)
		}

		if order.Total != "200.00" {
			t.Errorf(
				"expected total 200.00, got %s",
				order.Total,
			)
		}

		if order.Items[0].ID != 500 {
			t.Errorf(
				"expected item ID 500, got %d",
				order.Items[0].ID,
			)
		}

		if order.Items[0].OrderID != 100 {
			t.Errorf(
				"expected item order ID 100, got %d",
				order.Items[0].OrderID,
			)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("product not found", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		order := &domain.Order{
			Items: []domain.OrderItem{
				{
					ProductID: 999,
					Quantity:  1,
				},
			},
		}

		mock.ExpectBegin()

		mock.ExpectQuery(queryRegex(getProductQuery)).
			WithArgs(int64(999)).
			WillReturnError(sql.ErrNoRows)

		err := repo.Create(
			context.Background(),
			order,
		)

		if !errors.Is(err, ErrProductNotFound) {
			t.Fatalf(
				"expected ErrProductNotFound, got %v",
				err,
			)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("product query error", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		order := &domain.Order{
			Items: []domain.OrderItem{
				{
					ProductID: 1,
					Quantity:  1,
				},
			},
		}

		mock.ExpectBegin()

		mock.ExpectQuery(queryRegex(getProductQuery)).
			WithArgs(int64(1)).
			WillReturnError(errors.New("query error"))

		err := repo.Create(
			context.Background(),
			order,
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if err.Error() != "get product 1: query error" {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("insufficient stock", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		order := &domain.Order{
			Items: []domain.OrderItem{
				{
					ProductID: 1,
					Quantity:  10,
				},
			},
		}

		mock.ExpectBegin()

		mock.ExpectQuery(queryRegex(getProductQuery)).
			WithArgs(int64(1)).
			WillReturnRows(
				sqlmock.NewRows([]string{"price", "stock"}).
					AddRow("100.00", 5),
			)

		err := repo.Create(
			context.Background(),
			order,
		)

		if !errors.Is(err, ErrInsufficientStock) {
			t.Fatalf(
				"expected ErrInsufficientStock, got %v",
				err,
			)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("invalid price", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		order := &domain.Order{
			Items: []domain.OrderItem{
				{
					ProductID: 1,
					Quantity:  1,
				},
			},
		}

		mock.ExpectBegin()

		mock.ExpectQuery(queryRegex(getProductQuery)).
			WithArgs(int64(1)).
			WillReturnRows(
				sqlmock.NewRows([]string{"price", "stock"}).
					AddRow("invalid", 10),
			)

		err := repo.Create(
			context.Background(),
			order,
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if err.Error() == "" {
			t.Fatal("expected invalid price error")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("update stock error", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		order := &domain.Order{
			Items: []domain.OrderItem{
				{
					ProductID: 1,
					Quantity:  2,
				},
			},
		}

		mock.ExpectBegin()

		mock.ExpectQuery(queryRegex(getProductQuery)).
			WithArgs(int64(1)).
			WillReturnRows(
				sqlmock.NewRows([]string{"price", "stock"}).
					AddRow("100.00", 10),
			)

		mock.ExpectExec(queryRegex(updateStockQuery)).
			WithArgs(2, int64(1)).
			WillReturnError(errors.New("update stock error"))

		err := repo.Create(
			context.Background(),
			order,
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if err.Error() != "update stock for product 1: update stock error" {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("idempotency conflict", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		order := &domain.Order{
			Status:         domain.OrderStatusPaid,
			IdempotencyKey: "key-001",
			Items: []domain.OrderItem{
				{
					ProductID: 1,
					Quantity:  1,
				},
			},
		}

		mock.ExpectBegin()

		mock.ExpectQuery(queryRegex(getProductQuery)).
			WithArgs(int64(1)).
			WillReturnRows(
				sqlmock.NewRows([]string{"price", "stock"}).
					AddRow("100.00", 10),
			)

		mock.ExpectExec(queryRegex(updateStockQuery)).
			WithArgs(1, int64(1)).
			WillReturnResult(
				sqlmock.NewResult(0, 1),
			)

		mock.ExpectQuery(queryRegex(createOrderQuery)).
			WithArgs(
				domain.OrderStatusPaid,
				"100.00",
				"key-001",
			).
			WillReturnError(
				&pq.Error{Code: "23505"},
			)

		err := repo.Create(
			context.Background(),
			order,
		)

		if !errors.Is(err, ErrIdempotencyKeyExists) {
			t.Fatalf(
				"expected ErrIdempotencyKeyExists, got %v",
				err,
			)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("create order error", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		order := &domain.Order{
			Status:         domain.OrderStatusPaid,
			IdempotencyKey: "key-001",
			Items: []domain.OrderItem{
				{
					ProductID: 1,
					Quantity:  1,
				},
			},
		}

		mock.ExpectBegin()

		mock.ExpectQuery(queryRegex(getProductQuery)).
			WithArgs(int64(1)).
			WillReturnRows(
				sqlmock.NewRows([]string{"price", "stock"}).
					AddRow("100.00", 10),
			)

		mock.ExpectExec(queryRegex(updateStockQuery)).
			WithArgs(1, int64(1)).
			WillReturnResult(
				sqlmock.NewResult(0, 1),
			)

		mock.ExpectQuery(queryRegex(createOrderQuery)).
			WillReturnError(errors.New("create order error"))

		err := repo.Create(
			context.Background(),
			order,
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if err.Error() != "create order: create order error" {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("create order item error", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		order := &domain.Order{
			Status:         domain.OrderStatusPaid,
			IdempotencyKey: "key-001",
			Items: []domain.OrderItem{
				{
					ProductID: 1,
					Quantity:  1,
				},
			},
		}

		mock.ExpectBegin()

		mock.ExpectQuery(queryRegex(getProductQuery)).
			WithArgs(int64(1)).
			WillReturnRows(
				sqlmock.NewRows([]string{"price", "stock"}).
					AddRow("100.00", 10),
			)

		mock.ExpectExec(queryRegex(updateStockQuery)).
			WithArgs(1, int64(1)).
			WillReturnResult(
				sqlmock.NewResult(0, 1),
			)

		mock.ExpectQuery(queryRegex(createOrderQuery)).
			WithArgs(
				domain.OrderStatusPaid,
				"100.00",
				"key-001",
			).
			WillReturnRows(
				sqlmock.NewRows([]string{"id"}).
					AddRow(int64(100)),
			)

		mock.ExpectQuery(queryRegex(createOrderItemQuery)).
			WithArgs(
				int64(100),
				int64(1),
				1,
				"100.00",
				"100.00",
			).
			WillReturnError(errors.New("item error"))

		err := repo.Create(
			context.Background(),
			order,
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if err.Error() !=
			"create order item for product 1: item error" {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("commit error", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		order := &domain.Order{
			Status:         domain.OrderStatusPaid,
			IdempotencyKey: "key-001",
			Items: []domain.OrderItem{
				{
					ProductID: 1,
					Quantity:  1,
				},
			},
		}

		mock.ExpectBegin()

		mock.ExpectQuery(queryRegex(getProductQuery)).
			WithArgs(int64(1)).
			WillReturnRows(
				sqlmock.NewRows([]string{"price", "stock"}).
					AddRow("100.00", 10),
			)

		mock.ExpectExec(queryRegex(updateStockQuery)).
			WithArgs(1, int64(1)).
			WillReturnResult(
				sqlmock.NewResult(0, 1),
			)

		mock.ExpectQuery(queryRegex(createOrderQuery)).
			WithArgs(
				domain.OrderStatusPaid,
				"100.00",
				"key-001",
			).
			WillReturnRows(
				sqlmock.NewRows([]string{"id"}).
					AddRow(int64(100)),
			)

		mock.ExpectQuery(queryRegex(createOrderItemQuery)).
			WithArgs(
				int64(100),
				int64(1),
				1,
				"100.00",
				"100.00",
			).
			WillReturnRows(
				sqlmock.NewRows([]string{"id"}).
					AddRow(int64(500)),
			)

		mock.ExpectCommit().
			WillReturnError(errors.New("commit error"))

		err := repo.Create(
			context.Background(),
			order,
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if err.Error() != "commit purchase transaction: commit error" {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestPostgresPurchaseRepository_GetByIdempotencyKey(t *testing.T) {
	orderQuery := `
		SELECT
			id,
			status,
			total,
			idempotency_key
		FROM orders
		WHERE idempotency_key = $1
	`

	itemQuery := `
		SELECT
			id,
			order_id,
			product_id,
			quantity,
			unit_price,
			subtotal
		FROM order_items
		WHERE order_id = $1
		ORDER BY id
	`

	t.Run("success", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		mock.ExpectQuery(queryRegex(orderQuery)).
			WithArgs("key-001").
			WillReturnRows(
				sqlmock.NewRows([]string{
					"id",
					"status",
					"total",
					"idempotency_key",
				}).
					AddRow(
						100,
						"PAID",
						"200.00",
						"key-001",
					),
			)

		mock.ExpectQuery(queryRegex(itemQuery)).
			WithArgs(int64(100)).
			WillReturnRows(
				sqlmock.NewRows([]string{
					"id",
					"order_id",
					"product_id",
					"quantity",
					"unit_price",
					"subtotal",
				}).
					AddRow(
						500,
						100,
						1,
						2,
						"100.00",
						"200.00",
					),
			)

		order, err := repo.GetByIdempotencyKey(
			context.Background(),
			"key-001",
		)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if order.ID != 100 {
			t.Errorf("expected ID 100, got %d", order.ID)
		}

		if len(order.Items) != 1 {
			t.Fatalf(
				"expected 1 item, got %d",
				len(order.Items),
			)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		mock.ExpectQuery(queryRegex(orderQuery)).
			WithArgs("missing").
			WillReturnError(sql.ErrNoRows)

		order, err := repo.GetByIdempotencyKey(
			context.Background(),
			"missing",
		)

		if !errors.Is(err, ErrOrderNotFound) {
			t.Fatalf(
				"expected ErrOrderNotFound, got %v",
				err,
			)
		}

		if order != nil {
			t.Fatal("expected nil order")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("database error", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		mock.ExpectQuery(queryRegex(orderQuery)).
			WithArgs("key-001").
			WillReturnError(errors.New("database error"))

		order, err := repo.GetByIdempotencyKey(
			context.Background(),
			"key-001",
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if order != nil {
			t.Fatal("expected nil order")
		}

		if err.Error() !=
			"get order by idempotency key: database error" {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("load order items error", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		mock.ExpectQuery(queryRegex(orderQuery)).
			WithArgs("key-001").
			WillReturnRows(
				sqlmock.NewRows([]string{
					"id",
					"status",
					"total",
					"idempotency_key",
				}).
					AddRow(
						100,
						"PAID",
						"200.00",
						"key-001",
					),
			)

		mock.ExpectQuery(queryRegex(itemQuery)).
			WithArgs(int64(100)).
			WillReturnError(errors.New("load items error"))

		order, err := repo.GetByIdempotencyKey(
			context.Background(),
			"key-001",
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if order != nil {
			t.Fatal("expected nil order")
		}

		if err.Error() != "get order items: load items error" {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestPostgresPurchaseRepository_GetByID(t *testing.T) {
	orderQuery := `
		SELECT
			id,
			status,
			total,
			idempotency_key
		FROM orders
		WHERE id = $1
	`

	itemQuery := `
		SELECT
			id,
			order_id,
			product_id,
			quantity,
			unit_price,
			subtotal
		FROM order_items
		WHERE order_id = $1
		ORDER BY id
	`

	t.Run("success", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		mock.ExpectQuery(queryRegex(orderQuery)).
			WithArgs(int64(100)).
			WillReturnRows(
				sqlmock.NewRows([]string{
					"id",
					"status",
					"total",
					"idempotency_key",
				}).
					AddRow(
						100,
						"PAID",
						"200.00",
						"key-001",
					),
			)

		mock.ExpectQuery(queryRegex(itemQuery)).
			WithArgs(int64(100)).
			WillReturnRows(
				sqlmock.NewRows([]string{
					"id",
					"order_id",
					"product_id",
					"quantity",
					"unit_price",
					"subtotal",
				}).
					AddRow(
						500,
						100,
						1,
						2,
						"100.00",
						"200.00",
					),
			)

		order, err := repo.GetByID(
			context.Background(),
			100,
		)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if order.ID != 100 {
			t.Errorf("expected ID 100, got %d", order.ID)
		}

		if len(order.Items) != 1 {
			t.Fatalf(
				"expected 1 item, got %d",
				len(order.Items),
			)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		mock.ExpectQuery(queryRegex(orderQuery)).
			WithArgs(int64(999)).
			WillReturnError(sql.ErrNoRows)

		order, err := repo.GetByID(
			context.Background(),
			999,
		)

		if !errors.Is(err, ErrOrderNotFound) {
			t.Fatalf(
				"expected ErrOrderNotFound, got %v",
				err,
			)
		}

		if order != nil {
			t.Fatal("expected nil order")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("database error", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		mock.ExpectQuery(queryRegex(orderQuery)).
			WithArgs(int64(100)).
			WillReturnError(errors.New("database error"))

		order, err := repo.GetByID(
			context.Background(),
			100,
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if order != nil {
			t.Fatal("expected nil order")
		}

		if err.Error() != "get order by id: database error" {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("load order items error", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		mock.ExpectQuery(queryRegex(orderQuery)).
			WithArgs(int64(100)).
			WillReturnRows(
				sqlmock.NewRows([]string{
					"id",
					"status",
					"total",
					"idempotency_key",
				}).
					AddRow(
						100,
						"PAID",
						"200.00",
						"key-001",
					),
			)

		mock.ExpectQuery(queryRegex(itemQuery)).
			WithArgs(int64(100)).
			WillReturnError(errors.New("load items error"))

		order, err := repo.GetByID(
			context.Background(),
			100,
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if order != nil {
			t.Fatal("expected nil order")
		}

		if err.Error() != "get order items: load items error" {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestPostgresPurchaseRepository_GetAll(t *testing.T) {
	ordersQuery := `
		SELECT
			id,
			status,
			total,
			idempotency_key
		FROM orders
		ORDER BY id DESC
	`

	itemQuery := `
		SELECT
			id,
			order_id,
			product_id,
			quantity,
			unit_price,
			subtotal
		FROM order_items
		WHERE order_id = $1
		ORDER BY id
	`

	t.Run("success", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		mock.ExpectQuery(queryRegex(ordersQuery)).
			WillReturnRows(
				sqlmock.NewRows([]string{
					"id",
					"status",
					"total",
					"idempotency_key",
				}).
					AddRow(
						100,
						"PAID",
						"200.00",
						"key-001",
					),
			)

		mock.ExpectQuery(queryRegex(itemQuery)).
			WithArgs(int64(100)).
			WillReturnRows(
				sqlmock.NewRows([]string{
					"id",
					"order_id",
					"product_id",
					"quantity",
					"unit_price",
					"subtotal",
				}).
					AddRow(
						500,
						100,
						1,
						2,
						"100.00",
						"200.00",
					),
			)

		orders, err := repo.GetAll(
			context.Background(),
		)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(orders) != 1 {
			t.Fatalf(
				"expected 1 order, got %d",
				len(orders),
			)
		}

		if len(orders[0].Items) != 1 {
			t.Fatalf(
				"expected 1 item, got %d",
				len(orders[0].Items),
			)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("query error", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		mock.ExpectQuery(queryRegex(ordersQuery)).
			WillReturnError(errors.New("database error"))

		orders, err := repo.GetAll(
			context.Background(),
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if orders != nil {
			t.Fatal("expected nil orders")
		}

		if err.Error() != "get purchases: database error" {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("scan error", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		mock.ExpectQuery(queryRegex(ordersQuery)).
			WillReturnRows(
				sqlmock.NewRows([]string{
					"id",
					"status",
					"total",
					"idempotency_key",
				}).
					AddRow(
						"invalid-id",
						"PAID",
						"200.00",
						"key-001",
					),
			)

		orders, err := repo.GetAll(
			context.Background(),
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if orders != nil {
			t.Fatal("expected nil orders")
		}

		if err.Error() == "" {
			t.Fatal("expected scan error")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("load order items error", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		mock.ExpectQuery(queryRegex(ordersQuery)).
			WillReturnRows(
				sqlmock.NewRows([]string{
					"id",
					"status",
					"total",
					"idempotency_key",
				}).
					AddRow(
						100,
						"PAID",
						"200.00",
						"key-001",
					),
			)

		mock.ExpectQuery(queryRegex(itemQuery)).
			WithArgs(int64(100)).
			WillReturnError(errors.New("load items error"))

		orders, err := repo.GetAll(
			context.Background(),
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if orders != nil {
			t.Fatal("expected nil orders")
		}

		if err.Error() != "get order items: load items error" {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("rows error", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		rows := sqlmock.NewRows([]string{
			"id",
			"status",
			"total",
			"idempotency_key",
		}).
			AddRow(
				100,
				"PAID",
				"200.00",
				"key-001",
			).
			RowError(0, errors.New("row iteration error"))

		mock.ExpectQuery(queryRegex(ordersQuery)).
			WillReturnRows(rows)

		orders, err := repo.GetAll(
			context.Background(),
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if orders != nil {
			t.Fatal("expected nil orders")
		}

		if err.Error() != "iterate purchases: row iteration error" {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestPostgresPurchaseRepository_loadOrderItems(t *testing.T) {
	query := `
		SELECT
			id,
			order_id,
			product_id,
			quantity,
			unit_price,
			subtotal
		FROM order_items
		WHERE order_id = $1
		ORDER BY id
	`

	t.Run("success", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		order := &domain.Order{
			ID: 100,
		}

		mock.ExpectQuery(queryRegex(query)).
			WithArgs(int64(100)).
			WillReturnRows(
				sqlmock.NewRows([]string{
					"id",
					"order_id",
					"product_id",
					"quantity",
					"unit_price",
					"subtotal",
				}).
					AddRow(
						500,
						100,
						1,
						2,
						"100.00",
						"200.00",
					).
					AddRow(
						501,
						100,
						2,
						1,
						"50.00",
						"50.00",
					),
			)

		err := repo.loadOrderItems(
			context.Background(),
			order,
		)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(order.Items) != 2 {
			t.Fatalf(
				"expected 2 items, got %d",
				len(order.Items),
			)
		}

		if order.Items[0].ProductID != 1 {
			t.Errorf(
				"expected product ID 1, got %d",
				order.Items[0].ProductID,
			)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("query error", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		order := &domain.Order{
			ID: 100,
		}

		mock.ExpectQuery(queryRegex(query)).
			WithArgs(int64(100)).
			WillReturnError(errors.New("query error"))

		err := repo.loadOrderItems(
			context.Background(),
			order,
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if err.Error() != "get order items: query error" {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("scan error", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		order := &domain.Order{
			ID: 100,
		}

		mock.ExpectQuery(queryRegex(query)).
			WithArgs(int64(100)).
			WillReturnRows(
				sqlmock.NewRows([]string{
					"id",
					"order_id",
					"product_id",
					"quantity",
					"unit_price",
					"subtotal",
				}).
					AddRow(
						"invalid-id",
						100,
						1,
						2,
						"100.00",
						"200.00",
					),
			)

		err := repo.loadOrderItems(
			context.Background(),
			order,
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if err.Error() == "" {
			t.Fatal("expected scan error")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})

	t.Run("rows error", func(t *testing.T) {
		repo, mock, cleanup := newPurchaseRepositoryTest(t)
		defer cleanup()

		order := &domain.Order{
			ID: 100,
		}

		rows := sqlmock.NewRows([]string{
			"id",
			"order_id",
			"product_id",
			"quantity",
			"unit_price",
			"subtotal",
		}).
			AddRow(
				500,
				100,
				1,
				2,
				"100.00",
				"200.00",
			).
			RowError(0, errors.New("row iteration error"))

		mock.ExpectQuery(queryRegex(query)).
			WithArgs(int64(100)).
			WillReturnRows(rows)

		err := repo.loadOrderItems(
			context.Background(),
			order,
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if err.Error() != "iterate order items: row iteration error" {
			t.Fatalf("unexpected error: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet expectations: %v", err)
		}
	})
}

func TestCalculateOrderTotal(t *testing.T) {
	tests := []struct {
		name    string
		items   []domain.OrderItem
		want    string
		wantErr bool
	}{
		{
			name: "multiple items",
			items: []domain.OrderItem{
				{Subtotal: "100.00"},
				{Subtotal: "25.50"},
				{Subtotal: "10.25"},
			},
			want: "135.75",
		},
		{
			name:  "empty items",
			items: nil,
			want:  "0.00",
		},
		{
			name: "invalid subtotal",
			items: []domain.OrderItem{
				{Subtotal: "invalid"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			total, err := calculateOrderTotal(tt.items)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if total.FloatString(2) != tt.want {
				t.Fatalf(
					"expected %s, got %s",
					tt.want,
					total.FloatString(2),
				)
			}
		})
	}
}

func TestParseMoney(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{
			name:  "integer",
			value: "100",
			want:  "100.00",
		},
		{
			name:  "decimal",
			value: "100.50",
			want:  "100.50",
		},
		{
			name:  "zero",
			value: "0",
			want:  "0.00",
		},
		{
			name:    "invalid",
			value:   "abc",
			wantErr: true,
		},
		{
			name:    "negative",
			value:   "-10",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := parseMoney(tt.value)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if value.FloatString(2) != tt.want {
				t.Fatalf(
					"expected %s, got %s",
					tt.want,
					value.FloatString(2),
				)
			}
		})
	}
}

func TestFormatMoney(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "integer",
			in:   "100",
			want: "100.00",
		},
		{
			name: "decimal",
			in:   "100.50",
			want: "100.50",
		},
		{
			name: "rounding",
			in:   "100.555",
			want: "100.56",
		},
		{
			name: "zero",
			in:   "0",
			want: "0.00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := parseMoney(tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := formatMoney(value)

			if got != tt.want {
				t.Fatalf(
					"expected %s, got %s",
					tt.want,
					got,
				)
			}
		})
	}
}
