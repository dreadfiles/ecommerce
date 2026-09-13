# ecommerce

> **E-commerce Code Challenge**

**CSV Download Date: November 7, 2026**

A modular e-commerce application built with Go, Vue, PostgreSQL, Docker, and Nginx, focused on transactional consistency, inventory control, idempotent purchases, validation, concurrency safety, and extensibility.

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
* Vue web interface
* Nginx reverse proxy
* Versioned REST API
* OpenAPI API documentation
* Postman API collection and environment
* Automated unit tests
* End-to-end integration tests
* 99.5% overall statement coverage

## Technology Stack

| Technology              | Purpose                                      |
| ----------------------- | -------------------------------------------- |
| Go 1.27                 | Backend application and REST API             |
| Vue 3                   | Web frontend                                 |
| Vite                    | Frontend development and build tooling       |
| Node.js 22              | Frontend build environment                   |
| Nginx 1.27              | Static frontend server and API reverse proxy |
| PostgreSQL 16           | Relational persistence                       |
| Docker / Docker Compose | Containerization and service orchestration   |
| Dev Containers          | Reproducible development environment         |
| OpenAPI                 | REST API contract and documentation          |
| Postman                 | API testing and request collection           |

## Architecture

The application follows a **modular layered architecture**, with separate `product`, `purchase`, and `importer` modules.

Each backend module separates responsibilities through:

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

### Frontend and API Architecture

The web application is served through Nginx and communicates with the Go API using the same origin:

```text
Browser
   │
   ▼
Nginx :80
   │
   ├── Vue static files
   │
   └── /api/*
          │
          ▼
       Go API :8080
          │
          ▼
      PostgreSQL :5432
```

The frontend uses `/api/v1` as its API base path.

Nginx proxies requests under `/api/` to the Go backend, avoiding the need for browser-side CORS configuration.

## Database

PostgreSQL was selected for ACID transactions, relational integrity, unique constraints, and row-level locking.

Database initialization scripts are located under `migrations/`.

Audit fields such as `created_at` and `updated_at` provide record traceability.

Data types follow the business requirements:

* **Price:** decimal with 2 fractional digits for monetary values.
* **Weight:** decimal with 3 fractional digits, providing `0.001 kg` precision (1 gram).
* **Stock:** integer because inventory represents whole units.
* **Money calculations:** `math/big.Rat` avoids floating-point precision errors.

## Transactional Purchase Flow

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

## Web Application

The project includes a Vue-based web application under the `view/` directory.

The frontend provides:

* Product listing
* Product search
* Product creation
* Product editing
* Product deletion
* CSV product import
* Shopping cart
* Quantity management
* Purchase creation
* Payment failure simulation
* Purchase result display
* Responsive user interface

The shopping cart can be accessed from the cart control in the application header. Adding a product automatically focuses the shopping cart section.

After a successful purchase, the purchase result is displayed and automatically focused so the user can immediately review the completed transaction.

Completed purchase results can be closed from the interface without affecting the persisted purchase.

The CSV import section can also be opened and closed when needed, keeping the main product management interface focused on the product list.

### Frontend Structure

```text
view/
├── Dockerfile
├── nginx.conf
├── package.json
├── package-lock.json
├── vite.config.js
├── index.html
└── src/
    ├── api.js
    ├── App.vue
    ├── main.js
    └── style.css
```

The frontend build is performed inside the Docker image using Node.js 22.

The generated Vue application is served by Nginx.

## Documentation & API Testing

The `docs/` directory contains the API documentation and Postman files used to test the application.

* [OpenAPI specification](docs/openapi.yaml) — machine-readable API contract describing the available endpoints, request/response schemas, and API behavior.
* [Postman Collection](docs/ecommerce.postman_collection.json) — collection containing API requests and test scenarios for products, searches, CSV import, and purchases.
* [Postman Environment](docs/ecommerce-local.postman_environment.json) — local environment containing the `base_url` variable used by the Postman collection.
* [Troubleshooting guide](docs/troubleshooting.md) — common issues and solutions for running and testing the application.

The Postman collection can be imported into Postman together with the environment to execute the documented API scenarios against the local application.

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
* Product deletion protection when purchases exist

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

### Reviewer Environment

The complete application can be executed by a reviewer using Docker only.

### Requirements

* Docker

No local Go, Node.js, npm, or PostgreSQL installation is required.

From the project root:

```bash
docker compose up --build
```

This starts the complete application stack:

```text
view
app
db
```

The services are connected through the Docker Compose network.

### Web Application

The main application is available at:

```text
http://localhost
```

The Vue frontend is served by Nginx on port `80`.

Nginx also proxies frontend API requests under `/api/` to the Go backend.

### Backend API

The Go REST API is also directly available at:

```text
http://localhost:8080
```

This endpoint is useful for direct API testing with Postman or other HTTP clients.

### PostgreSQL

PostgreSQL runs inside the `db` container on port `5432` within the Docker network.

