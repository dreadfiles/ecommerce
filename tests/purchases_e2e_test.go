package tests

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"ecommerce/internal/database"
	importerhandler "ecommerce/internal/importer/handler"
	importerservice "ecommerce/internal/importer/service"
	producthandler "ecommerce/internal/product/handler"
	productrepository "ecommerce/internal/product/repository"
	productservice "ecommerce/internal/product/service"
	purchasehandler "ecommerce/internal/purchase/handler"
	purchaserepository "ecommerce/internal/purchase/repository"
	purchaseservice "ecommerce/internal/purchase/service"
	"ecommerce/internal/router"
)

type purchasesE2ETestEnvironment struct {
	server *httptest.Server
	db     *sql.DB
}

func purchasesE2ESetup(
	t *testing.T,
) *purchasesE2ETestEnvironment {
	t.Helper()

	db, err := database.NewPostgresDB()
	if err != nil {
		t.Fatalf(
			"connect to postgres: %v",
			err,
		)
	}

	productRepository := productrepository.NewPostgresProductRepository(
		db,
	)

	productService := productservice.NewProductService(
		productRepository,
	)

	productHandler := producthandler.NewProductHandler(
		productService,
	)

	productImportService := importerservice.NewProductImportService(
		productRepository,
	)

	productImportHandler := importerhandler.NewProductImportHandler(
		productImportService,
	)

	purchaseRepository := purchaserepository.NewPostgresPurchaseRepository(
		db,
	)

	purchaseService := purchaseservice.NewPurchaseService(
		purchaseRepository,
	)

	purchaseHandler := purchasehandler.NewPurchaseHandler(
		purchaseService,
	)

	handler := router.New(
		router.Dependencies{
			ProductHandler:  productHandler,
			ImportHandler:   productImportHandler,
			PurchaseHandler: purchaseHandler,
		},
	)

	server := httptest.NewServer(handler)

	environment := &purchasesE2ETestEnvironment{
		server: server,
		db:     db,
	}

	t.Cleanup(func() {
		server.Close()
		db.Close()
	})

	return environment
}

func purchasesE2EResetDatabase(
	t *testing.T,
	environment *purchasesE2ETestEnvironment,
) {
	t.Helper()

	queries := []string{
		"DELETE FROM order_items",
		"DELETE FROM orders",
		"DELETE FROM products",
	}

	for _, query := range queries {
		if _, err := environment.db.ExecContext(
			context.Background(),
			query,
		); err != nil {
			t.Fatalf(
				"reset database with %q: %v",
				query,
				err,
			)
		}
	}
}

func purchasesE2ERequest(
	t *testing.T,
	environment *purchasesE2ETestEnvironment,
	method string,
	path string,
	body any,
	headers map[string]string,
) *http.Response {
	t.Helper()

	var requestBody io.Reader

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			t.Fatalf(
				"marshal request body: %v",
				err,
			)
		}

		requestBody = bytes.NewReader(jsonBody)
	}

	request, err := http.NewRequest(
		method,
		environment.server.URL+path,
		requestBody,
	)
	if err != nil {
		t.Fatalf(
			"create request: %v",
			err,
		)
	}

	if body != nil {
		request.Header.Set(
			"Content-Type",
			"application/json",
		)
	}

	for key, value := range headers {
		request.Header.Set(
			key,
			value,
		)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf(
			"execute request: %v",
			err,
		)
	}

	return response
}

func purchasesE2EDecodeJSON(
	t *testing.T,
	response *http.Response,
	target any,
) {
	t.Helper()

	defer response.Body.Close()

	if err := json.NewDecoder(
		response.Body,
	).Decode(target); err != nil {
		t.Fatalf(
			"decode JSON response: %v",
			err,
		)
	}
}

func purchasesE2EAssertStatus(
	t *testing.T,
	response *http.Response,
	expectedStatus int,
) {
	t.Helper()

	defer response.Body.Close()

	if response.StatusCode != expectedStatus {
		body, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"expected status %d, got %d, body: %s",
			expectedStatus,
			response.StatusCode,
			string(body),
		)
	}
}

