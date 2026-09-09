package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	purchasedomain "ecommerce/internal/purchase/domain"
	"ecommerce/internal/purchase/dto"
	"ecommerce/internal/purchase/repository"
	"ecommerce/internal/purchase/service"
)

const (
	idempotencyKeyHeader = "Idempotency-Key"
	fakePaymentHeader    = "X-Fake-Payment"
)

type PurchaseHandler struct {
	service service.PurchaseService
}

func NewPurchaseHandler(
	service service.PurchaseService,
) *PurchaseHandler {
	return &PurchaseHandler{
		service: service,
	}
}

func (h *PurchaseHandler) Create(
	w http.ResponseWriter,
	r *http.Request,
) {
	var request dto.CreatePurchaseRequest

	if err := json.NewDecoder(
		r.Body,
	).Decode(&request); err != nil {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	idempotencyKey := r.Header.Get(
		idempotencyKeyHeader,
	)

	items := make(
		[]service.PurchaseItem,
		0,
		len(request.Items),
	)

	for _, item := range request.Items {
		items = append(
			items,
			service.PurchaseItem{
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
			},
		)
	}

	fakePaymentValue := r.Header.Get(
		fakePaymentHeader,
	)

	shouldFail := fakePaymentValue == "failure"

	paymentService := service.NewFakePaymentService(
		shouldFail,
	)

	order, err := h.service.Create(
		r.Context(),
		items,
		idempotencyKey,
		paymentService,
	)
	if err != nil {
		h.writeError(w, err)
		return
	}

	response := toPurchaseResponse(order)

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(
		response,
	); err != nil {
		return
	}
}

func (h *PurchaseHandler) GetAll(
	w http.ResponseWriter,
	r *http.Request,
) {
	orders, err := h.service.GetAll(
		r.Context(),
	)
	if err != nil {
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	response := make(
		[]dto.PurchaseResponse,
		0,
		len(orders),
	)

	for _, order := range orders {
		response = append(
			response,
			toPurchaseResponse(order),
		)
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(
		response,
	); err != nil {
		return
	}
}

func (h *PurchaseHandler) GetByID(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := strconv.ParseInt(
		r.PathValue("id"),
		10,
		64,
	)
	if err != nil || id <= 0 {
		http.Error(
			w,
			"invalid purchase id",
			http.StatusBadRequest,
		)
		return
	}

	order, err := h.service.GetByID(
		r.Context(),
		id,
	)
	if err != nil {
		switch {
		case errors.Is(
			err,
			repository.ErrOrderNotFound,
		):
			http.Error(
				w,
				"purchase not found",
				http.StatusNotFound,
			)

		case errors.Is(
			err,
			service.ErrInvalidPurchase,
		):
			http.Error(
				w,
				err.Error(),
				http.StatusBadRequest,
			)

		default:
			http.Error(
				w,
				"internal server error",
				http.StatusInternalServerError,
			)
		}

		return
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(
		toPurchaseResponse(order),
	); err != nil {
		return
	}
}

func (h *PurchaseHandler) writeError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		service.ErrInvalidPurchase,
	):
		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)

	case errors.Is(
		err,
		service.ErrInvalidProductID,
	):
		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)

	case errors.Is(
		err,
		service.ErrInvalidQuantity,
	):
		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)

	case errors.Is(
		err,
		service.ErrIdempotencyKeyRequired,
	):
		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)

	case errors.Is(
		err,
		service.ErrProductNotFound,
	):
		http.Error(
			w,
			err.Error(),
			http.StatusNotFound,
		)

	case errors.Is(
		err,
		service.ErrInsufficientStock,
	):
		http.Error(
			w,
			err.Error(),
			http.StatusConflict,
		)

	case errors.Is(
		err,
		service.ErrIdempotencyKeyConflict,
	):
		http.Error(
			w,
			err.Error(),
			http.StatusConflict,
		)

	case errors.Is(
		err,
		service.ErrPaymentDeclined,
	):
		http.Error(
			w,
			err.Error(),
			http.StatusPaymentRequired,
		)

	default:
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
	}
}

func toPurchaseResponse(
	order *purchasedomain.Order,
) dto.PurchaseResponse {
	items := make(
		[]dto.PurchaseItemResponse,
		0,
		len(order.Items),
	)

	for _, item := range order.Items {
		items = append(
			items,
			dto.PurchaseItemResponse{
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
				UnitPrice: item.UnitPrice,
				Subtotal:  item.Subtotal,
			},
		)
	}

	return dto.PurchaseResponse{
		ID:     order.ID,
		Status: string(order.Status),
		Total:  order.Total,
		Items:  items,
	}
}
