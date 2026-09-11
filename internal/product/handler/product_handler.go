package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"ecommerce/internal/product/domain"
	"ecommerce/internal/product/dto"
	"ecommerce/internal/product/repository"
	"ecommerce/internal/product/service"
	httptransport "ecommerce/internal/transport/http"
)

type ProductHandler struct {
	productService service.ProductService
}

func NewProductHandler(
	productService service.ProductService,
) *ProductHandler {
	return &ProductHandler{
		productService: productService,
	}
}

func (h *ProductHandler) Create(
	w http.ResponseWriter,
	r *http.Request,
) {
	var request dto.CreateProductRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		httptransport.WriteError(
			w,
			http.StatusBadRequest,
			"invalid request body",
		)
		return
	}

	product := &domain.Product{
		Name:        request.Name,
		SKU:         request.SKU,
		Description: request.Description,
		Category:    request.Category,
		Price:       request.Price,
		Stock:       request.Stock,
		WeightKg:    request.WeightKg,
	}

	if err := h.productService.Create(
		r.Context(),
		product,
	); err != nil {
		switch {
		case errors.Is(err, repository.ErrConflict):
			httptransport.WriteError(
				w,
				http.StatusConflict,
				err.Error(),
			)

		default:
			httptransport.WriteError(
				w,
				http.StatusBadRequest,
				err.Error(),
			)
		}

		return
	}

	httptransport.WriteJSON(
		w,
		http.StatusCreated,
		toProductResponse(product),
	)
}

func (h *ProductHandler) GetAll(
	w http.ResponseWriter,
	r *http.Request,
) {
	products, err := h.productService.GetAll(
		r.Context(),
	)
	if err != nil {
		httptransport.WriteError(
			w,
			http.StatusInternalServerError,
			"internal server error",
		)
		return
	}

	response := make(
		[]dto.ProductResponse,
		0,
		len(products),
	)

	for _, product := range products {
		response = append(
			response,
			toProductResponse(product),
		)
	}

	httptransport.WriteJSON(
		w,
		http.StatusOK,
		response,
	)
}

func (h *ProductHandler) GetByID(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := strconv.ParseInt(
		r.PathValue("id"),
		10,
		64,
	)
	if err != nil || id <= 0 {
		httptransport.WriteError(
			w,
			http.StatusBadRequest,
			"invalid product id",
		)
		return
	}

	product, err := h.productService.GetByID(
		r.Context(),
		id,
	)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			httptransport.WriteError(
				w,
				http.StatusNotFound,
				err.Error(),
			)

		default:
			httptransport.WriteError(
				w,
				http.StatusInternalServerError,
				"internal server error",
			)
		}

		return
	}

	httptransport.WriteJSON(
		w,
		http.StatusOK,
		toProductResponse(product),
	)
}

func (h *ProductHandler) Update(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := strconv.ParseInt(
		r.PathValue("id"),
		10,
		64,
	)
	if err != nil || id <= 0 {
		httptransport.WriteError(
			w,
			http.StatusBadRequest,
			"invalid product id",
		)
		return
	}

	var request dto.CreateProductRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		httptransport.WriteError(
			w,
			http.StatusBadRequest,
			"invalid request body",
		)
		return
	}

	product := &domain.Product{
		ID:          id,
		Name:        request.Name,
		SKU:         request.SKU,
		Description: request.Description,
		Category:    request.Category,
		Price:       request.Price,
		Stock:       request.Stock,
		WeightKg:    request.WeightKg,
	}

	if err := h.productService.Update(
		r.Context(),
		product,
	); err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			httptransport.WriteError(
				w,
				http.StatusNotFound,
				err.Error(),
			)

		case errors.Is(err, repository.ErrConflict):
			httptransport.WriteError(
				w,
				http.StatusConflict,
				err.Error(),
			)

		default:
			httptransport.WriteError(
				w,
				http.StatusBadRequest,
				err.Error(),
			)
		}

		return
	}

	httptransport.WriteJSON(
		w,
		http.StatusOK,
		toProductResponse(product),
	)
}

func (h *ProductHandler) Delete(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := strconv.ParseInt(
		r.PathValue("id"),
		10,
		64,
	)
	if err != nil || id <= 0 {
		httptransport.WriteError(
			w,
			http.StatusBadRequest,
			"invalid product id",
		)
		return
	}

	if err := h.productService.Delete(
		r.Context(),
		id,
	); err != nil {
		switch {
		case errors.Is(err, repository.ErrNotFound):
			httptransport.WriteError(
				w,
				http.StatusNotFound,
				err.Error(),
			)

		default:
			httptransport.WriteError(
				w,
				http.StatusInternalServerError,
				"internal server error",
			)
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ProductHandler) Search(
	w http.ResponseWriter,
	r *http.Request,
) {
	query := r.URL.Query().Get("q")

	products, err := h.productService.Search(
		r.Context(),
		query,
	)
	if err != nil {
		switch {
		case errors.Is(
			err,
			service.ErrInvalidSearchQuery,
		):
			httptransport.WriteError(
				w,
				http.StatusBadRequest,
				err.Error(),
			)

		default:
			httptransport.WriteError(
				w,
				http.StatusInternalServerError,
				"internal server error",
			)
		}

		return
	}

	response := make(
		[]dto.ProductResponse,
		0,
		len(products),
	)

	for _, product := range products {
		response = append(
			response,
			toProductResponse(product),
		)
	}

	httptransport.WriteJSON(
		w,
		http.StatusOK,
		response,
	)
}

func toProductResponse(
	product *domain.Product,
) dto.ProductResponse {
	return dto.ProductResponse{
		ID:          product.ID,
		Name:        product.Name,
		SKU:         product.SKU,
		Description: product.Description,
		Category:    product.Category,
		Price:       product.Price,
		Stock:       product.Stock,
		WeightKg:    product.WeightKg,
	}
}
