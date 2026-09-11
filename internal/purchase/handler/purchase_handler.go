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
	httptransport "ecommerce/internal/transport/http"
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
		httptransport.WriteError(
			w,
			http.StatusBadRequest,
			"invalid request body",
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

	httptransport.WriteJSON(
		w,
		http.StatusCreated,
		toPurchaseResponse(order),
	)
}

func (h *PurchaseHandler) GetAll(
	w http.ResponseWriter,
	r *http.Request,
) {
	orders, err := h.service.GetAll(
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

	httptransport.WriteJSON(
		w,
		http.StatusOK,
		response,
	)
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
		httptransport.WriteError(
			w,
			http.StatusBadRequest,
			"invalid purchase id",
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
			httptransport.WriteError(
				w,
				http.StatusNotFound,
				"purchase not found",
			)

		case errors.Is(
			err,
			service.ErrInvalidPurchase,
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

	httptransport.WriteJSON(
		w,
		http.StatusOK,
		toPurchaseResponse(order),
	)
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
		httptransport.WriteError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)

	case errors.Is(
		err,
		service.ErrInvalidProductID,
	):
		httptransport.WriteError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)

	case errors.Is(
		err,
		service.ErrInvalidQuantity,
	):
		httptransport.WriteError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)

	case errors.Is(
		err,
		service.ErrIdempotencyKeyRequired,
	):
		httptransport.WriteError(
			w,
			http.StatusBadRequest,
			err.Error(),
		)

	case errors.Is(
		err,
		service.ErrProductNotFound,
	):
		httptransport.WriteError(
			w,
			http.StatusNotFound,
			err.Error(),
		)

	case errors.Is(
		err,
		service.ErrInsufficientStock,
	):
		httptransport.WriteError(
			w,
			http.StatusConflict,
			err.Error(),
		)

	case errors.Is(
		err,
		service.ErrIdempotencyKeyConflict,
	):
		httptransport.WriteError(
			w,
			http.StatusConflict,
			err.Error(),
		)

	case errors.Is(
		err,
		service.ErrPaymentDeclined,
	):
		httptransport.WriteError(
			w,
			http.StatusPaymentRequired,
			err.Error(),
		)

	default:
		httptransport.WriteError(
			w,
			http.StatusInternalServerError,
			"internal server error",
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
