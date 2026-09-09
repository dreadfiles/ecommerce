package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"ecommerce/internal/product/domain"

	"github.com/lib/pq"
)

const postgresUniqueViolationCode = "23505"

type PostgresProductRepository struct {
	db *sql.DB
}

func NewPostgresProductRepository(db *sql.DB) *PostgresProductRepository {
	return &PostgresProductRepository{
		db: db,
	}
}

func (r *PostgresProductRepository) Create(
	ctx context.Context,
	product *domain.Product,
) error {
	if product == nil {
		return errors.New("product cannot be nil")
	}

	const query = `
		INSERT INTO products (
			name,
			sku,
			description,
			category,
			price,
			stock,
			weight_kg
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		product.Name,
		product.SKU,
		product.Description,
		product.Category,
		product.Price,
		product.Stock,
		product.WeightKg,
	).Scan(&product.ID)

	if err != nil {
		var pqErr *pq.Error

		if errors.As(err, &pqErr) &&
			pqErr.Code == postgresUniqueViolationCode {
			return ErrConflict
		}

		return fmt.Errorf("create product: %w", err)
	}

	return nil
}

func (r *PostgresProductRepository) GetByID(
	ctx context.Context,
	id int64,
) (*domain.Product, error) {
	const query = `
		SELECT
			id,
			name,
			sku,
			description,
			category,
			price,
			stock,
			weight_kg
		FROM products
		WHERE id = $1
	`

	product := &domain.Product{}

	err := r.db.QueryRowContext(
		ctx,
		query,
		id,
	).Scan(
		&product.ID,
		&product.Name,
		&product.SKU,
		&product.Description,
		&product.Category,
		&product.Price,
		&product.Stock,
		&product.WeightKg,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("get product by id: %w", err)
	}

	return product, nil
}

func (r *PostgresProductRepository) GetAll(
	ctx context.Context,
) ([]*domain.Product, error) {
	const query = `
		SELECT
			id,
			name,
			sku,
			description,
			category,
			price,
			stock,
			weight_kg
		FROM products
		ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get all products: %w", err)
	}
	defer rows.Close()

	products := make([]*domain.Product, 0)

	for rows.Next() {
		product := &domain.Product{}

		if err := rows.Scan(
			&product.ID,
			&product.Name,
			&product.SKU,
			&product.Description,
			&product.Category,
			&product.Price,
			&product.Stock,
			&product.WeightKg,
		); err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}

		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate products: %w", err)
	}

	return products, nil
}

func (r *PostgresProductRepository) Update(
	ctx context.Context,
	product *domain.Product,
) error {
	if product == nil {
		return errors.New("product cannot be nil")
	}

	const query = `
		UPDATE products
		SET
			name = $1,
			sku = $2,
			description = $3,
			category = $4,
			price = $5,
			stock = $6,
			weight_kg = $7,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $8
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		product.Name,
		product.SKU,
		product.Description,
		product.Category,
		product.Price,
		product.Stock,
		product.WeightKg,
		product.ID,
	)

	if err != nil {
		var pqErr *pq.Error

		if errors.As(err, &pqErr) &&
			pqErr.Code == postgresUniqueViolationCode {
			return ErrConflict
		}

		return fmt.Errorf("update product: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *PostgresProductRepository) Delete(
	ctx context.Context,
	id int64,
) error {
	const query = `
		DELETE FROM products
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *PostgresProductRepository) CreateMany(
	ctx context.Context,
	products []*domain.Product,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin product import transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	const query = `
		INSERT INTO products (
			name,
			sku,
			description,
			category,
			price,
			stock,
			weight_kg
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	for _, product := range products {
		if product == nil {
			return errors.New("product cannot be nil")
		}

		err := tx.QueryRowContext(
			ctx,
			query,
			product.Name,
			product.SKU,
			product.Description,
			product.Category,
			product.Price,
			product.Stock,
			product.WeightKg,
		).Scan(&product.ID)

		if err != nil {
			var pqErr *pq.Error

			if errors.As(err, &pqErr) &&
				pqErr.Code == postgresUniqueViolationCode {
				return ErrConflict
			}

			return fmt.Errorf(
				"create product with sku %q: %w",
				product.SKU,
				err,
			)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit product import transaction: %w", err)
	}

	return nil
}

func (r *PostgresProductRepository) Search(
	ctx context.Context,
	query string,
) ([]*domain.Product, error) {
	const sqlQuery = `
		SELECT id, name, sku, description, category, price, stock, weight_kg
		FROM products
		WHERE
			name ILIKE '%' || $1 || '%'
			OR sku ILIKE '%' || $1 || '%'
			OR description ILIKE '%' || $1 || '%'
			OR category ILIKE '%' || $1 || '%'
		ORDER BY id
	`

	rows, err := r.db.QueryContext(ctx, sqlQuery, query)
	if err != nil {
		return nil, fmt.Errorf("search products: %w", err)
	}
	defer rows.Close()

	products := make([]*domain.Product, 0)

	for rows.Next() {
		product := &domain.Product{}

		if err := rows.Scan(
			&product.ID,
			&product.Name,
			&product.SKU,
			&product.Description,
			&product.Category,
			&product.Price,
			&product.Stock,
			&product.WeightKg,
		); err != nil {
			return nil, fmt.Errorf("scan searched product: %w", err)
		}

		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate searched products: %w", err)
	}

	return products, nil
}
