package models

type UpsertProductCostRequest struct {
	CostPrice string `json:"cost_price"`
}

type RecordProductViewRequest struct {
	VisitorID string `json:"visitor_id"`
}

type TariffRequest struct {
	Name              string `json:"name"`
	CommissionPercent string `json:"commission_percent"`
}

type AssignVendorTariffRequest struct {
	TariffID int64 `json:"tariff_id"`
}
