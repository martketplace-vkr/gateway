package models

type CheckoutRequest struct {
	CheckoutID          string  `json:"checkout_id"`
	ProductIDs          []int64 `json:"product_ids"`
	ExpectedCartVersion uint64  `json:"expected_cart_version"`
}
