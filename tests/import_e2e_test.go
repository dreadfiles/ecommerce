package tests

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"mime/multipart"
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

var (
	importE2EServer *httptest.Server
	importE2EDB     *sql.DB
)

func importE2EHandler(t *testing.T) http.Handler {
	t.Helper()

	productRepository := productrepository.NewPostgresProductRepository(
		importE2EDatabase(t),
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
		importE2EDatabase(t),
	)

	purchaseService := purchaseservice.NewPurchaseService(
		purchaseRepository,
	)

	purchaseHandler := purchasehandler.NewPurchaseHandler(
		purchaseService,
	)

	return router.New(
		router.Dependencies{
			ProductHandler:  productHandler,
			ImportHandler:   productImportHandler,
			PurchaseHandler: purchaseHandler,
		},
	)
}

func importE2EServerInstance(t *testing.T) *httptest.Server {
	t.Helper()

	if importE2EServer == nil {
		importE2EServer = httptest.NewServer(
			importE2EHandler(t),
		)

		t.Cleanup(func() {
			importE2EServer.Close()
			importE2EServer = nil
		})
	}

	return importE2EServer
}

func importE2EDatabase(t *testing.T) *sql.DB {
	t.Helper()

	if importE2EDB == nil {
		db, err := database.NewPostgresDB()
		if err != nil {
			t.Fatalf(
				"connect to postgres: %v",
				err,
			)
		}

		importE2EDB = db

		t.Cleanup(func() {
			importE2EDB.Close()
			importE2EDB = nil
		})
	}

	return importE2EDB
}

func importE2EResetDatabase(t *testing.T) {
	t.Helper()

	db := importE2EDatabase(t)

	queries := []string{
		"DELETE FROM order_items",
		"DELETE FROM orders",
		"DELETE FROM products",
	}

	for _, query := range queries {
		if _, err := db.ExecContext(
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

func importE2EUploadCSV(
	t *testing.T,
	csvContent string,
) *http.Response {
	t.Helper()

	server := importE2EServerInstance(t)

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)

	fileWriter, err := writer.CreateFormFile(
		"file",
		"products.csv",
	)
	if err != nil {
		t.Fatalf(
			"create multipart file: %v",
			err,
		)
	}

	if _, err := fileWriter.Write(
		[]byte(csvContent),
	); err != nil {
		t.Fatalf(
			"write csv content: %v",
			err,
		)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf(
			"close multipart writer: %v",
			err,
		)
	}

	request, err := http.NewRequest(
		http.MethodPost,
		server.URL+"/api/v1/products/import",
		&body,
	)
	if err != nil {
		t.Fatalf(
			"create import request: %v",
			err,
		)
	}

	request.Header.Set(
		"Content-Type",
		writer.FormDataContentType(),
	)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf(
			"execute import request: %v",
			err,
		)
	}

	return response
}

func importE2ERequest(
	t *testing.T,
	method string,
	path string,
	body io.Reader,
) *http.Response {
	t.Helper()

	server := importE2EServerInstance(t)

	request, err := http.NewRequest(
		method,
		server.URL+path,
		body,
	)
	if err != nil {
		t.Fatalf(
			"create request: %v",
			err,
		)
	}

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf(
			"execute request: %v",
			err,
		)
	}

	return response
}

func importE2EAssertStatus(
	t *testing.T,
	response *http.Response,
	expected int,
) {
	t.Helper()

	if response.StatusCode != expected {
		body, _ := io.ReadAll(response.Body)

		response.Body.Close()

		t.Fatalf(
			"expected status %d, got %d, body: %s",
			expected,
			response.StatusCode,
			string(body),
		)
	}
}

func importE2EDecodeJSON(
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
			"decode json response: %v",
			err,
		)
	}
}

func TestProductImportE2E_ImportProducts(t *testing.T) {
	importE2EResetDatabase(t)

	csvContent := `name,sku,description,category,price,stock,weight_kg
Laptop,SKU-IMPORT-001,Professional laptop,Electronics,1500.00,10,2.500
Mouse,SKU-IMPORT-002,Wireless mouse,Accessories,35.50,50,0.250
`

	response := importE2EUploadCSV(
		t,
		csvContent,
	)

	importE2EAssertStatus(
		t,
		response,
		http.StatusCreated,
	)

	response = importE2ERequest(
		t,
		http.MethodGet,
		"/api/v1/products",
		nil,
	)

	importE2EAssertStatus(
		t,
		response,
		http.StatusOK,
	)

	var products []struct {
		ID          int64  `json:"id"`
		Name        string `json:"name"`
		SKU         string `json:"sku"`
		Description string `json:"description"`
		Category    string `json:"category"`
		Price       string `json:"price"`
		Stock       int    `json:"stock"`
		WeightKg    string `json:"weight_kg"`
	}

	importE2EDecodeJSON(
		t,
		response,
		&products,
	)

	if len(products) != 2 {
		t.Fatalf(
			"expected 2 imported products, got %d",
			len(products),
		)
	}

	if products[0].SKU != "SKU-IMPORT-001" {
		t.Fatalf(
			"expected first SKU SKU-IMPORT-001, got %s",
			products[0].SKU,
		)
	}

	if products[1].SKU != "SKU-IMPORT-002" {
		t.Fatalf(
			"expected second SKU SKU-IMPORT-002, got %s",
			products[1].SKU,
		)
	}
}

