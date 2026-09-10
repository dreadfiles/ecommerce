# Troubleshooting

Known development and runtime issues encountered during the project.

## Database

### PostgreSQL schema is missing

**Symptom:** Expected tables are not available after starting the application.

**Cause:** PostgreSQL initialization scripts run only when the database volume is created for the first time.

**Resolution:** Remove the PostgreSQL volume and restart the services so the scripts under `migrations/` are executed again.

### Database connection fails

**Symptom:** The application cannot establish a database connection.

**Cause:** Database configuration is missing or incorrect, or the application is using the wrong database host.

**Resolution:** Verify `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, and `DB_NAME`. When running through Docker Compose, `DB_HOST` must reference the PostgreSQL service.

### Database changes are not reflected

**Symptom:** Changes to the initialization scripts do not appear in the existing database.

**Cause:** Existing PostgreSQL volumes are not reinitialized automatically.

**Resolution:** Recreate the database volume when a clean initialization is required. For persistent environments, use an appropriate database migration instead.

## Application

### Repository does not implement an interface

**Symptom:** The compiler reports that a repository implementation is missing a method such as `GetAll`.

**Cause:** A method was added to the repository interface without updating its implementation.

**Resolution:** Implement the required method in the concrete repository and update the related service if necessary.

### Undefined identifier or error

**Symptom:** The compiler reports an undefined identifier.

**Cause:** An incorrect package qualifier or variable name was introduced.

**Resolution:** Verify the package import and use the correct error or identifier.

## CSV Import

### CSV validation fails

**Symptom:** The import returns validation errors with CSV row numbers.

**Cause:** One or more records do not comply with the required structure or validation rules.

**Resolution:** Review the reported rows and correct the corresponding records. The complete file is validated before persistence.

### Invalid decimal format

**Symptom:** A price or weight is rejected as an invalid decimal.

**Cause:** The value contains an unsupported format, such as a currency symbol or non-numeric text.

**Resolution:** Use a valid decimal value according to the CSV specification.

### Negative stock

**Symptom:** A product is rejected because its stock is negative.

**Cause:** Inventory cannot be below zero.

**Resolution:** Use a stock value greater than or equal to zero.

### Required field is missing

**Symptom:** A row is rejected because a required field is empty.

**Cause:** Required product information such as name, category, or weight is missing.

**Resolution:** Provide a valid value for every required field.

### Duplicate SKU

**Symptom:** The importer reports a duplicate SKU.

**Cause:** The same SKU appears more than once in the CSV or already exists in the database.

**Resolution:** Ensure that every product has a unique SKU.

### Partial CSV import

**Symptom:** An invalid CSV does not insert valid rows.

**Cause:** The importer intentionally validates the complete file before persistence.

**Resolution:** Correct all reported validation errors and submit the file again.

## Purchases

### Product not found

**Symptom:** A purchase references a product that cannot be found.

**Cause:** The requested product ID does not exist.

**Resolution:** Use an existing product ID.

### Insufficient stock

**Symptom:** The purchase is rejected because there is not enough inventory.

**Cause:** The requested quantity exceeds the current stock.

**Resolution:** Reduce the quantity or replenish the product inventory.

### Purchase uses an outdated price

**Symptom:** The price sent by the client is not used to calculate the purchase.

**Cause:** Product prices must be controlled by the server.

**Resolution:** The purchase flow obtains a quote from the current database values before processing the purchase.

### Stock changes between quote and purchase

**Symptom:** A purchase that was valid during quoting is rejected during final creation.

**Cause:** Another transaction may have changed the inventory after the quote was generated.

**Resolution:** Stock is checked again inside the purchase transaction using row-level locking. The final database state is authoritative.

### Concurrent purchases consume the same stock

**Symptom:** Multiple purchases attempt to consume the same inventory simultaneously.

**Cause:** Concurrent requests may access the same product.

**Resolution:** Product rows are locked with `FOR UPDATE` during stock-sensitive operations.

### Purchase is partially persisted

**Symptom:** An order or inventory update fails during purchase creation.

**Cause:** Order and inventory changes must remain consistent.

**Resolution:** Purchase creation uses a database transaction. If any operation fails, the transaction is rolled back.

## Idempotency

### Idempotency key is missing

**Symptom:** A purchase request is rejected because no idempotency key was provided.

**Cause:** Purchase creation requires an `Idempotency-Key`.

**Resolution:** Provide a unique key for each new purchase operation.

### Duplicate purchase request

**Symptom:** Repeating a purchase does not create another order.

**Cause:** The same idempotency key was already processed.

**Resolution:** Reuse the same key when retrying the same purchase. Use a new key for a new purchase.

### Idempotency conflict

**Symptom:** A purchase is rejected because the idempotency key was already used with different data.

**Cause:** One idempotency key cannot represent two different purchase operations.

**Resolution:** Use a new idempotency key for the new purchase.

### Concurrent duplicate requests

**Symptom:** Multiple requests with the same idempotency key arrive simultaneously.

**Cause:** Requests can reach the application at nearly the same time.

**Resolution:** The database unique constraint on the idempotency key provides an additional concurrency safeguard.

## Payment

### Fake payment is declined

**Symptom:** The purchase is rejected with a payment failure.

**Cause:** The fake payment provider is configured to simulate a declined payment.

**Resolution:** Remove the failure simulation header when a successful payment is required.

## HTTP API

### Invalid request body

**Symptom:** The API rejects the request payload.

**Cause:** The request body contains invalid JSON or does not match the expected DTO.

**Resolution:** Send a valid payload according to the endpoint contract.

### Unexpected HTTP 500

**Symptom:** The API returns an internal server error.

**Cause:** An unexpected infrastructure or application error occurred.

**Resolution:** Review the application logs and verify database connectivity and configuration.

## Docker

### API is unavailable

**Symptom:** The application cannot be reached on port `18080`.

**Cause:** The application container is not running or the host port is unavailable.

**Resolution:** Verify the Docker services and confirm that port `18080` is available.

### PostgreSQL is not ready when the application starts

**Symptom:** The application initially cannot connect to PostgreSQL during startup.

**Cause:** Container startup does not guarantee that PostgreSQL is immediately ready to accept connections.

**Resolution:** The database connection uses retry handling to tolerate temporary PostgreSQL startup delays.

### Application changes are not reflected

**Symptom:** Source changes are not visible in the running application.

**Cause:** The application image may contain a previous build.

**Resolution:** Rebuild the application image when source or dependency changes require a new image.

## Development Environment

### Dev Container does not start

**Symptom:** The development container cannot be opened.

**Cause:** Docker is unavailable or the Dev Container configuration cannot access the Docker environment.

**Resolution:** Verify that Docker is running and reopen the project using the Dev Container configuration.

### Development environment differs between machines

**Symptom:** The application behaves differently between development environments.

**Cause:** Local tooling or dependency versions may differ.

**Resolution:** Use the provided Dev Container to standardize the development environment.
