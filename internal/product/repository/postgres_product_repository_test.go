package repository

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"testing"

	"ecommerce/internal/product/domain"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
)

func queryRegex(query string) string {
	return regexp.QuoteMeta(
		strings.Join(strings.Fields(query), " "),
	)
}

func newProductRepositoryTest(t *testing.T) (
	*PostgresProductRepository,
	sqlmock.Sqlmock,
	func(),
) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	repo := NewPostgresProductRepository(db)

	cleanup := func() {
		_ = db.Close()
	}

	return repo, mock, cleanup
}

func testProduct() *domain.Product {
	return &domain.Product{
		ID:          1,
		Name:        "Laptop",
		SKU:         "LAP-001",
		Description: "Gaming laptop",
		Category:    "Computers",
		Price:       "1500.99",
		Stock:       10,
		WeightKg:    "1.250",
	}
}

func TestNewPostgresProductRepository(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	repo := NewPostgresProductRepository(db)

	if repo == nil {
		t.Fatal("expected repository, got nil")
	}

	if repo.db != db {
		t.Fatal("expected repository to use provided database")
	}
}

func TestPostgresProductRepository_Create(t *testing.T) {
	query := queryRegex(`
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
	`)

	tests := []struct {
		name      string
		product   *domain.Product
		setupMock func(sqlmock.Sqlmock)
		wantErr   error
		wantID    int64
	}{
		{
			name:    "success",
			product: testProduct(),
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).
					WithArgs(
						"Laptop",
						"LAP-001",
						"Gaming laptop",
						"Computers",
						"1500.99",
						10,
						"1.250",
					).
					WillReturnRows(
						sqlmock.NewRows([]string{"id"}).
							AddRow(int64(10)),
					)
			},
			wantID: 10,
		},
		{
			name:    "nil product",
			product: nil,
			setupMock: func(mock sqlmock.Sqlmock) {
			},
			wantErr: errors.New("product cannot be nil"),
		},
		{
			name:    "unique conflict",
			product: testProduct(),
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).
					WillReturnError(
						&pq.Error{Code: "23505"},
					)
			},
			wantErr: ErrConflict,
		},
		{
			name:    "database error",
			product: testProduct(),
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).
					WillReturnError(
						errors.New("database error"),
					)
			},
			wantErr: errors.New("create product: database error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := newProductRepositoryTest(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.Create(
				context.Background(),
				tt.product,
			)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				if tt.wantErr == ErrConflict {
					if !errors.Is(err, ErrConflict) {
						t.Fatalf(
							"expected ErrConflict, got %v",
							err,
						)
					}
				} else if err.Error() != tt.wantErr.Error() {
					t.Fatalf(
						"expected error %q, got %q",
						tt.wantErr.Error(),
						err.Error(),
					)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if tt.product.ID != tt.wantID {
					t.Fatalf(
						"expected ID %d, got %d",
						tt.wantID,
						tt.product.ID,
					)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf(
					"unmet expectations: %v",
					err,
				)
			}
		})
	}
}

func TestPostgresProductRepository_GetByID(t *testing.T) {
	query := queryRegex(`
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
	`)

	tests := []struct {
		name      string
		id        int64
		setupMock func(sqlmock.Sqlmock)
		wantErr   error
	}{
		{
			name: "success",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).
					WithArgs(int64(1)).
					WillReturnRows(
						sqlmock.NewRows([]string{
							"id",
							"name",
							"sku",
							"description",
							"category",
							"price",
							"stock",
							"weight_kg",
						}).
							AddRow(
								1,
								"Laptop",
								"LAP-001",
								"Gaming laptop",
								"Computers",
								"1500.99",
								10,
								"1.250",
							),
					)
			},
		},
		{
			name: "not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).
					WithArgs(int64(999)).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr: ErrNotFound,
		},
		{
			name: "database error",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).
					WithArgs(int64(1)).
					WillReturnError(
						errors.New("database error"),
					)
			},
			wantErr: errors.New(
				"get product by id: database error",
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := newProductRepositoryTest(t)
			defer cleanup()

			tt.setupMock(mock)

			product, err := repo.GetByID(
				context.Background(),
				tt.id,
			)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				if tt.wantErr == ErrNotFound {
					if !errors.Is(err, ErrNotFound) {
						t.Fatalf(
							"expected ErrNotFound, got %v",
							err,
						)
					}
				} else if err.Error() != tt.wantErr.Error() {
					t.Fatalf(
						"expected error %q, got %q",
						tt.wantErr.Error(),
						err.Error(),
					)
				}

				if product != nil {
					t.Fatal("expected nil product")
				}
			} else {
				if err != nil {
					t.Fatalf(
						"unexpected error: %v",
						err,
					)
				}

				if product == nil {
					t.Fatal("expected product, got nil")
				}

				if product.ID != 1 {
					t.Errorf(
						"expected ID 1, got %d",
						product.ID,
					)
				}

				if product.Price != "1500.99" {
					t.Errorf(
						"expected price 1500.99, got %s",
						product.Price,
					)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf(
					"unmet expectations: %v",
					err,
				)
			}
		})
	}
}