func purchasesE2ECreateProduct(
	t *testing.T,
	environment *purchasesE2ETestEnvironment,
	name string,
	sku string,
	price string,
	stock int,
) int64 {
	t.Helper()

	body := map[string]any{
		"name":        name,
		"sku":         sku,
		"description": "E2E test product",
		"category":    "TEST",
		"price":       price,
		"stock":       stock,
		"weight_kg":   "1.000",
	}

	response := purchasesE2ERequest(
		t,
		environment,
		http.MethodPost,
		"/api/v1/products",
		body,
		nil,
	)

	if response.StatusCode != http.StatusCreated {
		defer response.Body.Close()

		responseBody, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"create product: expected status %d, got %d, body: %s",
			http.StatusCreated,
			response.StatusCode,
			string(responseBody),
		)
	}

	var product struct {
		ID int64 `json:"id"`
	}

	purchasesE2EDecodeJSON(
		t,
		response,
		&product,
	)

	if product.ID <= 0 {
		t.Fatalf(
			"expected generated product ID, got %d",
			product.ID,
		)
	}

	return product.ID
}

func purchasesE2EFormatID(id int64) string {
	return formatInt64(id)
}

func formatInt64(id int64) string {
	if id < 0 {
		return "-" + formatInt64(-id)
	}

	if id == 0 {
		return "0"
	}

	var digits [20]byte
	position := len(digits)

	for id > 0 {
		position--
		digits[position] = byte('0' + id%10)
		id /= 10
	}

	return string(digits[position:])
}

func TestPurchasesE2E_CreatePurchaseAndReduceStock(t *testing.T) {
	environment := purchasesE2ESetup(t)

	purchasesE2EResetDatabase(
		t,
		environment,
	)

	productID := purchasesE2ECreateProduct(
		t,
		environment,
		"Gaming Mouse",
		"MOUSE-001",
		"100.00",
		10,
	)

	body := map[string]any{
		"items": []map[string]any{
			{
				"product_id": productID,
				"quantity":   2,
			},
		},
	}

	response := purchasesE2ERequest(
		t,
		environment,
		http.MethodPost,
		"/api/v1/purchases",
		body,
		map[string]string{
			"Idempotency-Key": "purchase-test-001",
		},
	)

	if response.StatusCode != http.StatusCreated {
		defer response.Body.Close()

		responseBody, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusCreated,
			response.StatusCode,
			string(responseBody),
		)
	}

	var purchase struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
		Total  string `json:"total"`
		Items  []struct {
			ProductID int64  `json:"product_id"`
			Quantity  int    `json:"quantity"`
			UnitPrice string `json:"unit_price"`
			Subtotal  string `json:"subtotal"`
		} `json:"items"`
	}

	purchasesE2EDecodeJSON(
		t,
		response,
		&purchase,
	)

	if purchase.ID <= 0 {
		t.Fatalf(
			"expected purchase ID greater than zero, got %d",
			purchase.ID,
		)
	}

	if purchase.Status != "PAID" {
		t.Fatalf(
			"expected status PAID, got %q",
			purchase.Status,
		)
	}

	if purchase.Total != "200.00" {
		t.Fatalf(
			"expected total 200.00, got %q",
			purchase.Total,
		)
	}

	if len(purchase.Items) != 1 {
		t.Fatalf(
			"expected 1 purchase item, got %d",
			len(purchase.Items),
		)
	}

	if purchase.Items[0].ProductID != productID {
		t.Fatalf(
			"expected product ID %d, got %d",
			productID,
			purchase.Items[0].ProductID,
		)
	}

	if purchase.Items[0].Quantity != 2 {
		t.Fatalf(
			"expected quantity 2, got %d",
			purchase.Items[0].Quantity,
		)
	}

	if purchase.Items[0].UnitPrice != "100.00" {
		t.Fatalf(
			"expected unit price 100.00, got %q",
			purchase.Items[0].UnitPrice,
		)
	}

	if purchase.Items[0].Subtotal != "200.00" {
		t.Fatalf(
			"expected subtotal 200.00, got %q",
			purchase.Items[0].Subtotal,
		)
	}

	getProductResponse := purchasesE2ERequest(
		t,
		environment,
		http.MethodGet,
		"/api/v1/products/"+purchasesE2EFormatID(productID),
		nil,
		nil,
	)

	if getProductResponse.StatusCode != http.StatusOK {
		defer getProductResponse.Body.Close()

		responseBody, _ := io.ReadAll(getProductResponse.Body)

		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusOK,
			getProductResponse.StatusCode,
			string(responseBody),
		)
	}

	var product struct {
		Stock int `json:"stock"`
	}

	purchasesE2EDecodeJSON(
		t,
		getProductResponse,
		&product,
	)

	if product.Stock != 8 {
		t.Fatalf(
			"expected stock 8 after purchase, got %d",
			product.Stock,
		)
	}
}

