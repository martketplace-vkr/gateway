package models

type CreateTopUpRequest struct {
	Amount         string `json:"amount"`
	CurrencyCode   int64  `json:"currency_code"`
	ProviderType   int32  `json:"provider_type"`
	ProviderName   string `json:"provider_name"`
	IdempotencyKey string `json:"idempotency_key"`
	SuccessURL     string `json:"success_url"`
	FailURL        string `json:"fail_url"`
	Network        string `json:"network"`
}

type CreateWithdrawalRequest struct {
	Amount          string `json:"amount"`
	CurrencyCode    int64  `json:"currency_code"`
	DestinationType int32  `json:"destination_type"`
	Destination     string `json:"destination"`
	IdempotencyKey  string `json:"idempotency_key"`
}