func TestPostgresProductRepository_GetAll(t *testing.T) {
	query := queryRegex(`
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
	`)

	t.Run("success", func(t *testing.T) {
		repo, mock, cleanup := newProductRepositoryTest(t)
		defer cleanup()

		mock.ExpectQuery(query).
			WillReturnRows(
				sqlmock.NewRows([]string{
					"id",
					"name",
					"sku",
					"description",
					"category",
					"price",
					"stock",
					"weight_kg",
				}).
					AddRow(
						1,
						"Laptop",
						"LAP-001",
						"Gaming laptop",
						"Computers",
						"1500.99",
						10,
						"1.250",
					).
					AddRow(
						2,
						"Mouse",
						"MOU-001",
						"Wireless mouse",
						"Accessories",
						"50.00",
						20,
						"0.150",
					),
			)

		products, err := repo.GetAll(
			context.Background(),
		)

		if err != nil {
			t.Fatalf(
				"unexpected error: %v",
				err,
			)
		}

		if len(products) != 2 {
			t.Fatalf(
				"expected 2 products, got %d",
				len(products),
			)
		}

		if products[0].Price != "1500.99" {
			t.Errorf(
				"expected first price 1500.99, got %s",
				products[0].Price,
			)
		}

		if products[1].Price != "50.00" {
			t.Errorf(
				"expected second price 50.00, got %s",
				products[1].Price,
			)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf(
				"unmet expectations: %v",
				err,
			)
		}
	})

	t.Run("query error", func(t *testing.T) {
		repo, mock, cleanup := newProductRepositoryTest(t)
		defer cleanup()

		mock.ExpectQuery(query).
			WillReturnError(
				errors.New("database error"),
			)

		products, err := repo.GetAll(
			context.Background(),
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if products != nil {
			t.Fatal("expected nil products")
		}

		if err.Error() !=
			"get all products: database error" {
			t.Fatalf(
				"unexpected error: %v",
				err,
			)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf(
				"unmet expectations: %v",
				err,
			)
		}
	})

	t.Run("scan error", func(t *testing.T) {
		repo, mock, cleanup := newProductRepositoryTest(t)
		defer cleanup()

		mock.ExpectQuery(query).
			WillReturnRows(
				sqlmock.NewRows([]string{
					"id",
					"name",
					"sku",
					"description",
					"category",
					"price",
					"stock",
					"weight_kg",
				}).
					AddRow(
						"invalid-id",
						"Laptop",
						"LAP-001",
						"Gaming laptop",
						"Computers",
						"1500.99",
						10,
						"1.250",
					),
			)

		products, err := repo.GetAll(
			context.Background(),
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if products != nil {
			t.Fatal("expected nil products")
		}

		if !strings.Contains(
			err.Error(),
			"scan product:",
		) {
			t.Fatalf(
				"expected scan product error, got: %v",
				err,
			)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf(
				"unmet expectations: %v",
				err,
			)
		}
	})

	t.Run("rows error", func(t *testing.T) {
		repo, mock, cleanup := newProductRepositoryTest(t)
		defer cleanup()

		rows := sqlmock.NewRows([]string{
			"id",
			"name",
			"sku",
			"description",
			"category",
			"price",
			"stock",
			"weight_kg",
		}).
			AddRow(
				1,
				"Laptop",
				"LAP-001",
				"Gaming laptop",
				"Computers",
				"1500.99",
				10,
				"1.250",
			).
			RowError(
				0,
				errors.New("row iteration error"),
			)

		mock.ExpectQuery(query).
			WillReturnRows(rows)

		products, err := repo.GetAll(
			context.Background(),
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if products != nil {
			t.Fatal("expected nil products")
		}

		if err.Error() !=
			"iterate products: row iteration error" {
			t.Fatalf(
				"unexpected error: %v",
				err,
			)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf(
				"unmet expectations: %v",
				err,
			)
		}
	})
}

func TestPostgresProductRepository_Update(t *testing.T) {
	query := queryRegex(`
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
	`)

	tests := []struct {
		name      string
		product   *domain.Product
		setupMock func(sqlmock.Sqlmock)
		wantErr   error
	}{
		{
			name:    "success",
			product: testProduct(),
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).
					WithArgs(
						"Laptop",
						"LAP-001",
						"Gaming laptop",
						"Computers",
						"1500.99",
						10,
						"1.250",
						int64(1),
					).
					WillReturnResult(
						sqlmock.NewResult(0, 1),
					)
			},
		},
		{
			name:    "nil product",
			product: nil,
			setupMock: func(mock sqlmock.Sqlmock) {
			},
			wantErr: errors.New(
				"product cannot be nil",
			),
		},
		{
			name:    "unique conflict",
			product: testProduct(),
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).
					WillReturnError(
						&pq.Error{Code: "23505"},
					)
			},
			wantErr: ErrConflict,
		},
		{
			name:    "database error",
			product: testProduct(),
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).
					WillReturnError(
						errors.New("database error"),
					)
			},
			wantErr: errors.New(
				"update product: database error",
			),
		},
		{
			name:    "rows affected error",
			product: testProduct(),
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).
					WillReturnResult(
						sqlmock.NewErrorResult(
							errors.New("rows affected error"),
						),
					)
			},
			wantErr: errors.New(
				"get affected rows: rows affected error",
			),
		},
		{
			name:    "not found",
			product: testProduct(),
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).
					WillReturnResult(
						sqlmock.NewResult(0, 0),
					)
			},
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := newProductRepositoryTest(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.Update(
				context.Background(),
				tt.product,
			)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				if tt.wantErr == ErrConflict {
					if !errors.Is(err, ErrConflict) {
						t.Fatalf(
							"expected ErrConflict, got %v",
							err,
						)
					}
				} else if tt.wantErr == ErrNotFound {
					if !errors.Is(err, ErrNotFound) {
						t.Fatalf(
							"expected ErrNotFound, got %v",
							err,
						)
					}
				} else if err.Error() != tt.wantErr.Error() {
					t.Fatalf(
						"expected error %q, got %q",
						tt.wantErr.Error(),
						err.Error(),
					)
				}
			} else if err != nil {
				t.Fatalf(
					"unexpected error: %v",
					err,
				)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf(
					"unmet expectations: %v",
					err,
				)
			}
		})
	}
}

