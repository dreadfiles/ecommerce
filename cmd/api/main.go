package main

import (
	"log"
	"net/http"

	"ecommerce/internal/database"
	importerhandler "ecommerce/internal/importer/handler"
	importerservice "ecommerce/internal/importer/service"
	producthandler "ecommerce/internal/product/handler"
	productrepository "ecommerce/internal/product/repository"
	productservice "ecommerce/internal/product/service"
	purchasehandler "ecommerce/internal/purchase/handler"
	purchaserepository "ecommerce/internal/purchase/repository"
	purchaseservice "ecommerce/internal/purchase/service"
)

const serverAddress = ":8080"

func main() {
	db, err := database.NewPostgresDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	productRepository := productrepository.NewPostgresProductRepository(db)
	productService := productservice.NewProductService(productRepository)
	productHandler := producthandler.NewProductHandler(productService)

	productImportService := importerservice.NewProductImportService(productRepository)
	productImportHandler := importerhandler.NewProductImportHandler(productImportService)

	purchaseRepository := purchaserepository.NewPostgresPurchaseRepository(db)
	purchaseService := purchaseservice.NewPurchaseService(purchaseRepository)
	purchaseHandler := purchasehandler.NewPurchaseHandler(purchaseService)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/v1/products", productHandler.Create)
	mux.HandleFunc("GET /api/v1/products", productHandler.GetAll)
	mux.HandleFunc("GET /api/v1/products/search", productHandler.Search)
	mux.HandleFunc("GET /api/v1/products/{id}", productHandler.GetByID)
	mux.HandleFunc("PUT /api/v1/products/{id}", productHandler.Update)
	mux.HandleFunc("DELETE /api/v1/products/{id}", productHandler.Delete)
	mux.HandleFunc("POST /api/v1/products/import", productImportHandler.Import)

	mux.HandleFunc("POST /api/v1/purchases", purchaseHandler.Create)
	mux.HandleFunc("GET /api/v1/purchases", purchaseHandler.GetAll)
	mux.HandleFunc("GET /api/v1/purchases/{id}", purchaseHandler.GetByID)

	log.Printf(
		"server listening on %s",
		serverAddress,
	)

	if err := http.ListenAndServe(
		serverAddress,
		mux,
	); err != nil {
		log.Fatal(err)
	}
}