func TestPurchasesE2E_GetPurchaseByID(t *testing.T) {
	environment := purchasesE2ESetup(t)

	purchasesE2EResetDatabase(
		t,
		environment,
	)

	productID := purchasesE2ECreateProduct(
		t,
		environment,
		"Keyboard",
		"KEYBOARD-001",
		"150.00",
		10,
	)

	body := map[string]any{
		"items": []map[string]any{
			{
				"product_id": productID,
				"quantity":   2,
			},
		},
	}

	createResponse := purchasesE2ERequest(
		t,
		environment,
		http.MethodPost,
		"/api/v1/purchases",
		body,
		map[string]string{
			"Idempotency-Key": "purchase-get-001",
		},
	)

	if createResponse.StatusCode != http.StatusCreated {
		defer createResponse.Body.Close()

		responseBody, _ := io.ReadAll(createResponse.Body)

		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusCreated,
			createResponse.StatusCode,
			string(responseBody),
		)
	}

	var createdPurchase struct {
		ID int64 `json:"id"`
	}

	purchasesE2EDecodeJSON(
		t,
		createResponse,
		&createdPurchase,
	)

	getResponse := purchasesE2ERequest(
		t,
		environment,
		http.MethodGet,
		"/api/v1/purchases/"+purchasesE2EFormatID(createdPurchase.ID),
		nil,
		nil,
	)

	if getResponse.StatusCode != http.StatusOK {
		defer getResponse.Body.Close()

		responseBody, _ := io.ReadAll(getResponse.Body)

		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusOK,
			getResponse.StatusCode,
			string(responseBody),
		)
	}

	var purchase struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
		Total  string `json:"total"`
	}

	purchasesE2EDecodeJSON(
		t,
		getResponse,
		&purchase,
	)

	if purchase.ID != createdPurchase.ID {
		t.Fatalf(
			"expected ID %d, got %d",
			createdPurchase.ID,
			purchase.ID,
		)
	}

	if purchase.Status != "PAID" {
		t.Fatalf(
			"expected status PAID, got %q",
			purchase.Status,
		)
	}

	if purchase.Total != "300.00" {
		t.Fatalf(
			"expected total 300.00, got %q",
			purchase.Total,
		)
	}
}

