package tests

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

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

type testServer struct {
	server *httptest.Server
}

type productResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	SKU         string `json:"sku"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Price       string `json:"price"`
	Stock       int    `json:"stock"`
	WeightKg    string `json:"weight_kg"`
}

type purchaseResponse struct {
	ID     int64                  `json:"id"`
	Status string                 `json:"status"`
	Total  string                 `json:"total"`
	Items  []purchaseItemResponse `json:"items"`
}

type purchaseItemResponse struct {
	ProductID int64  `json:"product_id"`
	Quantity  int    `json:"quantity"`
	UnitPrice string `json:"unit_price"`
	Subtotal  string `json:"subtotal"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()

	db, err := database.NewPostgresDB()
	if err != nil {
		t.Fatalf("connect to database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	cleanupDatabase(t, db)

	t.Cleanup(func() {
		cleanupDatabase(t, db)
	})

	productRepository := productrepository.NewPostgresProductRepository(db)

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

	t.Cleanup(func() {
		server.Close()
	})

	return &testServer{
		server: server,
	}
}

func cleanupDatabase(t *testing.T, db *sql.DB) {
	t.Helper()

	_, err := db.Exec(
		"TRUNCATE TABLE order_items, orders, products RESTART IDENTITY CASCADE",
	)
	if err != nil {
		t.Fatalf("cleanup database: %v", err)
	}
}

func newSKU(t *testing.T) string {
	t.Helper()

	return fmt.Sprintf(
		"TEST-%d",
		time.Now().UnixNano(),
	)
}

func createProduct(
	t *testing.T,
	ts *testServer,
	name string,
	sku string,
	price any,
	stock int,
) productResponse {
	t.Helper()

	priceValue := fmt.Sprintf("%v", price)

	requestBody := map[string]any{
		"name":        name,
		"sku":         sku,
		"description": "Integration test product",
		"category":    "TEST",
		"price":       priceValue,
		"stock":       stock,
		"weight_kg":   "1.000",
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("marshal product request: %v", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		ts.server.URL+"/api/v1/products",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("create product request: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("create product request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		responseBody, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusCreated,
			response.StatusCode,
			string(responseBody),
		)
	}

	var product productResponse

	if err := json.NewDecoder(response.Body).Decode(&product); err != nil {
		t.Fatalf("decode product response: %v", err)
	}

	return product
}

func createPurchase(
	t *testing.T,
	ts *testServer,
	productID int64,
	quantity int,
	idempotencyKey string,
) purchaseResponse {
	t.Helper()

	requestBody := map[string]any{
		"items": []map[string]any{
			{
				"product_id": productID,
				"quantity":   quantity,
			},
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("marshal purchase request: %v", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		ts.server.URL+"/api/v1/purchases",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("create purchase request: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", idempotencyKey)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("purchase request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		responseBody, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusCreated,
			response.StatusCode,
			string(responseBody),
		)
	}

	var purchase purchaseResponse

	if err := json.NewDecoder(
		response.Body,
	).Decode(&purchase); err != nil {
		t.Fatalf("decode purchase response: %v", err)
	}

	return purchase
}

func TestIntegration_ProductCreateAndGetByID(t *testing.T) {
	ts := newTestServer(t)

	product := createProduct(
		t,
		ts,
		"Integration Product",
		newSKU(t),
		"25.50",
		10,
	)

	request, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf(
			"%s/api/v1/products/%d",
			ts.server.URL,
			product.ID,
		),
		nil,
	)
	if err != nil {
		t.Fatalf("create get product request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("get product request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			response.StatusCode,
		)
	}

	var result productResponse

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode product response: %v", err)
	}

	if result.ID != product.ID {
		t.Fatalf(
			"expected product ID %d, got %d",
			product.ID,
			result.ID,
		)
	}

	if result.SKU != product.SKU {
		t.Fatalf(
			"expected SKU %s, got %s",
			product.SKU,
			result.SKU,
		)
	}
}

func TestIntegration_ProductUpdate(t *testing.T) {
	ts := newTestServer(t)

	product := createProduct(
		t,
		ts,
		"Original Product",
		newSKU(t),
		"10.00",
		5,
	)

	requestBody := map[string]any{
		"name":        "Updated Product",
		"sku":         product.SKU,
		"description": "Updated description",
		"category":    "UPDATED",
		"price":       "15.00",
		"stock":       20,
		"weight_kg":   "2.500",
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("marshal update request: %v", err)
	}

	request, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf(
			"%s/api/v1/products/%d",
			ts.server.URL,
			product.ID,
		),
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("create update request: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("update product request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusOK,
			response.StatusCode,
			string(responseBody),
		)
	}

	var result productResponse

	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("decode update response: %v", err)
	}

	if result.Name != "Updated Product" {
		t.Fatalf(
			"expected updated name, got %s",
			result.Name,
		)
	}

	if result.Price != "15.00" {
		t.Fatalf(
			"expected updated price 15.00, got %s",
			result.Price,
		)
	}

	if result.Stock != 20 {
		t.Fatalf(
			"expected updated stock 20, got %d",
			result.Stock,
		)
	}
}