func TestProductImportE2E_InvalidCSV(t *testing.T) {
	importE2EResetDatabase(t)

	csvContent := `name,sku,description,category,price,stock,weight_kg
Laptop,SKU-INVALID-001,Professional laptop,Electronics,1500.00,not-a-number,2.500
`

	response := importE2EUploadCSV(
		t,
		csvContent,
	)

	importE2EAssertStatus(
		t,
		response,
		http.StatusBadRequest,
	)

	response = importE2ERequest(
		t,
		http.MethodGet,
		"/api/v1/products",
		nil,
	)

	importE2EAssertStatus(
		t,
		response,
		http.StatusOK,
	)

	var products []any

	importE2EDecodeJSON(
		t,
		response,
		&products,
	)

	if len(products) != 0 {
		t.Fatalf(
			"expected no products after invalid import, got %d",
			len(products),
		)
	}
}

func TestProductImportE2E_DuplicateSKUInCSV(t *testing.T) {
	importE2EResetDatabase(t)

	csvContent := `name,sku,description,category,price,stock,weight_kg
Laptop,SKU-DUP-001,Professional laptop,Electronics,1500.00,10,2.500
Laptop Pro,SKU-DUP-001,Professional laptop,Electronics,2000.00,5,2.800
`

	response := importE2EUploadCSV(
		t,
		csvContent,
	)

	importE2EAssertStatus(
		t,
		response,
		http.StatusBadRequest,
	)

	response = importE2ERequest(
		t,
		http.MethodGet,
		"/api/v1/products",
		nil,
	)

	importE2EAssertStatus(
		t,
		response,
		http.StatusOK,
	)

	var products []any

	importE2EDecodeJSON(
		t,
		response,
		&products,
	)

	if len(products) != 0 {
		t.Fatalf(
			"expected no products after duplicate SKU import, got %d",
			len(products),
		)
	}
}

func TestProductImportE2E_DuplicateSKUAlreadyExists(t *testing.T) {
	importE2EResetDatabase(t)

	productJSON := []byte(`{
		"name": "Existing Product",
		"sku": "SKU-EXISTING-001",
		"description": "Existing product",
		"category": "Electronics",
		"price": "100.00",
		"stock": 10,
		"weight_kg": "1.000"
	}`)

	response := importE2ERequest(
		t,
		http.MethodPost,
		"/api/v1/products",
		bytes.NewReader(productJSON),
	)

	importE2EAssertStatus(
		t,
		response,
		http.StatusCreated,
	)

	csvContent := `name,sku,description,category,price,stock,weight_kg
Imported Product,SKU-EXISTING-001,Imported product,Electronics,150.00,5,1.500
`

	response = importE2EUploadCSV(
		t,
		csvContent,
	)

	importE2EAssertStatus(
		t,
		response,
		http.StatusBadRequest,
	)

	response = importE2ERequest(
		t,
		http.MethodGet,
		"/api/v1/products",
		nil,
	)

	importE2EAssertStatus(
		t,
		response,
		http.StatusOK,
	)

	var products []struct {
		SKU string `json:"sku"`
	}

	importE2EDecodeJSON(
		t,
		response,
		&products,
	)

	if len(products) != 1 {
		t.Fatalf(
			"expected 1 product after duplicate import, got %d",
			len(products),
		)
	}

	if products[0].SKU != "SKU-EXISTING-001" {
		t.Fatalf(
			"expected existing product to remain, got SKU %s",
			products[0].SKU,
		)
	}
}

func TestProductImportE2E_MissingFile(t *testing.T) {
	importE2EResetDatabase(t)

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)

	if err := writer.Close(); err != nil {
		t.Fatalf(
			"close multipart writer: %v",
			err,
		)
	}

	server := importE2EServerInstance(t)

	request, err := http.NewRequest(
		http.MethodPost,
		server.URL+"/api/v1/products/import",
		&body,
	)
	if err != nil {
		t.Fatalf(
			"create request: %v",
			err,
		)
	}

	request.Header.Set(
		"Content-Type",
		writer.FormDataContentType(),
	)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf(
			"execute request: %v",
			err,
		)
	}

	importE2EAssertStatus(
		t,
		response,
		http.StatusBadRequest,
	)

	response.Body.Close()
}