func TestPurchasesE2E_IdempotencyReturnsSamePurchase(t *testing.T) {
	environment := purchasesE2ESetup(t)

	purchasesE2EResetDatabase(
		t,
		environment,
	)

	productID := purchasesE2ECreateProduct(
		t,
		environment,
		"Monitor",
		"MONITOR-001",
		"500.00",
		10,
	)

	body := map[string]any{
		"items": []map[string]any{
			{
				"product_id": productID,
				"quantity":   2,
			},
		},
	}

	headers := map[string]string{
		"Idempotency-Key": "purchase-idempotent-001",
	}

	firstResponse := purchasesE2ERequest(
		t,
		environment,
		http.MethodPost,
		"/api/v1/purchases",
		body,
		headers,
	)

	if firstResponse.StatusCode != http.StatusCreated {
		defer firstResponse.Body.Close()

		responseBody, _ := io.ReadAll(firstResponse.Body)

		t.Fatalf(
			"expected first request status %d, got %d, body: %s",
			http.StatusCreated,
			firstResponse.StatusCode,
			string(responseBody),
		)
	}

	var firstPurchase struct {
		ID    int64  `json:"id"`
		Total string `json:"total"`
	}

	purchasesE2EDecodeJSON(
		t,
		firstResponse,
		&firstPurchase,
	)

	secondResponse := purchasesE2ERequest(
		t,
		environment,
		http.MethodPost,
		"/api/v1/purchases",
		body,
		headers,
	)

	if secondResponse.StatusCode != http.StatusCreated {
		defer secondResponse.Body.Close()

		responseBody, _ := io.ReadAll(secondResponse.Body)

		t.Fatalf(
			"expected second request status %d, got %d, body: %s",
			http.StatusCreated,
			secondResponse.StatusCode,
			string(responseBody),
		)
	}

	var secondPurchase struct {
		ID    int64  `json:"id"`
		Total string `json:"total"`
	}

	purchasesE2EDecodeJSON(
		t,
		secondResponse,
		&secondPurchase,
	)

	if secondPurchase.ID != firstPurchase.ID {
		t.Fatalf(
			"expected same purchase ID %d, got %d",
			firstPurchase.ID,
			secondPurchase.ID,
		)
	}

	if secondPurchase.Total != firstPurchase.Total {
		t.Fatalf(
			"expected same total %q, got %q",
			firstPurchase.Total,
			secondPurchase.Total,
		)
	}

	getProductResponse := purchasesE2ERequest(
		t,
		environment,
		http.MethodGet,
		"/api/v1/products/"+purchasesE2EFormatID(productID),
		nil,
		nil,
	)

	var product struct {
		Stock int `json:"stock"`
	}

	purchasesE2EDecodeJSON(
		t,
		getProductResponse,
		&product,
	)

	if product.Stock != 8 {
		t.Fatalf(
			"expected stock 8 after idempotent requests, got %d",
			product.Stock,
		)
	}
}

func TestPurchasesE2E_IdempotencyConflict(t *testing.T) {
	environment := purchasesE2ESetup(t)

	purchasesE2EResetDatabase(
		t,
		environment,
	)

	productID := purchasesE2ECreateProduct(
		t,
		environment,
		"Headphones",
		"HEADPHONES-001",
		"300.00",
		10,
	)

	headers := map[string]string{
		"Idempotency-Key": "purchase-conflict-001",
	}

	firstBody := map[string]any{
		"items": []map[string]any{
			{
				"product_id": productID,
				"quantity":   1,
			},
		},
	}

	firstResponse := purchasesE2ERequest(
		t,
		environment,
		http.MethodPost,
		"/api/v1/purchases",
		firstBody,
		headers,
	)

	purchasesE2EAssertStatus(
		t,
		firstResponse,
		http.StatusCreated,
	)

	secondBody := map[string]any{
		"items": []map[string]any{
			{
				"product_id": productID,
				"quantity":   2,
			},
		},
	}

	secondResponse := purchasesE2ERequest(
		t,
		environment,
		http.MethodPost,
		"/api/v1/purchases",
		secondBody,
		headers,
	)

	purchasesE2EAssertStatus(
		t,
		secondResponse,
		http.StatusConflict,
	)
}