func TestIntegration_ProductSearch(t *testing.T) {
	ts := newTestServer(t)

	createProduct(
		t,
		ts,
		"Mechanical Keyboard",
		newSKU(t),
		"80.00",
		10,
	)

	request, err := http.NewRequest(
		http.MethodGet,
		ts.server.URL+"/api/v1/products/search?q=Keyboard",
		nil,
	)
	if err != nil {
		t.Fatalf("create search request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("search products request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			response.StatusCode,
		)
	}

	var products []productResponse

	if err := json.NewDecoder(response.Body).Decode(&products); err != nil {
		t.Fatalf("decode search response: %v", err)
	}

	if len(products) != 1 {
		t.Fatalf(
			"expected 1 product, got %d",
			len(products),
		)
	}

	if products[0].Name != "Mechanical Keyboard" {
		t.Fatalf(
			"expected Mechanical Keyboard, got %s",
			products[0].Name,
		)
	}
}

func TestIntegration_ProductDelete(t *testing.T) {
	ts := newTestServer(t)

	product := createProduct(
		t,
		ts,
		"Product To Delete",
		newSKU(t),
		"20.00",
		5,
	)

	request, err := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf(
			"%s/api/v1/products/%d",
			ts.server.URL,
			product.ID,
		),
		nil,
	)
	if err != nil {
		t.Fatalf("create delete request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("delete product request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNoContent {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNoContent,
			response.StatusCode,
		)
	}

	getRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf(
			"%s/api/v1/products/%d",
			ts.server.URL,
			product.ID,
		),
		nil,
	)
	if err != nil {
		t.Fatalf("create get deleted product request: %v", err)
	}

	getResponse, err := http.DefaultClient.Do(getRequest)
	if err != nil {
		t.Fatalf("get deleted product request: %v", err)
	}
	defer getResponse.Body.Close()

	if getResponse.StatusCode != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			getResponse.StatusCode,
		)
	}
}

func TestIntegration_ProductDuplicateSKU(t *testing.T) {
	ts := newTestServer(t)

	sku := newSKU(t)

	createProduct(
		t,
		ts,
		"First Product",
		sku,
		"10.00",
		5,
	)

	requestBody := map[string]any{
		"name":        "Second Product",
		"sku":         sku,
		"description": "Duplicate SKU",
		"category":    "TEST",
		"price":       "20.00",
		"stock":       5,
		"weight_kg":   "1.000",
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("marshal duplicate product: %v", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		ts.server.URL+"/api/v1/products",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("create duplicate product request: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("duplicate product request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusConflict {
		responseBody, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusConflict,
			response.StatusCode,
			string(responseBody),
		)
	}
}

func TestIntegration_ProductCSVImport(t *testing.T) {
	ts := newTestServer(t)

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)

	fileWriter, err := writer.CreateFormFile(
		"file",
		"products.csv",
	)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}

	csvWriter := csv.NewWriter(fileWriter)

	rows := [][]string{
		{
			"name",
			"sku",
			"description",
			"category",
			"price",
			"stock",
			"weight_kg",
		},
		{
			"Imported Product 1",
			newSKU(t),
			"Imported product",
			"IMPORT",
			"10.50",
			"15",
			"1.250",
		},
		{
			"Imported Product 2",
			newSKU(t),
			"Imported product",
			"IMPORT",
			"20.00",
			"20",
			"2.000",
		},
	}

	if err := csvWriter.WriteAll(rows); err != nil {
		t.Fatalf("write CSV: %v", err)
	}

	if err := csvWriter.Error(); err != nil {
		t.Fatalf("CSV writer error: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		ts.server.URL+"/api/v1/products/import",
		&body,
	)
	if err != nil {
		t.Fatalf("create import request: %v", err)
	}

	request.Header.Set(
		"Content-Type",
		writer.FormDataContentType(),
	)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("import request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		responseBody, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusCreated,
			response.StatusCode,
			string(responseBody),
		)
	}

	getRequest, err := http.NewRequest(
		http.MethodGet,
		ts.server.URL+"/api/v1/products",
		nil,
	)
	if err != nil {
		t.Fatalf("create get products request: %v", err)
	}

	getResponse, err := http.DefaultClient.Do(getRequest)
	if err != nil {
		t.Fatalf("get imported products: %v", err)
	}
	defer getResponse.Body.Close()

	if getResponse.StatusCode != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			getResponse.StatusCode,
		)
	}

	var products []productResponse

	if err := json.NewDecoder(
		getResponse.Body,
	).Decode(&products); err != nil {
		t.Fatalf("decode products response: %v", err)
	}

	if len(products) != 2 {
		t.Fatalf(
			"expected 2 imported products, got %d",
			len(products),
		)
	}
}

