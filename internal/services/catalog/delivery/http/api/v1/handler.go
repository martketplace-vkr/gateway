package v1

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	catalogclient "github.com/martketplace-vkr/catalog/pkg/api/grpc/v1/client"
	"github.com/martketplace-vkr/gateway/internal/common/httpx"
	"github.com/martketplace-vkr/gateway/internal/common/middleware"
	"github.com/martketplace-vkr/gateway/internal/services/catalog/models"
)

type Handler struct {
	catalogClient catalogclient.CatalogClientServiceClient
	timeout       time.Duration
}

func New(catalogClient catalogclient.CatalogClientServiceClient, timeout time.Duration) *Handler {
	return &Handler{
		catalogClient: catalogClient,
		timeout:       timeout,
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

	return httpx.WriteProtoJSON(c, resp)
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

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) CreateProduct(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	req := models.CreateProductRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.catalogClient.CreateProduct(ctx, &catalogclient.CreateProductRequest{
		VendorId:    user.ID,
		CategoryId:  req.CategoryID,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		StockCount:  req.StockCount,
		Attributes:  toProductAttributeInputs(req.Attributes),
		Images:      toProductImageInputs(req.Images),
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) UpdateProduct(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	productID, err := strconv.ParseInt(c.Params("product_id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid product_id")
	}

	req := models.UpdateProductRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.catalogClient.UpdateProduct(ctx, &catalogclient.UpdateProductRequest{
		ProductId:   productID,
		VendorId:    user.ID,
		CategoryId:  req.CategoryID,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		StockCount:  req.StockCount,
		Attributes:  toProductAttributeInputs(req.Attributes),
		Images:      toProductImageInputs(req.Images),
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func toProductAttributeInputs(attributes []models.ProductAttributeInput) []*catalogclient.ProductAttributeInput {
	if len(attributes) == 0 {
		return nil
	}

	result := make([]*catalogclient.ProductAttributeInput, 0, len(attributes))
	for _, attribute := range attributes {
		result = append(result, &catalogclient.ProductAttributeInput{
			Name:  attribute.Name,
			Value: attribute.Value,
		})
	}

	return result
}

func toProductImageInputs(images []models.ProductImageInput) []*catalogclient.ProductImageInput {
	if len(images) == 0 {
		return nil
	}

	result := make([]*catalogclient.ProductImageInput, 0, len(images))
	for _, image := range images {
		result = append(result, &catalogclient.ProductImageInput{
			Url:    image.URL,
			IsMain: image.IsMain,
		})
	}

	return result
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