func TestPostgresProductRepository_Delete(t *testing.T) {
	query := queryRegex(`
		DELETE FROM products
		WHERE id = $1
	`)

	tests := []struct {
		name      string
		id        int64
		setupMock func(sqlmock.Sqlmock)
		wantErr   error
	}{
		{
			name: "success",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).
					WithArgs(int64(1)).
					WillReturnResult(
						sqlmock.NewResult(0, 1),
					)
			},
		},
		{
			name: "database error",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).
					WithArgs(int64(1)).
					WillReturnError(
						errors.New("database error"),
					)
			},
			wantErr: errors.New(
				"delete product: database error",
			),
		},
		{
			name: "rows affected error",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).
					WithArgs(int64(1)).
					WillReturnResult(
						sqlmock.NewErrorResult(
							errors.New("rows affected error"),
						),
					)
			},
			wantErr: errors.New(
				"get affected rows: rows affected error",
			),
		},
		{
			name: "not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).
					WithArgs(int64(999)).
					WillReturnResult(
						sqlmock.NewResult(0, 0),
					)
			},
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := newProductRepositoryTest(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.Delete(
				context.Background(),
				tt.id,
			)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatal("expected error, got nil")
				}

				if tt.wantErr == ErrNotFound {
					if !errors.Is(err, ErrNotFound) {
						t.Fatalf(
							"expected ErrNotFound, got %v",
							err,
						)
					}
				} else if err.Error() != tt.wantErr.Error() {
					t.Fatalf(
						"expected error %q, got %q",
						tt.wantErr.Error(),
						err.Error(),
					)
				}
			} else if err != nil {
				t.Fatalf(
					"unexpected error: %v",
					err,
				)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf(
					"unmet expectations: %v",
					err,
				)
			}
		})
	}
}