func TestIntegration_PurchaseSuccess(t *testing.T) {
	ts := newTestServer(t)

	product := createProduct(
		t,
		ts,
		"Purchase Product",
		newSKU(t),
		"25.00",
		10,
	)

	purchase := createPurchase(
		t,
		ts,
		product.ID,
		2,
		"idempotency-success-1",
	)

	if purchase.Status != "PAID" {
		t.Fatalf(
			"expected status PAID, got %s",
			purchase.Status,
		)
	}

	if purchase.Total != "50.00" {
		t.Fatalf(
			"expected total 50.00, got %s",
			purchase.Total,
		)
	}

	if len(purchase.Items) != 1 {
		t.Fatalf(
			"expected 1 item, got %d",
			len(purchase.Items),
		)
	}

	if purchase.Items[0].ProductID != product.ID {
		t.Fatalf(
			"expected product ID %d, got %d",
			product.ID,
			purchase.Items[0].ProductID,
		)
	}

	if purchase.Items[0].Quantity != 2 {
		t.Fatalf(
			"expected quantity 2, got %d",
			purchase.Items[0].Quantity,
		)
	}

	if purchase.Items[0].UnitPrice != "25.00" {
		t.Fatalf(
			"expected unit price 25.00, got %s",
			purchase.Items[0].UnitPrice,
		)
	}

	if purchase.Items[0].Subtotal != "50.00" {
		t.Fatalf(
			"expected subtotal 50.00, got %s",
			purchase.Items[0].Subtotal,
		)
	}
}

