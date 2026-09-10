# ecommerce

> **E-commerce Code Challenge**

**CSV Download Date: November 7, 2026**

A modular e-commerce application built with Go and PostgreSQL, focused on transactional consistency, inventory control, idempotent purchases, validation, and extensibility.

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
* **Repository:** Persistence abstraction.
* **Domain:** Core business entities.
* **DTO:** External API contracts.

This separation reduces coupling and keeps business logic independent from transport and infrastructure concerns.

Repository and payment interfaces allow infrastructure implementations to evolve independently from the business layer.

## Database

PostgreSQL was selected for ACID transactions, relational integrity, unique constraints, and row-level locking.

Database initialization scripts are located under `migrations/`. Audit fields such as `created_at` and `updated_at` provide record traceability.

Data types follow the business requirements:

* **Price:** decimal with 2 fractional digits for monetary values.
* **Weight:** decimal with 3 fractional digits, providing `0.001 kg` precision (1 gram).
* **Stock:** integer because inventory represents whole units.
* **Money calculations:** `math/big.Rat` avoids floating-point precision errors.

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

Before creating a purchase, the system obtains a server-side quote using current product prices and validates inventory. Client-provided prices are not trusted.

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

The CSV is provided to the application as an external input through the product import endpoint. It is not stored as application data in the repository.

## Idempotency

Purchases require an `Idempotency-Key`.

A repeated request with the same key is treated as a retry of the same operation. Reusing the key with different purchase data is rejected.

Database uniqueness constraints provide additional protection against concurrent duplicate requests.

## Payment

The challenge does not require a real payment provider, so a fake payment implementation is used.

Payment is defined through an interface so a provider such as PayU can be introduced later without changing the purchase business logic.

The fake provider can simulate payment failure through the `X-Fake-Payment` header.

## Error Handling

Errors are handled according to application responsibility:

* **Repository:** persistence conditions
* **Service:** business rules
* **Handler:** HTTP response mapping

This prevents infrastructure and transport concerns from leaking into the business layer.

## Run the Application

### Requirement

* Docker

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

The project includes a Dev Container configuration for a consistent development environment.

Recommended development tools:

* Docker
* Visual Studio Code
* Dev Containers extension

Open the project in Visual Studio Code and reopen it in the Dev Container. Docker Compose manages the application and database services.

Using the Dev Container is optional for running the application.

## Project Structure

```text
.
├── .devcontainer/
│   ├── devcontainer.json
│   └── Dockerfile
├── internal/
│   ├── database/
│   ├── importer/
│   ├── product/
│   ├── purchase/
│   └── transport/
├── migrations/
├── docs/
│   ├── openapi.yaml
│   └── troubleshooting.md
├── .gitignore
├── docker-compose.yml
├── go.mod
├── go.sum
├── main.go
└── README.md
```

## Engineering Decisions

The main decisions were driven by consistency, maintainability, and extensibility:

* **Modular architecture:** isolates product, purchase, and import capabilities.
* **Layered design:** separates transport, business logic, and persistence.
* **DTOs and Domain models:** separate API contracts from business entities.
* **PostgreSQL:** provides transactional and concurrency guarantees.
* **`big.Rat`:** prevents monetary precision issues.
* **Pre-validated CSV:** prevents partial imports.
* **Transactions:** guarantee atomic purchase operations.
* **`FOR UPDATE`:** protects inventory during concurrent purchases.
* **Idempotency:** makes purchase retries safe.
* **Interfaces:** allow payment and persistence implementations to evolve independently.
* **API versioning:** provides a stable evolution path.
* **OpenAPI:** provides an explicit and machine-readable API contract.

## Alternatives Considered

* Floating-point arithmetic was rejected for monetary calculations because of precision risks.
* Direct database access from handlers was rejected to avoid coupling transport and persistence.
* Partial CSV imports were rejected to preserve consistency.
* Provider-specific payment logic was rejected to keep future integrations replaceable.
* Database-specific logic in the domain was rejected to preserve separation of concerns.

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

### Troubleshooting

Common problems and solutions are documented in:

[Troubleshooting Guide](docs/troubleshooting.md)

## Challenge Scope

The implementation covers the requested product management, CSV import, product search, purchase processing, fake payment, local database, containerization, API documentation, and documented engineering decisions.

## License

This project was developed as part of an e-commerce code challenge.