func TestPurchasesE2E_InsufficientStock(t *testing.T) {
	environment := purchasesE2ESetup(t)

	purchasesE2EResetDatabase(
		t,
		environment,
	)

	productID := purchasesE2ECreateProduct(
		t,
		environment,
		"Limited Product",
		"LIMITED-001",
		"100.00",
		2,
	)

	body := map[string]any{
		"items": []map[string]any{
			{
				"product_id": productID,
				"quantity":   3,
			},
		},
	}

	response := purchasesE2ERequest(
		t,
		environment,
		http.MethodPost,
		"/api/v1/purchases",
		body,
		map[string]string{
			"Idempotency-Key": "purchase-stock-001",
		},
	)

	purchasesE2EAssertStatus(
		t,
		response,
		http.StatusConflict,
	)

	getProductResponse := purchasesE2ERequest(
		t,
		environment,
		http.MethodGet,
		"/api/v1/products/"+purchasesE2EFormatID(productID),
		nil,
		nil,
	)

	var product struct {
		Stock int `json:"stock"`
	}

	purchasesE2EDecodeJSON(
		t,
		getProductResponse,
		&product,
	)

	if product.Stock != 2 {
		t.Fatalf(
			"expected stock to remain 2, got %d",
			product.Stock,
		)
	}
}

func TestPurchasesE2E_ProductNotFound(t *testing.T) {
	environment := purchasesE2ESetup(t)

	purchasesE2EResetDatabase(
		t,
		environment,
	)

	body := map[string]any{
		"items": []map[string]any{
			{
				"product_id": 999999,
				"quantity":   1,
			},
		},
	}

	response := purchasesE2ERequest(
		t,
		environment,
		http.MethodPost,
		"/api/v1/purchases",
		body,
		map[string]string{
			"Idempotency-Key": "purchase-not-found-001",
		},
	)

	purchasesE2EAssertStatus(
		t,
		response,
		http.StatusNotFound,
	)
}

func TestPurchasesE2E_PaymentFailure(t *testing.T) {
	environment := purchasesE2ESetup(t)

	purchasesE2EResetDatabase(
		t,
		environment,
	)

	productID := purchasesE2ECreateProduct(
		t,
		environment,
		"Payment Test Product",
		"PAYMENT-001",
		"100.00",
		10,
	)

	body := map[string]any{
		"items": []map[string]any{
			{
				"product_id": productID,
				"quantity":   2,
			},
		},
	}

	response := purchasesE2ERequest(
		t,
		environment,
		http.MethodPost,
		"/api/v1/purchases",
		body,
		map[string]string{
			"Idempotency-Key": "purchase-payment-failure-001",
			"X-Fake-Payment":  "failure",
		},
	)

	purchasesE2EAssertStatus(
		t,
		response,
		http.StatusPaymentRequired,
	)

	getProductResponse := purchasesE2ERequest(
		t,
		environment,
		http.MethodGet,
		"/api/v1/products/"+purchasesE2EFormatID(productID),
		nil,
		nil,
	)

	var product struct {
		Stock int `json:"stock"`
	}

	purchasesE2EDecodeJSON(
		t,
		getProductResponse,
		&product,
	)

	if product.Stock != 10 {
		t.Fatalf(
			"expected stock 10 after failed payment, got %d",
			product.Stock,
		)
	}
}

func TestPurchasesE2E_InvalidPurchase(t *testing.T) {
	environment := purchasesE2ESetup(t)

	purchasesE2EResetDatabase(
		t,
		environment,
	)

	body := map[string]any{
		"items": []map[string]any{},
	}

	response := purchasesE2ERequest(
		t,
		environment,
		http.MethodPost,
		"/api/v1/purchases",
		body,
		map[string]string{
			"Idempotency-Key": "purchase-invalid-001",
		},
	)

	purchasesE2EAssertStatus(
		t,
		response,
		http.StatusBadRequest,
	)
}

func TestPurchasesE2E_MissingIdempotencyKey(t *testing.T) {
	environment := purchasesE2ESetup(t)

	purchasesE2EResetDatabase(
		t,
		environment,
	)

	productID := purchasesE2ECreateProduct(
		t,
		environment,
		"Idempotency Test",
		"IDEMPOTENCY-001",
		"100.00",
		10,
	)

	body := map[string]any{
		"items": []map[string]any{
			{
				"product_id": productID,
				"quantity":   1,
			},
		},
	}

	response := purchasesE2ERequest(
		t,
		environment,
		http.MethodPost,
		"/api/v1/purchases",
		body,
		nil,
	)

	purchasesE2EAssertStatus(
		t,
		response,
		http.StatusBadRequest,
	)
}
