package models

type ProductAttributeInput struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ProductImageInput struct {
	URL    string `json:"url"`
	IsMain bool   `json:"is_main"`
}

type CreateProductRequest struct {
	VendorID    int64                   `json:"vendor_id"`
	CategoryID  int64                   `json:"category_id"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Price       string                  `json:"price"`
	StockCount  uint32                  `json:"stock_count"`
	Attributes  []ProductAttributeInput `json:"attributes"`
	Images      []ProductImageInput     `json:"images"`
}

type CreateCategoryRequest struct {
	Name     string `json:"name"`
	ParentID *int64 `json:"parent_id"`
}

type UpdateProductRequest struct {
	VendorID    int64                   `json:"vendor_id"`
	CategoryID  int64                   `json:"category_id"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Price       string                  `json:"price"`
	StockCount  uint32                  `json:"stock_count"`
	Attributes  []ProductAttributeInput `json:"attributes"`
	Images      []ProductImageInput     `json:"images"`
}
