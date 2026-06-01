package v1

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	catalogadmin "github.com/martketplace-vkr/catalog/pkg/api/grpc/v1/admin"
	catalogclient "github.com/martketplace-vkr/catalog/pkg/api/grpc/v1/client"
	catalogdomain "github.com/martketplace-vkr/catalog/pkg/api/grpc/v1/domain"
	catalogvendor "github.com/martketplace-vkr/catalog/pkg/api/grpc/v1/vendor"
	"github.com/martketplace-vkr/gateway/internal/common/httpx"
	"github.com/martketplace-vkr/gateway/internal/common/middleware"
	"github.com/martketplace-vkr/gateway/internal/services/catalog/models"
)

type Handler struct {
	catalogClient       catalogclient.CatalogClientServiceClient
	catalogAdminClient  catalogadmin.CatalogAdminServiceClient
	catalogVendorClient catalogvendor.CatalogVendorServiceClient
	timeout             time.Duration
}

func New(
	catalogClient catalogclient.CatalogClientServiceClient,
	catalogAdminClient catalogadmin.CatalogAdminServiceClient,
	catalogVendorClient catalogvendor.CatalogVendorServiceClient,
	timeout time.Duration,
) *Handler {
	return &Handler{
		catalogClient:       catalogClient,
		catalogAdminClient:  catalogAdminClient,
		catalogVendorClient: catalogVendorClient,
		timeout:             timeout,
	}
}

