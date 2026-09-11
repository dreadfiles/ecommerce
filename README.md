# ecommerce

> **E-commerce Code Challenge**

**CSV Download Date: November 7, 2026**

A modular e-commerce application built with Go and PostgreSQL, focused on transactional consistency, inventory control, idempotent purchases, validation, concurrency safety, and extensibility.

## Features

* Product CRUD and multi-field search
* CSV product import with validation
* Multi-product purchases
* Inventory concurrency control
* Idempotent purchase processing
* Transactional order creation
* Fake payment provider
* PostgreSQL persistence
* Docker-based execution
* Versioned REST API
* OpenAPI API documentation
* Automated unit tests
* End-to-end integration tests
* 99.5% overall statement coverage

## Technology Stack

| Technology              | Purpose                                    |
| ----------------------- | ------------------------------------------ |
| Go                      | Backend application                        |
| PostgreSQL 16           | Relational persistence                     |
| Docker / Docker Compose | Containerization and service orchestration |
| Dev Containers          | Reproducible development environment       |
| OpenAPI                 | REST API contract and documentation        |

## Architecture

The application follows a **modular layered architecture**, with separate `product`, `purchase`, and `importer` modules.

Each module separates responsibilities through:

```text
Handler → Service → Repository → Database
```

* **Handler:** HTTP request and response handling.
* **Service:** Business rules and application workflows.
* **Repository:** Persistence abstraction and database access.
* **Domain:** Core business entities.
* **DTO:** External API contracts.

This separation reduces coupling and keeps business logic independent from transport and infrastructure concerns.

Repository and payment interfaces allow infrastructure implementations to evolve independently from the business layer.

## Database

PostgreSQL was selected for ACID transactions, relational integrity, unique constraints, and row-level locking.

Database initialization scripts are located under `migrations/`.

Audit fields such as `created_at` and `updated_at` provide record traceability.

Data types follow the business requirements:

* **Price:** decimal with 2 fractional digits for monetary values.
* **Weight:** decimal with 3 fractional digits, providing `0.001 kg` precision (1 gram).
* **Stock:** integer because inventory represents whole units.
* **Money calculations:** `math/big.Rat` avoids floating-point precision errors.

### Transactional Purchase Flow

Purchase creation follows a server-side transactional workflow:

1. Validate the purchase request.
2. Validate the idempotency key.
3. Check whether the request has already been processed.
4. Obtain a server-side quote using current product prices.
5. Validate product availability and inventory.
6. Process the payment.
7. Create the order and update inventory within a database transaction.

Client-provided prices are never trusted.

Relevant product rows are locked using `FOR UPDATE` during stock-sensitive operations. This prevents concurrent purchases from consuming the same inventory.

## Configuration

Database configuration is provided through environment variables:

```text
DB_HOST
DB_PORT
DB_USER
DB_PASSWORD
DB_NAME
```

This keeps environment-specific configuration outside the source code.

## API

The REST API is versioned under `/api/v1` to provide a stable contract and a controlled path for future breaking changes.

### Products

```text
POST   /api/v1/products
GET    /api/v1/products
GET    /api/v1/products/{id}
PUT    /api/v1/products/{id}
DELETE /api/v1/products/{id}
GET    /api/v1/products/search
POST   /api/v1/products/import
```

Product search uses query parameters to support multiple search criteria without creating separate endpoints for each field.

### Purchases

```text
POST /api/v1/purchases
GET  /api/v1/purchases
GET  /api/v1/purchases/{id}
```

A purchase represents a basket containing one or more products.

Before creating a purchase, the system obtains a server-side quote using current product prices and validates inventory.

Purchase creation and stock updates are executed in a database transaction, providing all-or-nothing behavior.

`FOR UPDATE` locks the relevant product rows during stock-sensitive operations to prevent concurrent purchases from consuming the same inventory.

## CSV Import

The complete CSV is validated before persistence to prevent partial imports.

Validation includes:

* Required fields
* Decimal formats
* Non-negative stock
* Unique SKUs
* Valid product data

Validation errors include the corresponding CSV row.

Expected structure:

```text
name,sku,description,category,price,stock,weight_kg
```

The CSV is provided to the application as external input through the product import endpoint. It is not stored as application data in the repository.

## Idempotency

Purchases require an `Idempotency-Key`.

A repeated request with the same key is treated as a retry of the same operation.

Reusing the same key with different purchase data is rejected.

Database uniqueness constraints provide additional protection against concurrent duplicate requests.

This makes purchase retries safe while preventing accidental duplicate orders.

## Payment

The challenge does not require a real payment provider, so a fake payment implementation is used.

Payment is defined through an interface so a provider such as PayU can be introduced later without changing the purchase business logic.

The fake provider can simulate payment failure through the `X-Fake-Payment` header.

## Error Handling

Errors are handled according to application responsibility:

* **Repository:** persistence and database conditions
* **Service:** business rules and application errors
* **Handler:** HTTP response mapping

This prevents infrastructure and transport concerns from leaking into the business layer.

The API uses structured JSON error responses with appropriate HTTP status codes.

## Testing

The project includes automated **unit tests and end-to-end integration tests** covering the main application layers and business workflows.

### Unit Tests

Unit tests cover application logic in isolation, including:

* Business rules and validations
* Product service
* Purchase service
* HTTP handlers
* CSV parser
* CSV validator
* Repository behavior
* Payment service behavior
* Error handling and mapping
* Edge cases and invalid inputs

Mocks are used where isolation is appropriate, particularly for service-level tests.

### End-to-End Integration Tests

End-to-end tests are located under `tests/` and exercise the API through the HTTP layer using the application router and a real PostgreSQL database.

The E2E suite covers:

* Product CRUD operations
* Product search
* CSV product import
* Purchase creation
* Inventory management
* Idempotent purchases
* Payment failure scenarios
* API error responses
* Database persistence
* Transactional purchase flows

This validates the integration between the HTTP, service, repository, and database layers.

### Running Tests

Run the complete test suite:

```bash
go test ./...
```

Compile all packages without executing tests:

```bash
go test ./... -run '^$'
```

Build the complete application:

```bash
go build ./...
```

### Test Coverage

Generate the coverage report for the complete application:

```bash
go test ./... -coverpkg=./internal/... -coverprofile=coverage.out
```

Display the coverage summary:

```bash
go tool cover -func=coverage.out
```

The current test suite achieves:

```text
99.5% overall statement coverage
```

across the `internal` application packages.

## Run the Application

### Requirements

* Docker
* Visual Studio Code
* Dev Containers extension

Docker Compose is provided to start the application and PostgreSQL services together.

From the project root:

```bash
docker compose up --build
```

The database is initialized using the SQL scripts under `migrations/`.

The API is available at:

```text
http://localhost:18080
```

To stop the application:

```bash
docker compose down
```

## Development Environment

The project includes a Dev Container configuration for a consistent and reproducible development environment.

Recommended development tools:

* Docker
* Visual Studio Code
* Dev Containers extension

Open the project in Visual Studio Code and reopen it in the Dev Container.

The Dev Container provides the development environment and integrates Docker Compose with the application and PostgreSQL services.

Using the Dev Container is recommended for development.

## Project Structure

```text
.
├── .devcontainer/
│   ├── devcontainer.json
│   └── Dockerfile
├── cmd/
│   └── api/
│       └── main.go
├── docs/
│   ├── openapi.yaml
│   └── troubleshooting.md
├── frontend/
├── internal/
│   ├── database/
│   ├── importer/
│   │   ├── handler/
│   │   ├── parser/
│   │   ├── service/
│   │   └── validator/
│   ├── product/
│   │   ├── domain/
│   │   ├── dto/
│   │   ├── handler/
│   │   ├── repository/
│   │   └── service/
│   ├── purchase/
│   │   ├── domain/
│   │   ├── dto/
│   │   ├── handler/
│   │   ├── repository/
│   │   └── service/
│   ├── router/
│   └── transport/
├── migrations/
├── tests/
├── .gitignore
├── docker-compose.yml
├── go.mod
├── go.sum
└── README.md
```

## Engineering Decisions

The main decisions were driven by consistency, maintainability, testability, and extensibility:

* **Modular architecture:** isolates product, purchase, and import capabilities.
* **Layered design:** separates transport, business logic, and persistence.
* **DTOs and Domain models:** separate API contracts from business entities.
* **PostgreSQL:** provides transactional and concurrency guarantees.
* **`big.Rat`:** prevents monetary precision issues.
* **Pre-validated CSV:** prevents partial imports.
* **Database transactions:** guarantee atomic purchase operations.
* **`FOR UPDATE`:** protects inventory during concurrent purchases.
* **Idempotency:** makes purchase retries safe.
* **Interfaces:** allow payment and persistence implementations to evolve independently.
* **API versioning:** provides a stable evolution path.
* **OpenAPI:** provides an explicit and machine-readable API contract.
* **Automated testing:** validates business logic and end-to-end application behavior.

## Alternatives Considered

* Floating-point arithmetic was rejected for monetary calculations because of precision risks.
* Direct database access from handlers was rejected to avoid coupling transport and persistence.
* Partial CSV imports were rejected to preserve consistency.
* Provider-specific payment logic was rejected to keep future integrations replaceable.
* Database-specific logic in the domain was rejected to preserve separation of concerns.
* Trusting client-provided product prices was rejected to prevent inconsistent or manipulated purchase totals.

## Documentation

### OpenAPI

The complete API contract is available in:

[OpenAPI Specification](docs/openapi.yaml)

To visualize the API documentation:

1. Open `docs/openapi.yaml`.
2. Copy the complete contents of the file.
3. Open [Swagger Editor](https://editor.swagger.io/).
4. Paste the YAML content into the editor.
5. Swagger Editor will render the API documentation and available endpoints.

The OpenAPI file is kept in the repository as the source of truth for the API contract.

### Troubleshooting

Common development and runtime issues are documented in:

[Troubleshooting Guide](docs/troubleshooting.md)

## Challenge Scope

The implementation covers the requested product management, CSV import, product search, purchase processing, fake payment, local database, containerization, API documentation, automated testing, and documented engineering decisions.

The application is designed to demonstrate production-oriented backend practices including modular architecture, transactional consistency, concurrency control, idempotency, validation, persistence abstraction, and automated testing.

## License

This project was developed as part of an e-commerce code challenge.
