package models

type AddItemRequest struct {
	ProductID int64  `json:"product_id"`
	VendorID  int64  `json:"vendor_id"`
	Quantity  uint32 `json:"quantity"`
}

type UpdateItemQuantityRequest struct {
	Quantity uint32 `json:"quantity"`
}
