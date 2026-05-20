package models

type CheckoutRequest struct {
	CheckoutID          string  `json:"checkout_id"`
	ProductIDs          []int64 `json:"product_ids"`
	ExpectedCartVersion uint64  `json:"expected_cart_version"`
	DeliveryAddressID   int64   `json:"delivery_address_id"`
}

type UpdateVendorOrderRequest struct {
	Status            string `json:"status"`
	FulfillmentStatus string `json:"fulfillment_status"`
}

type UpdatePaymentStatusRequest struct {
	PaymentStatus string `json:"payment_status"`
}