func TestIntegration_PurchaseInsufficientStock(t *testing.T) {
	ts := newTestServer(t)

	product := createProduct(
		t,
		ts,
		"Limited Stock Product",
		newSKU(t),
		"30.00",
		2,
	)

	requestBody := map[string]any{
		"items": []map[string]any{
			{
				"product_id": product.ID,
				"quantity":   3,
			},
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("marshal purchase request: %v", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		ts.server.URL+"/api/v1/purchases",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("create purchase request: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "idempotency-stock-error")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("purchase request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusConflict {
		responseBody, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusConflict,
			response.StatusCode,
			string(responseBody),
		)
	}

	getRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf(
			"%s/api/v1/products/%d",
			ts.server.URL,
			product.ID,
		),
		nil,
	)
	if err != nil {
		t.Fatalf("create get product request: %v", err)
	}

	getResponse, err := http.DefaultClient.Do(getRequest)
	if err != nil {
		t.Fatalf("get product request: %v", err)
	}
	defer getResponse.Body.Close()

	var result productResponse

	if err := json.NewDecoder(
		getResponse.Body,
	).Decode(&result); err != nil {
		t.Fatalf("decode product response: %v", err)
	}

	if result.Stock != 2 {
		t.Fatalf(
			"expected stock to remain 2, got %d",
			result.Stock,
		)
	}
}

func TestIntegration_PurchasePaymentFailure(t *testing.T) {
	ts := newTestServer(t)

	product := createProduct(
		t,
		ts,
		"Payment Failure Product",
		newSKU(t),
		"40.00",
		10,
	)

	requestBody := map[string]any{
		"items": []map[string]any{
			{
				"product_id": product.ID,
				"quantity":   2,
			},
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("marshal purchase request: %v", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		ts.server.URL+"/api/v1/purchases",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("create purchase request: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "idempotency-payment-failure")
	request.Header.Set("X-Fake-Payment", "failure")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("purchase request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusPaymentRequired {
		responseBody, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusPaymentRequired,
			response.StatusCode,
			string(responseBody),
		)
	}

	getRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf(
			"%s/api/v1/products/%d",
			ts.server.URL,
			product.ID,
		),
		nil,
	)
	if err != nil {
		t.Fatalf("create get product request: %v", err)
	}

	getResponse, err := http.DefaultClient.Do(getRequest)
	if err != nil {
		t.Fatalf("get product request: %v", err)
	}
	defer getResponse.Body.Close()

	var result productResponse

	if err := json.NewDecoder(
		getResponse.Body,
	).Decode(&result); err != nil {
		t.Fatalf("decode product response: %v", err)
	}

	if result.Stock != 10 {
		t.Fatalf(
			"expected stock to remain 10, got %d",
			result.Stock,
		)
	}
}

func TestIntegration_PurchaseIdempotencySameRequest(t *testing.T) {
	ts := newTestServer(t)

	product := createProduct(
		t,
		ts,
		"Idempotent Product",
		newSKU(t),
		"15.00",
		10,
	)

	idempotencyKey := "idempotency-same-request"

	firstPurchase := createPurchase(
		t,
		ts,
		product.ID,
		2,
		idempotencyKey,
	)

	secondPurchase := createPurchase(
		t,
		ts,
		product.ID,
		2,
		idempotencyKey,
	)

	if firstPurchase.ID != secondPurchase.ID {
		t.Fatalf(
			"expected same purchase ID, got %d and %d",
			firstPurchase.ID,
			secondPurchase.ID,
		)
	}

	getRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf(
			"%s/api/v1/products/%d",
			ts.server.URL,
			product.ID,
		),
		nil,
	)
	if err != nil {
		t.Fatalf("create get product request: %v", err)
	}

	response, err := http.DefaultClient.Do(getRequest)
	if err != nil {
		t.Fatalf("get product request: %v", err)
	}
	defer response.Body.Close()

	var result productResponse

	if err := json.NewDecoder(
		response.Body,
	).Decode(&result); err != nil {
		t.Fatalf("decode product response: %v", err)
	}

	if result.Stock != 8 {
		t.Fatalf(
			"expected stock 8 after idempotent requests, got %d",
			result.Stock,
		)
	}
}

func TestIntegration_PurchaseIdempotencyDifferentRequest(t *testing.T) {
	ts := newTestServer(t)

	product := createProduct(
		t,
		ts,
		"Idempotency Conflict Product",
		newSKU(t),
		"10.00",
		10,
	)

	idempotencyKey := "idempotency-different-request"

	createPurchase(
		t,
		ts,
		product.ID,
		1,
		idempotencyKey,
	)

	requestBody := map[string]any{
		"items": []map[string]any{
			{
				"product_id": product.ID,
				"quantity":   2,
			},
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("marshal purchase request: %v", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		ts.server.URL+"/api/v1/purchases",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("create purchase request: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", idempotencyKey)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("purchase request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusConflict {
		responseBody, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusConflict,
			response.StatusCode,
			string(responseBody),
		)
	}
}

func TestIntegration_PurchaseProductNotFound(t *testing.T) {
	ts := newTestServer(t)

	requestBody := map[string]any{
		"items": []map[string]any{
			{
				"product_id": 999999,
				"quantity":   1,
			},
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("marshal purchase request: %v", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		ts.server.URL+"/api/v1/purchases",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("create purchase request: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "idempotency-product-not-found")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("purchase request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotFound {
		responseBody, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusNotFound,
			response.StatusCode,
			string(responseBody),
		)
	}
}

func TestIntegration_GetPurchaseByID(t *testing.T) {
	ts := newTestServer(t)

	product := createProduct(
		t,
		ts,
		"Get Purchase Product",
		newSKU(t),
		"12.00",
		10,
	)

	purchase := createPurchase(
		t,
		ts,
		product.ID,
		2,
		"idempotency-get-purchase",
	)

	request, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf(
			"%s/api/v1/purchases/%d",
			ts.server.URL,
			purchase.ID,
		),
		nil,
	)
	if err != nil {
		t.Fatalf("create get purchase request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("get purchase request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusOK,
			response.StatusCode,
			string(responseBody),
		)
	}

	var result purchaseResponse

	if err := json.NewDecoder(
		response.Body,
	).Decode(&result); err != nil {
		t.Fatalf("decode purchase response: %v", err)
	}

	if result.ID != purchase.ID {
		t.Fatalf(
			"expected purchase ID %d, got %d",
			purchase.ID,
			result.ID,
		)
	}

	if result.Total != "24.00" {
		t.Fatalf(
			"expected total 24.00, got %s",
			result.Total,
		)
	}
}

func TestIntegration_GetPurchases(t *testing.T) {
	ts := newTestServer(t)

	product := createProduct(
		t,
		ts,
		"Purchase List Product",
		newSKU(t),
		"10.00",
		20,
	)

	createPurchase(
		t,
		ts,
		product.ID,
		1,
		"idempotency-list-1",
	)

	createPurchase(
		t,
		ts,
		product.ID,
		2,
		"idempotency-list-2",
	)

	request, err := http.NewRequest(
		http.MethodGet,
		ts.server.URL+"/api/v1/purchases",
		nil,
	)
	if err != nil {
		t.Fatalf("create purchases request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("get purchases request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusOK,
			response.StatusCode,
			string(responseBody),
		)
	}

	var purchases []purchaseResponse

	if err := json.NewDecoder(
		response.Body,
	).Decode(&purchases); err != nil {
		t.Fatalf("decode purchases response: %v", err)
	}

	if len(purchases) != 2 {
		t.Fatalf(
			"expected 2 purchases, got %d",
			len(purchases),
		)
	}
}

func TestIntegration_GetNonExistentProduct(t *testing.T) {
	ts := newTestServer(t)

	request, err := http.NewRequest(
		http.MethodGet,
		ts.server.URL+"/api/v1/products/999999",
		nil,
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			response.StatusCode,
		)
	}
}

func TestIntegration_InvalidProductID(t *testing.T) {
	ts := newTestServer(t)

	request, err := http.NewRequest(
		http.MethodGet,
		ts.server.URL+"/api/v1/products/invalid",
		nil,
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			response.StatusCode,
		)
	}
}

func TestIntegration_MissingIdempotencyKey(t *testing.T) {
	ts := newTestServer(t)

	product := createProduct(
		t,
		ts,
		"Missing Idempotency Product",
		newSKU(t),
		"10.00",
		10,
	)

	requestBody := map[string]any{
		"items": []map[string]any{
			{
				"product_id": product.ID,
				"quantity":   1,
			},
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("marshal purchase request: %v", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		ts.server.URL+"/api/v1/purchases",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("create purchase request: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("purchase request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		responseBody, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusBadRequest,
			response.StatusCode,
			string(responseBody),
		)
	}
}

func TestIntegration_GetProductList(t *testing.T) {
	ts := newTestServer(t)

	createProduct(
		t,
		ts,
		"Product One",
		newSKU(t),
		"10.00",
		5,
	)

	createProduct(
		t,
		ts,
		"Product Two",
		newSKU(t),
		"20.00",
		10,
	)

	request, err := http.NewRequest(
		http.MethodGet,
		ts.server.URL+"/api/v1/products",
		nil,
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("get products request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			response.StatusCode,
		)
	}

	var products []productResponse

	if err := json.NewDecoder(
		response.Body,
	).Decode(&products); err != nil {
		t.Fatalf("decode products response: %v", err)
	}

	if len(products) != 2 {
		t.Fatalf(
			"expected 2 products, got %d",
			len(products),
		)
	}
}

func TestIntegration_PurchaseTotalAndStock(t *testing.T) {
	ts := newTestServer(t)

	product := createProduct(
		t,
		ts,
		"Stock Product",
		newSKU(t),
		"7.50",
		10,
	)

	purchase := createPurchase(
		t,
		ts,
		product.ID,
		4,
		"stock-test-key",
	)

	expectedTotal := "30.00"

	if purchase.Total != expectedTotal {
		t.Fatalf(
			"expected total %s, got %s",
			expectedTotal,
			purchase.Total,
		)
	}

	request, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf(
			"%s/api/v1/products/%d",
			ts.server.URL,
			product.ID,
		),
		nil,
	)
	if err != nil {
		t.Fatalf("create product request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("get product request: %v", err)
	}
	defer response.Body.Close()

	var result productResponse

	if err := json.NewDecoder(
		response.Body,
	).Decode(&result); err != nil {
		t.Fatalf("decode product response: %v", err)
	}

	expectedStock := 6

	if result.Stock != expectedStock {
		t.Fatalf(
			"expected stock %d, got %d",
			expectedStock,
			result.Stock,
		)
	}
}

func TestIntegration_PurchaseWithoutItems(t *testing.T) {
	ts := newTestServer(t)

	requestBody := map[string]any{
		"items": []any{},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		ts.server.URL+"/api/v1/purchases",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "empty-items-key")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("purchase request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		responseBody, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusBadRequest,
			response.StatusCode,
			string(responseBody),
		)
	}
}

func TestIntegration_PurchaseInvalidQuantity(t *testing.T) {
	ts := newTestServer(t)

	product := createProduct(
		t,
		ts,
		"Invalid Quantity Product",
		newSKU(t),
		"10.00",
		10,
	)

	requestBody := map[string]any{
		"items": []map[string]any{
			{
				"product_id": product.ID,
				"quantity":   0,
			},
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		ts.server.URL+"/api/v1/purchases",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "invalid-quantity-key")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("purchase request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		responseBody, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusBadRequest,
			response.StatusCode,
			string(responseBody),
		)
	}
}

func TestIntegration_PurchaseResponseJSON(t *testing.T) {
	ts := newTestServer(t)

	product := createProduct(
		t,
		ts,
		"JSON Product",
		newSKU(t),
		"100.00",
		5,
	)

	purchase := createPurchase(
		t,
		ts,
		product.ID,
		1,
		"json-response-key",
	)

	if purchase.ID <= 0 {
		t.Fatalf(
			"expected a valid purchase ID, got %d",
			purchase.ID,
		)
	}

	if purchase.Status == "" {
		t.Fatal("expected purchase status")
	}

	if purchase.Total == "" {
		t.Fatal("expected purchase total")
	}

	if len(purchase.Items) != 1 {
		t.Fatalf(
			"expected one purchase item, got %d",
			len(purchase.Items),
		)
	}
}

func TestIntegration_ProductResponseJSON(t *testing.T) {
	ts := newTestServer(t)

	product := createProduct(
		t,
		ts,
		"JSON Product",
		newSKU(t),
		"99.99",
		7,
	)

	if product.ID <= 0 {
		t.Fatalf(
			"expected a valid product ID, got %d",
			product.ID,
		)
	}

	if product.Name == "" {
		t.Fatal("expected product name")
	}

	if product.SKU == "" {
		t.Fatal("expected product SKU")
	}

	if product.Price != "99.99" {
		t.Fatalf(
			"expected price 99.99, got %s",
			product.Price,
		)
	}

	if product.Stock != 7 {
		t.Fatalf(
			"expected stock 7, got %d",
			product.Stock,
		)
	}
}

func TestIntegration_ProductImportMissingFile(t *testing.T) {
	ts := newTestServer(t)

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("other", "value"); err != nil {
		t.Fatalf("write multipart field: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		ts.server.URL+"/api/v1/products/import",
		&body,
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	request.Header.Set(
		"Content-Type",
		writer.FormDataContentType(),
	)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("import request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			response.StatusCode,
		)
	}
}

func TestIntegration_ProductInvalidJSON(t *testing.T) {
	ts := newTestServer(t)

	request, err := http.NewRequest(
		http.MethodPost,
		ts.server.URL+"/api/v1/products",
		bytes.NewBufferString(`{"name":`),
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("create product request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			response.StatusCode,
		)
	}
}

func TestIntegration_ProductUpdateNotFound(t *testing.T) {
	ts := newTestServer(t)

	requestBody := map[string]any{
		"name":        "Updated Product",
		"sku":         newSKU(t),
		"description": "Description",
		"category":    "TEST",
		"price":       "10.00",
		"stock":       5,
		"weight_kg":   "1.000",
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	request, err := http.NewRequest(
		http.MethodPut,
		ts.server.URL+"/api/v1/products/999999",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("update product request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			response.StatusCode,
		)
	}
}

func TestIntegration_DeleteProductNotFound(t *testing.T) {
	ts := newTestServer(t)

	request, err := http.NewRequest(
		http.MethodDelete,
		ts.server.URL+"/api/v1/products/999999",
		nil,
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("delete product request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			response.StatusCode,
		)
	}
}

func TestIntegration_PurchaseNotFound(t *testing.T) {
	ts := newTestServer(t)

	request, err := http.NewRequest(
		http.MethodGet,
		ts.server.URL+"/api/v1/purchases/999999",
		nil,
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("get purchase request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNotFound,
			response.StatusCode,
		)
	}
}

func TestIntegration_PurchaseInvalidID(t *testing.T) {
	ts := newTestServer(t)

	request, err := http.NewRequest(
		http.MethodGet,
		ts.server.URL+"/api/v1/purchases/invalid",
		nil,
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("get purchase request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			response.StatusCode,
		)
	}
}

func TestIntegration_ProductImportInvalidCSV(t *testing.T) {
	ts := newTestServer(t)

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)

	fileWriter, err := writer.CreateFormFile(
		"file",
		"invalid.csv",
	)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}

	fileContent := []byte(
		"invalid,header\ninvalid,data\n",
	)

	if _, err := fileWriter.Write(fileContent); err != nil {
		t.Fatalf("write invalid CSV: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		ts.server.URL+"/api/v1/products/import",
		&body,
	)
	if err != nil {
		t.Fatalf("create import request: %v", err)
	}

	request.Header.Set(
		"Content-Type",
		writer.FormDataContentType(),
	)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("import request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			response.StatusCode,
		)
	}
}

func TestIntegration_ProductImportDuplicateSKU(t *testing.T) {
	ts := newTestServer(t)

	sku := newSKU(t)

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)

	fileWriter, err := writer.CreateFormFile(
		"file",
		"duplicate.csv",
	)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}

	csvContent := fmt.Sprintf(
		"name,sku,description,category,price,stock,weight_kg\n"+
			"Product 1,%s,Description,TEST,10.00,10,1.000\n"+
			"Product 2,%s,Description,TEST,20.00,10,2.000\n",
		sku,
		sku,
	)

	if _, err := fileWriter.Write(
		[]byte(csvContent),
	); err != nil {
		t.Fatalf("write CSV: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		ts.server.URL+"/api/v1/products/import",
		&body,
	)
	if err != nil {
		t.Fatalf("create import request: %v", err)
	}

	request.Header.Set(
		"Content-Type",
		writer.FormDataContentType(),
	)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("import request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		responseBody, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusBadRequest,
			response.StatusCode,
			string(responseBody),
		)
	}
}

func TestIntegration_ProductSearchEmptyQuery(t *testing.T) {
	ts := newTestServer(t)

	request, err := http.NewRequest(
		http.MethodGet,
		ts.server.URL+"/api/v1/products/search?q=",
		nil,
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("search request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			response.StatusCode,
		)
	}
}

func TestIntegration_PurchaseDuplicateProducts(t *testing.T) {
	ts := newTestServer(t)

	product := createProduct(
		t,
		ts,
		"Duplicate Item Product",
		newSKU(t),
		"10.00",
		10,
	)

	requestBody := map[string]any{
		"items": []map[string]any{
			{
				"product_id": product.ID,
				"quantity":   1,
			},
			{
				"product_id": product.ID,
				"quantity":   2,
			},
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		ts.server.URL+"/api/v1/purchases",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(
		"Idempotency-Key",
		"duplicate-products-key",
	)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("purchase request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		responseBody, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusBadRequest,
			response.StatusCode,
			string(responseBody),
		)
	}
}

func TestIntegration_ProductCreateInvalidData(t *testing.T) {
	ts := newTestServer(t)

	requestBody := map[string]any{
		"name":        "",
		"sku":         "",
		"description": "",
		"category":    "",
		"price":       "invalid",
		"stock":       -1,
		"weight_kg":   "invalid",
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		ts.server.URL+"/api/v1/products",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("create product request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		responseBody, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusBadRequest,
			response.StatusCode,
			string(responseBody),
		)
	}
}

func TestIntegration_PurchaseMissingProductID(t *testing.T) {
	ts := newTestServer(t)

	requestBody := map[string]any{
		"items": []map[string]any{
			{
				"product_id": 0,
				"quantity":   1,
			},
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		ts.server.URL+"/api/v1/purchases",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(
		"Idempotency-Key",
		"invalid-product-id-key",
	)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("purchase request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		responseBody, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"expected status %d, got %d: %s",
			http.StatusBadRequest,
			response.StatusCode,
			string(responseBody),
		)
	}
}

func TestIntegration_ProductIDParsing(t *testing.T) {
	ts := newTestServer(t)

	tests := []struct {
		name string
		path string
	}{
		{
			name: "zero",
			path: "/api/v1/products/0",
		},
		{
			name: "negative",
			path: "/api/v1/products/-1",
		},
		{
			name: "non numeric",
			path: "/api/v1/products/abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, err := http.NewRequest(
				http.MethodGet,
				ts.server.URL+tt.path,
				nil,
			)
			if err != nil {
				t.Fatalf("create request: %v", err)
			}

			response, err := http.DefaultClient.Do(request)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer response.Body.Close()

			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf(
					"expected status %d, got %d",
					http.StatusBadRequest,
					response.StatusCode,
				)
			}
		})
	}
}

func TestIntegration_PurchaseIDParsing(t *testing.T) {
	ts := newTestServer(t)

	tests := []struct {
		name string
		path string
	}{
		{
			name: "zero",
			path: "/api/v1/purchases/0",
		},
		{
			name: "negative",
			path: "/api/v1/purchases/-1",
		},
		{
			name: "non numeric",
			path: "/api/v1/purchases/abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, err := http.NewRequest(
				http.MethodGet,
				ts.server.URL+tt.path,
				nil,
			)
			if err != nil {
				t.Fatalf("create request: %v", err)
			}

			response, err := http.DefaultClient.Do(request)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer response.Body.Close()

			if response.StatusCode != http.StatusBadRequest {
				t.Fatalf(
					"expected status %d, got %d",
					http.StatusBadRequest,
					response.StatusCode,
				)
			}
		})
	}
}

func TestIntegration_ProductImportEmptyCSV(t *testing.T) {
	ts := newTestServer(t)

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)

	fileWriter, err := writer.CreateFormFile(
		"file",
		"empty.csv",
	)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}

	csvContent := "name,sku,description,category,price,stock,weight_kg\n"

	if _, err := fileWriter.Write(
		[]byte(csvContent),
	); err != nil {
		t.Fatalf("write CSV: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		ts.server.URL+"/api/v1/products/import",
		&body,
	)
	if err != nil {
		t.Fatalf("create import request: %v", err)
	}

	request.Header.Set(
		"Content-Type",
		writer.FormDataContentType(),
	)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("import request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			response.StatusCode,
		)
	}
}

