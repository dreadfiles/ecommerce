package domain

type OrderStatus string

const (
	OrderStatusPending OrderStatus = "PENDING"
	OrderStatusPaid    OrderStatus = "PAID"
	OrderStatusFailed  OrderStatus = "FAILED"
)

type Order struct {
	ID             int64
	Status         OrderStatus
	Total          string
	IdempotencyKey string
	Items          []OrderItem
}

type OrderItem struct {
	ID        int64
	OrderID   int64
	ProductID int64
	Quantity  int
	UnitPrice string
	Subtotal  string
}
