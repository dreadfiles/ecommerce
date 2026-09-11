package router

import (
	nethttp "net/http"

	importerhandler "ecommerce/internal/importer/handler"
	producthandler "ecommerce/internal/product/handler"
	purchasehandler "ecommerce/internal/purchase/handler"
)

type Dependencies struct {
	ProductHandler  *producthandler.ProductHandler
	ImportHandler   *importerhandler.ProductImportHandler
	PurchaseHandler *purchasehandler.PurchaseHandler
}

func New(dependencies Dependencies) nethttp.Handler {
	mux := nethttp.NewServeMux()

	mux.HandleFunc(
		"POST /api/v1/products",
		dependencies.ProductHandler.Create,
	)

	mux.HandleFunc(
		"GET /api/v1/products",
		dependencies.ProductHandler.GetAll,
	)

	mux.HandleFunc(
		"GET /api/v1/products/search",
		dependencies.ProductHandler.Search,
	)

	mux.HandleFunc(
		"GET /api/v1/products/{id}",
		dependencies.ProductHandler.GetByID,
	)

	mux.HandleFunc(
		"PUT /api/v1/products/{id}",
		dependencies.ProductHandler.Update,
	)

	mux.HandleFunc(
		"DELETE /api/v1/products/{id}",
		dependencies.ProductHandler.Delete,
	)

	mux.HandleFunc(
		"POST /api/v1/products/import",
		dependencies.ImportHandler.Import,
	)

	mux.HandleFunc(
		"POST /api/v1/purchases",
		dependencies.PurchaseHandler.Create,
	)

	mux.HandleFunc(
		"GET /api/v1/purchases",
		dependencies.PurchaseHandler.GetAll,
	)

	mux.HandleFunc(
		"GET /api/v1/purchases/{id}",
		dependencies.PurchaseHandler.GetByID,
	)

	return mux
}
