package dto

type CreatePurchaseRequest struct {
	Items []PurchaseItemRequest `json:"items"`
}

type PurchaseItemRequest struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}

type PurchaseResponse struct {
	ID     int64                  `json:"id"`
	Status string                 `json:"status"`
	Total  string                 `json:"total"`
	Items  []PurchaseItemResponse `json:"items"`
}

type PurchaseItemResponse struct {
	ProductID int64  `json:"product_id"`
	Quantity  int    `json:"quantity"`
	UnitPrice string `json:"unit_price"`
	Subtotal  string `json:"subtotal"`
}
