package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	purchasedomain "ecommerce/internal/purchase/domain"
	"ecommerce/internal/purchase/dto"
	"ecommerce/internal/purchase/service"
)

const idempotencyKeyHeader = "Idempotency-Key"

type PurchaseHandler struct {
	service service.PurchaseService
}

func NewPurchaseHandler(service service.PurchaseService) *PurchaseHandler {
	return &PurchaseHandler{
		service: service,
	}
}

func (h *PurchaseHandler) Create(w http.ResponseWriter, r *http.Request) {
	var request dto.CreatePurchaseRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	idempotencyKey := r.Header.Get(idempotencyKeyHeader)

	items := make([]service.PurchaseItem, 0, len(request.Items))

	for _, item := range request.Items {
		items = append(items, service.PurchaseItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	order, err := h.service.Create(
		r.Context(),
		items,
		idempotencyKey,
	)
	if err != nil {
		h.writeError(w, err)
		return
	}

	response := toPurchaseResponse(order)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}

func (h *PurchaseHandler) writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidPurchase):
		http.Error(w, err.Error(), http.StatusBadRequest)

	case errors.Is(err, service.ErrInvalidProductID):
		http.Error(w, err.Error(), http.StatusBadRequest)

	case errors.Is(err, service.ErrInvalidQuantity):
		http.Error(w, err.Error(), http.StatusBadRequest)

	case errors.Is(err, service.ErrIdempotencyKeyRequired):
		http.Error(w, err.Error(), http.StatusBadRequest)

	case errors.Is(err, service.ErrProductNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)

	case errors.Is(err, service.ErrInsufficientStock):
		http.Error(w, err.Error(), http.StatusConflict)

	case errors.Is(err, service.ErrIdempotencyKeyConflict):
		http.Error(w, err.Error(), http.StatusConflict)

	default:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func toPurchaseResponse(order *purchasedomain.Order) dto.PurchaseResponse {
	items := make([]dto.PurchaseItemResponse, 0, len(order.Items))

	for _, item := range order.Items {
		items = append(items, dto.PurchaseItemResponse{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: item.UnitPrice,
			Subtotal:  item.Subtotal,
		})
	}

	return dto.PurchaseResponse{
		ID:     order.ID,
		Status: string(order.Status),
		Total:  order.Total,
		Items:  items,
	}
}
