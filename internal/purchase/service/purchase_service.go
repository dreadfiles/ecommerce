package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	purchasedomain "ecommerce/internal/purchase/domain"
	"ecommerce/internal/purchase/repository"
)

type PurchaseService interface {
	Create(
		ctx context.Context,
		items []PurchaseItem,
		idempotencyKey string,
	) (*purchasedomain.Order, error)
}

type PurchaseItem struct {
	ProductID int64
	Quantity  int
}

type purchaseService struct {
	purchaseRepository repository.PurchaseRepository
}

func NewPurchaseService(
	purchaseRepository repository.PurchaseRepository,
) PurchaseService {
	return &purchaseService{
		purchaseRepository: purchaseRepository,
	}
}

func (s *purchaseService) Create(
	ctx context.Context,
	items []PurchaseItem,
	idempotencyKey string,
) (*purchasedomain.Order, error) {
	if err := validatePurchase(items, idempotencyKey); err != nil {
		return nil, err
	}

	normalizedKey := strings.TrimSpace(idempotencyKey)

	orderItems := make([]purchasedomain.OrderItem, 0, len(items))

	for _, item := range items {
		orderItems = append(orderItems, purchasedomain.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	order := &purchasedomain.Order{
		Status:         purchasedomain.OrderStatusPending,
		IdempotencyKey: normalizedKey,
		Items:          orderItems,
	}

	if err := s.purchaseRepository.Create(ctx, order); err != nil {
		if errors.Is(err, repository.ErrIdempotencyKeyExists) {
			existingOrder, getErr := s.purchaseRepository.GetByIdempotencyKey(
				ctx,
				normalizedKey,
			)
			if getErr != nil {
				return nil, fmt.Errorf(
					"get existing purchase: %w",
					getErr,
				)
			}

			if !samePurchaseItems(items, existingOrder.Items) {
				return nil, ErrIdempotencyKeyConflict
			}

			return existingOrder, nil
		}

		switch {
		case errors.Is(err, repository.ErrProductNotFound):
			return nil, ErrProductNotFound

		case errors.Is(err, repository.ErrInsufficientStock):
			return nil, ErrInsufficientStock

		default:
			return nil, fmt.Errorf("create purchase: %w", err)
		}
	}

	return order, nil
}

func validatePurchase(
	items []PurchaseItem,
	idempotencyKey string,
) error {
	if len(items) == 0 {
		return ErrInvalidPurchase
	}

	if strings.TrimSpace(idempotencyKey) == "" {
		return ErrIdempotencyKeyRequired
	}

	productIDs := make(map[int64]struct{}, len(items))

	for _, item := range items {
		if item.ProductID <= 0 {
			return ErrInvalidProductID
		}

		if item.Quantity <= 0 {
			return fmt.Errorf(
				"%w: quantity for product %d must be greater than zero",
				ErrInvalidQuantity,
				item.ProductID,
			)
		}

		if _, exists := productIDs[item.ProductID]; exists {
			return fmt.Errorf(
				"%w: product %d appears more than once",
				ErrInvalidPurchase,
				item.ProductID,
			)
		}

		productIDs[item.ProductID] = struct{}{}
	}

	return nil
}

func samePurchaseItems(
	requestItems []PurchaseItem,
	orderItems []purchasedomain.OrderItem,
) bool {
	if len(requestItems) != len(orderItems) {
		return false
	}

	orderItemsByProduct := make(map[int64]int, len(orderItems))

	for _, item := range orderItems {
		orderItemsByProduct[item.ProductID] = item.Quantity
	}

	for _, item := range requestItems {
		quantity, exists := orderItemsByProduct[item.ProductID]
		if !exists || quantity != item.Quantity {
			return false
		}
	}

	return true
}