func TestPostgresProductRepository_CreateMany(t *testing.T) {
	query := queryRegex(`
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
	`)

	t.Run("success", func(t *testing.T) {
		repo, mock, cleanup := newProductRepositoryTest(t)
		defer cleanup()

		products := []*domain.Product{
			testProduct(),
			{
				Name:        "Mouse",
				SKU:         "MOU-001",
				Description: "Wireless mouse",
				Category:    "Accessories",
				Price:       "50.00",
				Stock:       20,
				WeightKg:    "0.150",
			},
		}

		mock.ExpectBegin()

		mock.ExpectQuery(query).
			WithArgs(
				"Laptop",
				"LAP-001",
				"Gaming laptop",
				"Computers",
				"1500.99",
				10,
				"1.250",
			).
			WillReturnRows(
				sqlmock.NewRows([]string{"id"}).
					AddRow(int64(1)),
			)

		mock.ExpectQuery(query).
			WithArgs(
				"Mouse",
				"MOU-001",
				"Wireless mouse",
				"Accessories",
				"50.00",
				20,
				"0.150",
			).
			WillReturnRows(
				sqlmock.NewRows([]string{"id"}).
					AddRow(int64(2)),
			)

		mock.ExpectCommit()

		err := repo.CreateMany(
			context.Background(),
			products,
		)

		if err != nil {
			t.Fatalf(
				"unexpected error: %v",
				err,
			)
		}

		if products[0].ID != 1 {
			t.Errorf(
				"expected first ID 1, got %d",
				products[0].ID,
			)
		}

		if products[1].ID != 2 {
			t.Errorf(
				"expected second ID 2, got %d",
				products[1].ID,
			)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf(
				"unmet expectations: %v",
				err,
			)
		}
	})

	t.Run("begin transaction error", func(t *testing.T) {
		repo, mock, cleanup := newProductRepositoryTest(t)
		defer cleanup()

		mock.ExpectBegin().
			WillReturnError(
				errors.New("begin error"),
			)

		err := repo.CreateMany(
			context.Background(),
			[]*domain.Product{testProduct()},
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if err.Error() !=
			"begin product import transaction: begin error" {
			t.Fatalf(
				"unexpected error: %v",
				err,
			)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf(
				"unmet expectations: %v",
				err,
			)
		}
	})

	t.Run("nil product", func(t *testing.T) {
		repo, mock, cleanup := newProductRepositoryTest(t)
		defer cleanup()

		mock.ExpectBegin()

		err := repo.CreateMany(
			context.Background(),
			[]*domain.Product{nil},
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if err.Error() != "product cannot be nil" {
			t.Fatalf(
				"unexpected error: %v",
				err,
			)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf(
				"unmet expectations: %v",
				err,
			)
		}
	})

	t.Run("unique conflict", func(t *testing.T) {
		repo, mock, cleanup := newProductRepositoryTest(t)
		defer cleanup()

		product := testProduct()

		mock.ExpectBegin()

		mock.ExpectQuery(query).
			WithArgs(
				"Laptop",
				"LAP-001",
				"Gaming laptop",
				"Computers",
				"1500.99",
				10,
				"1.250",
			).
			WillReturnError(
				&pq.Error{Code: "23505"},
			)

		err := repo.CreateMany(
			context.Background(),
			[]*domain.Product{product},
		)

		if !errors.Is(err, ErrConflict) {
			t.Fatalf(
				"expected ErrConflict, got %v",
				err,
			)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf(
				"unmet expectations: %v",
				err,
			)
		}
	})

	t.Run("database error", func(t *testing.T) {
		repo, mock, cleanup := newProductRepositoryTest(t)
		defer cleanup()

		product := testProduct()

		mock.ExpectBegin()

		mock.ExpectQuery(query).
			WillReturnError(
				errors.New("insert error"),
			)

		err := repo.CreateMany(
			context.Background(),
			[]*domain.Product{product},
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if err.Error() !=
			"create product with sku \"LAP-001\": insert error" {
			t.Fatalf(
				"unexpected error: %v",
				err,
			)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf(
				"unmet expectations: %v",
				err,
			)
		}
	})

	t.Run("commit error", func(t *testing.T) {
		repo, mock, cleanup := newProductRepositoryTest(t)
		defer cleanup()

		product := testProduct()

		mock.ExpectBegin()

		mock.ExpectQuery(query).
			WithArgs(
				"Laptop",
				"LAP-001",
				"Gaming laptop",
				"Computers",
				"1500.99",
				10,
				"1.250",
			).
			WillReturnRows(
				sqlmock.NewRows([]string{"id"}).
					AddRow(int64(1)),
			)

		mock.ExpectCommit().
			WillReturnError(
				errors.New("commit error"),
			)

		err := repo.CreateMany(
			context.Background(),
			[]*domain.Product{product},
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if err.Error() !=
			"commit product import transaction: commit error" {
			t.Fatalf(
				"unexpected error: %v",
				err,
			)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf(
				"unmet expectations: %v",
				err,
			)
		}
	})
}

func TestPostgresProductRepository_Search(t *testing.T) {
	query := queryRegex(`
		SELECT id, name, sku, description, category, price, stock, weight_kg
		FROM products
		WHERE
			name ILIKE '%' || $1 || '%'
			OR sku ILIKE '%' || $1 || '%'
			OR description ILIKE '%' || $1 || '%'
			OR category ILIKE '%' || $1 || '%'
		ORDER BY id
	`)

	t.Run("success", func(t *testing.T) {
		repo, mock, cleanup := newProductRepositoryTest(t)
		defer cleanup()

		mock.ExpectQuery(query).
			WithArgs("laptop").
			WillReturnRows(
				sqlmock.NewRows([]string{
					"id",
					"name",
					"sku",
					"description",
					"category",
					"price",
					"stock",
					"weight_kg",
				}).
					AddRow(
						1,
						"Laptop",
						"LAP-001",
						"Gaming laptop",
						"Computers",
						"1500.99",
						10,
						"1.250",
					),
			)

		products, err := repo.Search(
			context.Background(),
			"laptop",
		)

		if err != nil {
			t.Fatalf(
				"unexpected error: %v",
				err,
			)
		}

		if len(products) != 1 {
			t.Fatalf(
				"expected 1 product, got %d",
				len(products),
			)
		}

		if products[0].Name != "Laptop" {
			t.Errorf(
				"expected Laptop, got %s",
				products[0].Name,
			)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf(
				"unmet expectations: %v",
				err,
			)
		}
	})

	t.Run("query error", func(t *testing.T) {
		repo, mock, cleanup := newProductRepositoryTest(t)
		defer cleanup()

		mock.ExpectQuery(query).
			WithArgs("laptop").
			WillReturnError(
				errors.New("database error"),
			)

		products, err := repo.Search(
			context.Background(),
			"laptop",
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if products != nil {
			t.Fatal("expected nil products")
		}

		if err.Error() !=
			"search products: database error" {
			t.Fatalf(
				"unexpected error: %v",
				err,
			)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf(
				"unmet expectations: %v",
				err,
			)
		}
	})

	t.Run("scan error", func(t *testing.T) {
		repo, mock, cleanup := newProductRepositoryTest(t)
		defer cleanup()

		mock.ExpectQuery(query).
			WithArgs("laptop").
			WillReturnRows(
				sqlmock.NewRows([]string{
					"id",
					"name",
					"sku",
					"description",
					"category",
					"price",
					"stock",
					"weight_kg",
				}).
					AddRow(
						1,
						"Laptop",
						"LAP-001",
						"Gaming laptop",
						"Computers",
						"1500.99",
						"invalid-stock",
						"1.250",
					),
			)

		products, err := repo.Search(
			context.Background(),
			"laptop",
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if products != nil {
			t.Fatal("expected nil products")
		}

		if !strings.Contains(
			err.Error(),
			"scan searched product:",
		) {
			t.Fatalf(
				"expected scan error, got: %v",
				err,
			)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf(
				"unmet expectations: %v",
				err,
			)
		}
	})

	t.Run("rows error", func(t *testing.T) {
		repo, mock, cleanup := newProductRepositoryTest(t)
		defer cleanup()

		rows := sqlmock.NewRows([]string{
			"id",
			"name",
			"sku",
			"description",
			"category",
			"price",
			"stock",
			"weight_kg",
		}).
			AddRow(
				1,
				"Laptop",
				"LAP-001",
				"Gaming laptop",
				"Computers",
				"1500.99",
				10,
				"1.250",
			).
			RowError(
				0,
				errors.New("row iteration error"),
			)

		mock.ExpectQuery(query).
			WithArgs("laptop").
			WillReturnRows(rows)

		products, err := repo.Search(
			context.Background(),
			"laptop",
		)

		if err == nil {
			t.Fatal("expected error, got nil")
		}

		if products != nil {
			t.Fatal("expected nil products")
		}

		if err.Error() !=
			"iterate searched products: row iteration error" {
			t.Fatalf(
				"unexpected error: %v",
				err,
			)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf(
				"unmet expectations: %v",
				err,
			)
		}
	})
}