func (h *Handler) GetCategories(c *fiber.Ctx) error {
	parentID, hasParentID, err := parseOptionalInt64Query(c, "parent_id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid parent_id")
	}

	filterByParent, err := parseBoolQuery(c, "filter_by_parent", hasParentID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid filter_by_parent")
	}

	includeChildren, err := parseBoolQuery(c, "include_children", false)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid include_children")
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.catalogClient.GetCategories(ctx, &catalogclient.GetCategoriesRequest{
		ParentId:        parentID,
		FilterByParent:  filterByParent,
		IncludeChildren: includeChildren,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) CreateCategory(c *fiber.Ctx) error {
	req := models.CreateCategoryRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}

	if req.ParentID != nil && *req.ParentID <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "parent_id must be greater than zero")
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.catalogAdminClient.CreateCategory(ctx, &catalogadmin.CreateCategoryRequest{
		Name:     name,
		ParentId: req.ParentID,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) DeleteCategory(c *fiber.Ctx) error {
	categoryID, err := strconv.ParseInt(c.Params("category_id"), 10, 64)
	if err != nil || categoryID <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid category_id")
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.catalogAdminClient.DeleteCategory(ctx, &catalogadmin.DeleteCategoryRequest{
		CategoryId: categoryID,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) ListProducts(c *fiber.Ctx) error {
	categoryID, _, err := parseOptionalInt64Query(c, "category_id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid category_id")
	}

	vendorID, _, err := parseOptionalInt64Query(c, "vendor_id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid vendor_id")
	}

	pageSize, err := parseOptionalUint32Query(c, "page_size", 0)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid page_size")
	}

	pageToken, err := parseOptionalUint64Query(c, "page_token", 0)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid page_token")
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.catalogClient.ListProducts(ctx, &catalogclient.ListProductsRequest{
		CategoryId: categoryID,
		VendorId:   vendorID,
		PageSize:   pageSize,
		PageToken:  pageToken,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return c.JSON(models.ListProductsResponse{
		Products:      mapProducts(resp.GetProducts()),
		NextPageToken: resp.GetNextPageToken(),
	})
}

func (h *Handler) GetProduct(c *fiber.Ctx) error {
	productID, err := strconv.ParseInt(c.Params("product_id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid product_id")
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.catalogClient.GetProduct(ctx, &catalogclient.GetProductRequest{
		ProductId: productID,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return c.JSON(models.GetProductResponse{
		Product: mapProduct(resp.GetProduct()),
	})
}

func (h *Handler) CreateProduct(c *fiber.Ctx) error {
	req := models.CreateProductRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.catalogVendorClient.CreateProduct(ctx, &catalogvendor.CreateProductRequest{
		VendorId:          user.ID,
		CategoryId:        req.CategoryID,
		Name:              req.Name,
		Description:       req.Description,
		Price:             req.Price,
		AcceptsCrypto:     req.AcceptsCrypto,
		CryptoPricingMode: req.CryptoPricingMode,
		CryptoPriceUsdt:   req.CryptoPriceUSDT,
		StockCount:        req.StockCount,
		Characteristics:   mapProductSections(req.Attributes),
		Images:            mapProductImages(req.Images),
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return c.JSON(models.GetProductResponse{
		Product: mapProduct(resp.GetProduct()),
	})
}

func (h *Handler) UpdateProduct(c *fiber.Ctx) error {
	req := models.UpdateProductRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	productID, err := strconv.ParseInt(c.Params("product_id"), 10, 64)
	if err != nil || productID <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid product_id")
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.catalogVendorClient.UpdateProduct(ctx, &catalogvendor.UpdateProductRequest{
		ProductId:         productID,
		VendorId:          user.ID,
		CategoryId:        req.CategoryID,
		Name:              req.Name,
		Description:       req.Description,
		Price:             req.Price,
		AcceptsCrypto:     req.AcceptsCrypto,
		CryptoPricingMode: req.CryptoPricingMode,
		CryptoPriceUsdt:   req.CryptoPriceUSDT,
		StockCount:        req.StockCount,
		Characteristics:   mapProductSections(req.Attributes),
		Images:            mapProductImages(req.Images),
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return c.JSON(models.GetProductResponse{
		Product: mapProduct(resp.GetProduct()),
	})
}

func (h *Handler) GetUSDTExchangeRate(c *fiber.Ctx) error {
	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.catalogAdminClient.GetUSDTExchangeRate(ctx, &catalogadmin.GetUSDTExchangeRateRequest{})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) UpdateUSDTExchangeRate(c *fiber.Ctx) error {
	req := models.UpdateUSDTExchangeRateRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.catalogAdminClient.UpdateUSDTExchangeRate(ctx, &catalogadmin.UpdateUSDTExchangeRateRequest{
		RubPerUsdt: strings.TrimSpace(req.RubPerUSDT),
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) GetVendorProducts(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.catalogVendorClient.GetVendorProduct(ctx, &catalogvendor.GetVendorProductRequest{
		VendorID: user.ID,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return c.JSON(models.ListProductsResponse{
		Products: mapProducts(resp.GetProducts()),
	})
}

func mapProductSections(sections []models.ProductAttributeSectionInput) []*catalogdomain.ProductCharacteristicInput {
	result := make([]*catalogdomain.ProductCharacteristicInput, 0, len(sections))
	for _, section := range sections {
		result = append(result, &catalogdomain.ProductCharacteristicInput{
			Title:      strings.TrimSpace(section.Title),
			Attributes: mapSectionAttributes(section.Attributes),
		})
	}

	return result
}

func mapSectionAttributes(attributes []models.ProductAttributeInput) []*catalogdomain.ProductAttributeInput {
	result := make([]*catalogdomain.ProductAttributeInput, 0, len(attributes))
	for _, attribute := range attributes {
		result = append(result, &catalogdomain.ProductAttributeInput{
			Name:  strings.TrimSpace(attribute.Key),
			Value: strings.TrimSpace(attribute.Value),
		})
	}

	return result
}

func mapProductImages(images []models.ProductImageInput) []*catalogdomain.ProductImageInput {
	result := make([]*catalogdomain.ProductImageInput, 0, len(images))
	for _, image := range images {
		result = append(result, &catalogdomain.ProductImageInput{
			Url:    image.URL,
			IsMain: image.IsMain,
		})
	}

	return result
}

func mapProducts(products []*catalogdomain.Product) []models.ProductResponse {
	result := make([]models.ProductResponse, 0, len(products))
	for _, product := range products {
		if mapped := mapProduct(product); mapped != nil {
			result = append(result, *mapped)
		}
	}

	return result
}

func mapProduct(product *catalogdomain.Product) *models.ProductResponse {
	if product == nil {
		return nil
	}

	return &models.ProductResponse{
		ID:                 strconv.FormatInt(product.GetId(), 10),
		VendorID:           strconv.FormatInt(product.GetVendorId(), 10),
		CategoryID:         strconv.FormatInt(product.GetCategoryId(), 10),
		Name:               product.GetName(),
		Description:        product.GetDescription(),
		Price:              product.GetPrice(),
		AcceptsCrypto:      product.GetAcceptsCrypto(),
		CryptoPricingMode:  product.GetCryptoPricingMode(),
		CryptoPriceUSDT:    product.GetCryptoPriceUsdt(),
		EffectiveUSDTPrice: product.GetEffectiveUsdtPrice(),
		RubPerUSDT:         product.GetRubPerUsdt(),
		StockCount:         product.GetStockCount(),
		Attributes:         mapProductAttributeSections(product.GetCharacteristics()),
		Images:             mapProductImageResponses(product.GetImages()),
		CreatedAt:          mapTimestamp(product.GetCreatedAt()),
		UpdatedAt:          mapTimestamp(product.GetUpdatedAt()),
	}
}

func mapProductAttributeSections(sections []*catalogdomain.ProductCharacteristic) []models.ProductAttributeSectionResponse {
	result := make([]models.ProductAttributeSectionResponse, 0, len(sections))
	for _, section := range sections {
		if section == nil {
			continue
		}

		result = append(result, models.ProductAttributeSectionResponse{
			Title:      section.GetTitle(),
			Attributes: mapSectionAttributeResponses(section.GetAttributes()),
		})
	}

	return result
}

func mapSectionAttributeResponses(attributes []*catalogdomain.ProductAttribute) []models.ProductAttributeResponse {
	result := make([]models.ProductAttributeResponse, 0, len(attributes))
	for _, attribute := range attributes {
		if attribute == nil {
			continue
		}

		result = append(result, models.ProductAttributeResponse{
			Key:   attribute.GetName(),
			Value: attribute.GetValue(),
		})
	}

	return result
}

func mapProductImageResponses(images []*catalogdomain.ProductImage) []models.ProductImageResponse {
	result := make([]models.ProductImageResponse, 0, len(images))
	for _, image := range images {
		if image == nil {
			continue
		}

		result = append(result, models.ProductImageResponse{
			ID:     strconv.FormatInt(image.GetId(), 10),
			URL:    image.GetUrl(),
			IsMain: image.GetIsMain(),
		})
	}

	return result
}

func mapTimestamp(ts interface{ AsTime() time.Time }) string {
	if ts == nil {
		return ""
	}

	return ts.AsTime().UTC().Format(time.RFC3339)
}

func parseOptionalInt64Query(c *fiber.Ctx, key string) (int64, bool, error) {
	raw := c.Query(key)
	if raw == "" {
		return 0, false, nil
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, true, err
	}

	return value, true, nil
}

func parseOptionalUint32Query(c *fiber.Ctx, key string, defaultValue uint32) (uint32, error) {
	raw := c.Query(key)
	if raw == "" {
		return defaultValue, nil
	}

	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, err
	}

	return uint32(value), nil
}

func parseOptionalUint64Query(c *fiber.Ctx, key string, defaultValue uint64) (uint64, error) {
	raw := c.Query(key)
	if raw == "" {
		return defaultValue, nil
	}

	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, err
	}

	return value, nil
}

func parseBoolQuery(c *fiber.Ctx, key string, defaultValue bool) (bool, error) {
	raw := c.Query(key)
	if raw == "" {
		return defaultValue, nil
	}

	return strconv.ParseBool(raw)
}
