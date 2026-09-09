package dto

type CreateProductRequest struct {
	Name        string `json:"name"`
	SKU         string `json:"sku"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Price       string `json:"price"`
	Stock       int    `json:"stock"`
	WeightKg    string `json:"weight_kg"`
}

type ProductResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	SKU         string `json:"sku"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Price       string `json:"price"`
	Stock       int    `json:"stock"`
	WeightKg    string `json:"weight_kg"`
}