func TestIntegration_ProductCreateAndList(t *testing.T) {
	ts := newTestServer(t)

	for i := 1; i <= 3; i++ {
		createProduct(
			t,
			ts,
			"Product "+strconv.Itoa(i),
			newSKU(t),
			"10.00",
			i,
		)
	}

	request, err := http.NewRequest(
		http.MethodGet,
		ts.server.URL+"/api/v1/products",
		nil,
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("get products request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			response.StatusCode,
		)
	}

	var products []productResponse

	if err := json.NewDecoder(
		response.Body,
	).Decode(&products); err != nil {
		t.Fatalf("decode products: %v", err)
	}

	if len(products) != 3 {
		t.Fatalf(
			"expected 3 products, got %d",
			len(products),
		)
	}
}

func TestIntegration_ErrorResponseIsJSON(t *testing.T) {
	ts := newTestServer(t)

	request, err := http.NewRequest(
		http.MethodGet,
		ts.server.URL+"/api/v1/products/invalid",
		nil,
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			response.StatusCode,
		)
	}

	if response.Header.Get("Content-Type") != "application/json" {
		t.Fatalf(
			"expected JSON content type, got %s",
			response.Header.Get("Content-Type"),
		)
	}

	var result errorResponse

	if err := json.NewDecoder(
		response.Body,
	).Decode(&result); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	if result.Error == "" {
		t.Fatal("expected error message")
	}
}

