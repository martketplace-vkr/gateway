package models

type ProductAttributeInput struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ProductAttributeSectionInput struct {
	Title      string                  `json:"title"`
	Attributes []ProductAttributeInput `json:"attributes"`
}

type ProductImageInput struct {
	URL    string `json:"url"`
	IsMain bool   `json:"is_main"`
}

type ProductAttributeResponse struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ProductAttributeSectionResponse struct {
	Title      string                     `json:"title"`
	Attributes []ProductAttributeResponse `json:"attributes"`
}

type ProductImageResponse struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	IsMain bool   `json:"is_main"`
}

type ProductResponse struct {
	ID          string                            `json:"id"`
	VendorID    string                            `json:"vendor_id"`
	CategoryID  string                            `json:"category_id"`
	Name        string                            `json:"name"`
	Description string                            `json:"description"`
	Price       string                            `json:"price"`
	StockCount  uint32                            `json:"stock_count"`
	Attributes  []ProductAttributeSectionResponse `json:"attributes"`
	Images      []ProductImageResponse            `json:"images"`
	CreatedAt   string                            `json:"created_at,omitempty"`
	UpdatedAt   string                            `json:"updated_at,omitempty"`
}

type ListProductsResponse struct {
	Products      []ProductResponse `json:"products"`
	NextPageToken uint64            `json:"next_page_token,omitempty"`
}

type GetProductResponse struct {
	Product *ProductResponse `json:"product,omitempty"`
}

type CreateProductRequest struct {
	VendorID    int64                          `json:"vendor_id"`
	CategoryID  int64                          `json:"category_id"`
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	Price       string                         `json:"price"`
	StockCount  uint32                         `json:"stock_count"`
	Attributes  []ProductAttributeSectionInput `json:"attributes"`
	Images      []ProductImageInput            `json:"images"`
}

type CreateCategoryRequest struct {
	Name     string `json:"name"`
	ParentID *int64 `json:"parent_id"`
}

type UpdateProductRequest struct {
	VendorID    int64                          `json:"vendor_id"`
	CategoryID  int64                          `json:"category_id"`
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	Price       string                         `json:"price"`
	StockCount  uint32                         `json:"stock_count"`
	Attributes  []ProductAttributeSectionInput `json:"attributes"`
	Images      []ProductImageInput            `json:"images"`
}
