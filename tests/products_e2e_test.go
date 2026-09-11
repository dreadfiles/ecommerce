package tests

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
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

type productsE2ETestEnvironment struct {
	server *httptest.Server
	db     *sql.DB
}

func productsE2ESetup(t *testing.T) *productsE2ETestEnvironment {
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

	environment := &productsE2ETestEnvironment{
		server: server,
		db:     db,
	}

	t.Cleanup(func() {
		server.Close()
		db.Close()
	})

	return environment
}

func productsE2EResetDatabase(
	t *testing.T,
	environment *productsE2ETestEnvironment,
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

func productsE2ERequest(
	t *testing.T,
	environment *productsE2ETestEnvironment,
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

func productsE2EDecodeJSON(
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

func productsE2EAssertStatus(
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

func productsE2ECreateProduct(
	t *testing.T,
	environment *productsE2ETestEnvironment,
	name string,
	sku string,
	price any,
	stock int,
) int64 {
	t.Helper()

	body := map[string]any{
		"name":        name,
		"sku":         sku,
		"description": "E2E test product",
		"category":    "TEST",
		"price":       fmt.Sprintf("%v", price),
		"stock":       stock,
		"weight_kg":   "1.000",
	}

	response := productsE2ERequest(
		t,
		environment,
		http.MethodPost,
		"/api/v1/products",
		body,
		nil,
	)

	if response.StatusCode != http.StatusCreated {
		defer response.Body.Close()

		bodyBytes, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"create product: expected status %d, got %d, body: %s",
			http.StatusCreated,
			response.StatusCode,
			string(bodyBytes),
		)
	}

	var product struct {
		ID int64 `json:"id"`
	}

	productsE2EDecodeJSON(
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

func productsE2EFormatID(id int64) string {
	return strconv.FormatInt(id, 10)
}

func TestProductsE2E_CreateAndGetProduct(t *testing.T) {
	environment := productsE2ESetup(t)

	productsE2EResetDatabase(
		t,
		environment,
	)

	createBody := map[string]any{
		"name":        "Laptop",
		"sku":         "LAPTOP-001",
		"description": "Integration test laptop",
		"category":    "Electronics",
		"price":       "2500.00",
		"stock":       10,
		"weight_kg":   "2.500",
	}

	createResponse := productsE2ERequest(
		t,
		environment,
		http.MethodPost,
		"/api/v1/products",
		createBody,
		nil,
	)

	if createResponse.StatusCode != http.StatusCreated {
		defer createResponse.Body.Close()

		body, _ := io.ReadAll(createResponse.Body)

		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusCreated,
			createResponse.StatusCode,
			string(body),
		)
	}

	var createdProduct struct {
		ID          int64  `json:"id"`
		Name        string `json:"name"`
		SKU         string `json:"sku"`
		Description string `json:"description"`
		Category    string `json:"category"`
		Price       string `json:"price"`
		Stock       int    `json:"stock"`
		WeightKg    string `json:"weight_kg"`
	}

	productsE2EDecodeJSON(
		t,
		createResponse,
		&createdProduct,
	)

	if createdProduct.ID <= 0 {
		t.Fatalf(
			"expected generated product ID, got %d",
			createdProduct.ID,
		)
	}

	if createdProduct.Name != "Laptop" {
		t.Fatalf(
			"expected name Laptop, got %q",
			createdProduct.Name,
		)
	}

	if createdProduct.SKU != "LAPTOP-001" {
		t.Fatalf(
			"expected SKU LAPTOP-001, got %q",
			createdProduct.SKU,
		)
	}

	getResponse := productsE2ERequest(
		t,
		environment,
		http.MethodGet,
		"/api/v1/products/"+productsE2EFormatID(createdProduct.ID),
		nil,
		nil,
	)

	if getResponse.StatusCode != http.StatusOK {
		defer getResponse.Body.Close()

		body, _ := io.ReadAll(getResponse.Body)

		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusOK,
			getResponse.StatusCode,
			string(body),
		)
	}

	var fetchedProduct struct {
		ID    int64  `json:"id"`
		SKU   string `json:"sku"`
		Price string `json:"price"`
		Stock int    `json:"stock"`
	}

	productsE2EDecodeJSON(
		t,
		getResponse,
		&fetchedProduct,
	)

	if fetchedProduct.ID != createdProduct.ID {
		t.Fatalf(
			"expected product ID %d, got %d",
			createdProduct.ID,
			fetchedProduct.ID,
		)
	}

	if fetchedProduct.SKU != "LAPTOP-001" {
		t.Fatalf(
			"expected SKU LAPTOP-001, got %q",
			fetchedProduct.SKU,
		)
	}

	if fetchedProduct.Price != "2500.00" {
		t.Fatalf(
			"expected price 2500.00, got %q",
			fetchedProduct.Price,
		)
	}

	if fetchedProduct.Stock != 10 {
		t.Fatalf(
			"expected stock 10, got %d",
			fetchedProduct.Stock,
		)
	}
}

func TestProductsE2E_GetAllProducts(t *testing.T) {
	environment := productsE2ESetup(t)

	productsE2EResetDatabase(
		t,
		environment,
	)

	productsE2ECreateProduct(
		t,
		environment,
		"Product A",
		"SKU-A",
		"100.00",
		5,
	)

	productsE2ECreateProduct(
		t,
		environment,
		"Product B",
		"SKU-B",
		"200.00",
		10,
	)

	response := productsE2ERequest(
		t,
		environment,
		http.MethodGet,
		"/api/v1/products",
		nil,
		nil,
	)

	if response.StatusCode != http.StatusOK {
		defer response.Body.Close()

		body, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusOK,
			response.StatusCode,
			string(body),
		)
	}

	var products []struct {
		ID  int64  `json:"id"`
		SKU string `json:"sku"`
	}

	productsE2EDecodeJSON(
		t,
		response,
		&products,
	)

	if len(products) != 2 {
		t.Fatalf(
			"expected 2 products, got %d",
			len(products),
		)
	}

	if products[0].SKU != "SKU-A" {
		t.Fatalf(
			"expected first SKU SKU-A, got %q",
			products[0].SKU,
		)
	}

	if products[1].SKU != "SKU-B" {
		t.Fatalf(
			"expected second SKU SKU-B, got %q",
			products[1].SKU,
		)
	}
}

func TestProductsE2E_SearchProducts(t *testing.T) {
	environment := productsE2ESetup(t)

	productsE2EResetDatabase(
		t,
		environment,
	)

	productsE2ECreateProduct(
		t,
		environment,
		"Mechanical Keyboard",
		"KEYBOARD-001",
		"150.00",
		20,
	)

	productsE2ECreateProduct(
		t,
		environment,
		"Wireless Mouse",
		"MOUSE-001",
		"80.00",
		30,
	)

	response := productsE2ERequest(
		t,
		environment,
		http.MethodGet,
		"/api/v1/products/search?q=keyboard",
		nil,
		nil,
	)

	if response.StatusCode != http.StatusOK {
		defer response.Body.Close()

		body, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusOK,
			response.StatusCode,
			string(body),
		)
	}

	var products []struct {
		SKU string `json:"sku"`
	}

	productsE2EDecodeJSON(
		t,
		response,
		&products,
	)

	if len(products) != 1 {
		t.Fatalf(
			"expected 1 search result, got %d",
			len(products),
		)
	}

	if products[0].SKU != "KEYBOARD-001" {
		t.Fatalf(
			"expected KEYBOARD-001, got %q",
			products[0].SKU,
		)
	}
}

func TestProductsE2E_UpdateProduct(t *testing.T) {
	environment := productsE2ESetup(t)

	productsE2EResetDatabase(
		t,
		environment,
	)

	productID := productsE2ECreateProduct(
		t,
		environment,
		"Old Product",
		"UPDATE-001",
		"100.00",
		5,
	)

	updateBody := map[string]any{
		"name":        "Updated Product",
		"sku":         "UPDATE-001",
		"description": "Updated description",
		"category":    "UPDATED",
		"price":       "150.00",
		"stock":       8,
		"weight_kg":   "1.500",
	}

	response := productsE2ERequest(
		t,
		environment,
		http.MethodPut,
		"/api/v1/products/"+productsE2EFormatID(productID),
		updateBody,
		nil,
	)

	if response.StatusCode != http.StatusOK {
		defer response.Body.Close()

		body, _ := io.ReadAll(response.Body)

		t.Fatalf(
			"expected status %d, got %d, body: %s",
			http.StatusOK,
			response.StatusCode,
			string(body),
		)
	}

	var product struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Price string `json:"price"`
		Stock int    `json:"stock"`
	}

	productsE2EDecodeJSON(
		t,
		response,
		&product,
	)

	if product.ID != productID {
		t.Fatalf(
			"expected ID %d, got %d",
			productID,
			product.ID,
		)
	}

	if product.Name != "Updated Product" {
		t.Fatalf(
			"expected Updated Product, got %q",
			product.Name,
		)
	}

	if product.Price != "150.00" {
		t.Fatalf(
			"expected price 150.00, got %q",
			product.Price,
		)
	}

	if product.Stock != 8 {
		t.Fatalf(
			"expected stock 8, got %d",
			product.Stock,
		)
	}
}

