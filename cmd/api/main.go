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
	"ecommerce/internal/router"
)

const serverAddress = ":8080"

func main() {
	db, err := database.NewPostgresDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

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

	log.Printf(
		"server listening on %s",
		serverAddress,
	)

	if err := http.ListenAndServe(
		serverAddress,
		handler,
	); err != nil {
		log.Fatal(err)
	}
}