func TestIntegration_PurchaseStockNotChangedAfterPaymentFailure(t *testing.T) {
	ts := newTestServer(t)

	product := createProduct(
		t,
		ts,
		"Payment Stock Product",
		newSKU(t),
		"25.00",
		8,
	)

	requestBody := map[string]any{
		"items": []map[string]any{
			{
				"product_id": product.ID,
				"quantity":   3,
			},
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		ts.server.URL+"/api/v1/purchases",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(
		"Idempotency-Key",
		"payment-failure-stock-key",
	)
	request.Header.Set(
		"X-Fake-Payment",
		"failure",
	)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("purchase request: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusPaymentRequired {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusPaymentRequired,
			response.StatusCode,
		)
	}

	getRequest, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf(
			"%s/api/v1/products/%d",
			ts.server.URL,
			product.ID,
		),
		nil,
	)
	if err != nil {
		t.Fatalf("create product request: %v", err)
	}

	getResponse, err := http.DefaultClient.Do(getRequest)
	if err != nil {
		t.Fatalf("get product request: %v", err)
	}
	defer getResponse.Body.Close()

	var result productResponse

	if err := json.NewDecoder(
		getResponse.Body,
	).Decode(&result); err != nil {
		t.Fatalf("decode product response: %v", err)
	}

	if result.Stock != 8 {
		t.Fatalf(
			"expected stock 8, got %d",
			result.Stock,
		)
	}
}