The database is not required to be installed locally.

The default Docker Compose database configuration is:

```text
Host: db
Port: 5432
Database: ecommerce
User: ecommerce
Password: ecommerce
```

### Docker Services

The standard reviewer workflow uses:

```text
view
├── Vue application
└── Nginx :80

app
└── Go API :8080

db
└── PostgreSQL :5432
```

The complete request flow is:

```text
Browser
   │
   ▼
localhost:80
   │
   ▼
Nginx
   │
   ├── Vue application
   │
   └── /api/*
          │
          ▼
      app:8080
          │
          ▼
      db:5432
```

To stop the application:

```bash
docker compose down
```

The PostgreSQL data volume is preserved when using `docker compose down`.

To remove the PostgreSQL data volume and initialize the database from scratch:

```bash
docker compose down -v
```

The database is initialized using the SQL scripts under `migrations/`.

## Development Environment

The project includes a Dev Container configuration for a consistent and reproducible development environment.

### Requirements

* Docker
* Visual Studio Code
* Dev Containers extension

Open the project in Visual Studio Code and reopen it in the Dev Container.

The Dev Container uses the `app-dev` service and the same PostgreSQL service defined in the root Docker Compose configuration.

The development application service uses the Docker Compose `dev` profile and is started automatically by the Dev Container configuration.

Inside the Dev Container, the API can be started with:

```bash
go run ./cmd/api
```

The API is available at:

```text
http://localhost:8080
```

The development and reviewer environments use the same PostgreSQL service and persistent database volume while keeping the application containers separated:

```text
Development:

app-dev + db

Reviewer:

view + app + db
```

The `view` service is part of the reviewer/full Docker application workflow. The Dev Container is focused on backend development and does not replace the frontend production container.

When the Dev Container is closed, the `app-dev` container is removed while the PostgreSQL container and its persistent data remain available according to the Docker Compose lifecycle.

## Frontend Development

The frontend source code is located under `view/`.

The frontend can be built using the Docker-based environment without requiring Node.js or npm to be installed on the host machine.

The production frontend image uses:

```text
Node.js 22 Alpine
        │
        ▼
   Vite build
        │
        ▼
  Nginx 1.27 Alpine
```

The frontend uses the API base path:

```text
/api/v1
```

When running the complete Docker application, Nginx forwards these requests to the Go API.

The frontend dependencies are locked using:

```text
view/package-lock.json
```

The Docker build uses `npm ci` to provide reproducible dependency installation.

The `view/node_modules` directory is generated during dependency installation and should not be committed to the repository.

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
│   ├── ecommerce.postman_collection.json
│   ├── ecommerce-local.postman_environment.json
│   ├── openapi.yaml
│   └── troubleshooting.md
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
│       └── http/
├── migrations/
├── tests/
├── view/
│   ├── src/
│   │   ├── api.js
│   │   ├── App.vue
│   │   ├── main.js
│   │   └── style.css
│   ├── Dockerfile
│   ├── nginx.conf
│   ├── index.html
│   ├── package.json
│   ├── package-lock.json
│   └── vite.config.js
├── .gitignore
├── docker-compose.yml
├── Dockerfile
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
* **Vue:** provides a lightweight web interface for interacting with the ecommerce application.
* **Nginx:** serves the compiled frontend and provides a reverse proxy for API requests.
* **Docker Compose:** orchestrates the frontend, backend, and database services.
* **Docker Compose profiles:** separate the development application service from the standard reviewer workflow while keeping a single PostgreSQL service.

## Alternatives Considered

* Floating-point arithmetic was rejected for monetary calculations because of precision risks.

* Direct database access from handlers was rejected to avoid coupling HTTP transport to persistence.

* Client-provided prices were rejected as a source of truth because prices must be determined server-side.

* Partial CSV persistence was rejected to prevent inconsistent imports.

* Deleting products with existing purchases was rejected because historical purchase references must remain valid. The database foreign key constraint is used as the final integrity barrier, and the API returns `409 Conflict`.

* A real payment provider was not implemented because the challenge does not require external payment processing. A fake provider behind an interface keeps the purchase flow testable and extensible.

* Serving the frontend directly from the Vite development server was not used in the reviewer workflow. The production frontend is built into a container and served through Nginx.

* A separate frontend API origin was avoided. Nginx uses the same origin for the web application and `/api/` requests, simplifying deployment and avoiding unnecessary CORS configuration.

## Conclusion

This project demonstrates a complete Go backend and Vue web application with clear separation of responsibilities, transactional data management, concurrency-safe inventory handling, idempotent purchases, validated data imports, automated testing, and containerized execution.

The design prioritizes correctness, maintainability, and testability while keeping the codebase simple enough to understand and extend.

The complete Docker environment provides a straightforward reviewer experience:

```text
docker compose up --build
```

Then open:

```text
http://localhost
```

The web interface provides access to the ecommerce functionality, while the backend API remains directly available at:

```text
http://localhost:8080
```
