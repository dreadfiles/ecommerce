package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/big"

	"ecommerce/internal/purchase/domain"

	"github.com/lib/pq"
)

const postgresUniqueViolationCode = "23505"

type PostgresPurchaseRepository struct {
	db *sql.DB
}

func NewPostgresPurchaseRepository(db *sql.DB) *PostgresPurchaseRepository {
	return &PostgresPurchaseRepository{
		db: db,
	}
}

func (r *PostgresPurchaseRepository) GetQuote(
	ctx context.Context,
	items []domain.OrderItem,
) (*domain.Order, error) {
	if len(items) == 0 {
		return nil, errors.New("order must contain at least one item")
	}

	const getProductQuery = `
		SELECT price, stock
		FROM products
		WHERE id = $1
	`

	orderItems := make([]domain.OrderItem, 0, len(items))

	for _, requestedItem := range items {
		var price string
		var stock int

		err := r.db.QueryRowContext(
			ctx,
			getProductQuery,
			requestedItem.ProductID,
		).Scan(
			&price,
			&stock,
		)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf(
					"product %d: %w",
					requestedItem.ProductID,
					ErrProductNotFound,
				)
			}

			return nil, fmt.Errorf(
				"get product %d: %w",
				requestedItem.ProductID,
				err,
			)
		}

		if stock < requestedItem.Quantity {
			return nil, fmt.Errorf(
				"product %d: %w",
				requestedItem.ProductID,
				ErrInsufficientStock,
			)
		}

		unitPrice, err := parseMoney(price)
		if err != nil {
			return nil, fmt.Errorf(
				"invalid price for product %d: %w",
				requestedItem.ProductID,
				err,
			)
		}

		quantity := new(big.Rat).SetInt64(
			int64(requestedItem.Quantity),
		)

		subtotal := new(big.Rat).Mul(
			unitPrice,
			quantity,
		)

		orderItems = append(
			orderItems,
			domain.OrderItem{
				ProductID: requestedItem.ProductID,
				Quantity:  requestedItem.Quantity,
				UnitPrice: price,
				Subtotal:  formatMoney(subtotal),
			},
		)
	}

	total, err := calculateOrderTotal(orderItems)
	if err != nil {
		return nil, fmt.Errorf(
			"calculate order total: %w",
			err,
		)
	}

	return &domain.Order{
		Status: domain.OrderStatusPending,
		Total:  formatMoney(total),
		Items:  orderItems,
	}, nil
}

func (r *PostgresPurchaseRepository) Create(
	ctx context.Context,
	order *domain.Order,
) error {
	if order == nil {
		return errors.New("order cannot be nil")
	}

	if len(order.Items) == 0 {
		return errors.New("order must contain at least one item")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(
			"begin purchase transaction: %w",
			err,
		)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	const getProductQuery = `
		SELECT price, stock
		FROM products
		WHERE id = $1
		FOR UPDATE
	`

	const updateStockQuery = `
		UPDATE products
		SET
			stock = stock - $1,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $2
	`

	for i := range order.Items {
		item := &order.Items[i]

		var price string
		var stock int

		err := tx.QueryRowContext(
			ctx,
			getProductQuery,
			item.ProductID,
		).Scan(
			&price,
			&stock,
		)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf(
					"product %d: %w",
					item.ProductID,
					ErrProductNotFound,
				)
			}

			return fmt.Errorf(
				"get product %d: %w",
				item.ProductID,
				err,
			)
		}

		if stock < item.Quantity {
			return fmt.Errorf(
				"product %d: %w",
				item.ProductID,
				ErrInsufficientStock,
			)
		}

		unitPrice, err := parseMoney(price)
		if err != nil {
			return fmt.Errorf(
				"invalid price for product %d: %w",
				item.ProductID,
				err,
			)
		}

		quantity := new(big.Rat).SetInt64(
			int64(item.Quantity),
		)

		subtotal := new(big.Rat).Mul(
			unitPrice,
			quantity,
		)

		item.UnitPrice = price
		item.Subtotal = formatMoney(subtotal)

		_, err = tx.ExecContext(
			ctx,
			updateStockQuery,
			item.Quantity,
			item.ProductID,
		)
		if err != nil {
			return fmt.Errorf(
				"update stock for product %d: %w",
				item.ProductID,
				err,
			)
		}
	}

	total, err := calculateOrderTotal(order.Items)
	if err != nil {
		return fmt.Errorf(
			"calculate order total: %w",
			err,
		)
	}

	order.Total = formatMoney(total)

	const createOrderQuery = `
		INSERT INTO orders (
			status,
			total,
			idempotency_key
		)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	err = tx.QueryRowContext(
		ctx,
		createOrderQuery,
		order.Status,
		order.Total,
		order.IdempotencyKey,
	).Scan(&order.ID)

	if err != nil {
		var pqErr *pq.Error

		if errors.As(err, &pqErr) &&
			pqErr.Code == postgresUniqueViolationCode {
			return ErrIdempotencyKeyExists
		}

		return fmt.Errorf(
			"create order: %w",
			err,
		)
	}

	const createOrderItemQuery = `
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

	for i := range order.Items {
		item := &order.Items[i]

		err = tx.QueryRowContext(
			ctx,
			createOrderItemQuery,
			order.ID,
			item.ProductID,
			item.Quantity,
			item.UnitPrice,
			item.Subtotal,
		).Scan(&item.ID)

		if err != nil {
			return fmt.Errorf(
				"create order item for product %d: %w",
				item.ProductID,
				err,
			)
		}

		item.OrderID = order.ID
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(
			"commit purchase transaction: %w",
			err,
		)
	}

	return nil
}