func TestProductsE2E_DeleteProduct(t *testing.T) {
	environment := productsE2ESetup(t)

	productsE2EResetDatabase(
		t,
		environment,
	)

	productID := productsE2ECreateProduct(
		t,
		environment,
		"Delete Product",
		"DELETE-001",
		"100.00",
		5,
	)

	deleteResponse := productsE2ERequest(
		t,
		environment,
		http.MethodDelete,
		"/api/v1/products/"+productsE2EFormatID(productID),
		nil,
		nil,
	)

	productsE2EAssertStatus(
		t,
		deleteResponse,
		http.StatusNoContent,
	)

	getResponse := productsE2ERequest(
		t,
		environment,
		http.MethodGet,
		"/api/v1/products/"+productsE2EFormatID(productID),
		nil,
		nil,
	)

	productsE2EAssertStatus(
		t,
		getResponse,
		http.StatusNotFound,
	)
}

func TestProductsE2E_DuplicateSKU(t *testing.T) {
	environment := productsE2ESetup(t)

	productsE2EResetDatabase(
		t,
		environment,
	)

	productsE2ECreateProduct(
		t,
		environment,
		"First Product",
		"DUPLICATE-001",
		"100.00",
		5,
	)

	body := map[string]any{
		"name":        "Second Product",
		"sku":         "DUPLICATE-001",
		"description": "Duplicate SKU test",
		"category":    "TEST",
		"price":       "200.00",
		"stock":       5,
		"weight_kg":   "1.000",
	}

	response := productsE2ERequest(
		t,
		environment,
		http.MethodPost,
		"/api/v1/products",
		body,
		nil,
	)

	productsE2EAssertStatus(
		t,
		response,
		http.StatusConflict,
	)
}

func TestProductsE2E_InvalidProduct(t *testing.T) {
	environment := productsE2ESetup(t)

	productsE2EResetDatabase(
		t,
		environment,
	)

	body := map[string]any{
		"name":        "",
		"sku":         "",
		"description": "",
		"category":    "",
		"price":       "-10.00",
		"stock":       -1,
		"weight_kg":   "",
	}

	response := productsE2ERequest(
		t,
		environment,
		http.MethodPost,
		"/api/v1/products",
		body,
		nil,
	)

	productsE2EAssertStatus(
		t,
		response,
		http.StatusBadRequest,
	)
}