func (r *PostgresPurchaseRepository) GetByIdempotencyKey(
	ctx context.Context,
	key string,
) (*domain.Order, error) {
	const getOrderQuery = `
		SELECT
			id,
			status,
			total,
			idempotency_key
		FROM orders
		WHERE idempotency_key = $1
	`

	var order domain.Order

	err := r.db.QueryRowContext(
		ctx,
		getOrderQuery,
		key,
	).Scan(
		&order.ID,
		&order.Status,
		&order.Total,
		&order.IdempotencyKey,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}

		return nil, fmt.Errorf(
			"get order by idempotency key: %w",
			err,
		)
	}

	if err := r.loadOrderItems(
		ctx,
		&order,
	); err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *PostgresPurchaseRepository) GetByID(
	ctx context.Context,
	id int64,
) (*domain.Order, error) {
	const getOrderQuery = `
		SELECT
			id,
			status,
			total,
			idempotency_key
		FROM orders
		WHERE id = $1
	`

	var order domain.Order

	err := r.db.QueryRowContext(
		ctx,
		getOrderQuery,
		id,
	).Scan(
		&order.ID,
		&order.Status,
		&order.Total,
		&order.IdempotencyKey,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOrderNotFound
		}

		return nil, fmt.Errorf(
			"get order by id: %w",
			err,
		)
	}

	if err := r.loadOrderItems(
		ctx,
		&order,
	); err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *PostgresPurchaseRepository) GetAll(
	ctx context.Context,
) ([]*domain.Order, error) {
	const getOrdersQuery = `
		SELECT
			id,
			status,
			total,
			idempotency_key
		FROM orders
		ORDER BY id DESC
	`

	rows, err := r.db.QueryContext(
		ctx,
		getOrdersQuery,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"get purchases: %w",
			err,
		)
	}
	defer rows.Close()

	orders := make([]*domain.Order, 0)

	for rows.Next() {
		var order domain.Order

		if err := rows.Scan(
			&order.ID,
			&order.Status,
			&order.Total,
			&order.IdempotencyKey,
		); err != nil {
			return nil, fmt.Errorf(
				"scan purchase: %w",
				err,
			)
		}

		if err := r.loadOrderItems(
			ctx,
			&order,
		); err != nil {
			return nil, err
		}

		orders = append(
			orders,
			&order,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate purchases: %w",
			err,
		)
	}

	return orders, nil
}

func (r *PostgresPurchaseRepository) loadOrderItems(
	ctx context.Context,
	order *domain.Order,
) error {
	const getOrderItemsQuery = `
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

	rows, err := r.db.QueryContext(
		ctx,
		getOrderItemsQuery,
		order.ID,
	)
	if err != nil {
		return fmt.Errorf(
			"get order items: %w",
			err,
		)
	}
	defer rows.Close()

	order.Items = make(
		[]domain.OrderItem,
		0,
	)

	for rows.Next() {
		var item domain.OrderItem

		if err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.ProductID,
			&item.Quantity,
			&item.UnitPrice,
			&item.Subtotal,
		); err != nil {
			return fmt.Errorf(
				"scan order item: %w",
				err,
			)
		}

		order.Items = append(
			order.Items,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf(
			"iterate order items: %w",
			err,
		)
	}

	return nil
}

func calculateOrderTotal(
	items []domain.OrderItem,
) (*big.Rat, error) {
	total := new(big.Rat)

	for _, item := range items {
		subtotal, err := parseMoney(item.Subtotal)
		if err != nil {
			return nil, err
		}

		total.Add(total, subtotal)
	}

	return total, nil
}

func parseMoney(value string) (*big.Rat, error) {
	rat := new(big.Rat)

	if _, ok := rat.SetString(value); !ok {
		return nil, errors.New("invalid monetary value")
	}

	if rat.Sign() < 0 {
		return nil, errors.New("monetary value cannot be negative")
	}

	return rat, nil
}

func formatMoney(value *big.Rat) string {
	return value.FloatString(2)
}
