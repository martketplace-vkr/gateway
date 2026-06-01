package v1

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	analyticsadmin "github.com/martketplace-vkr/analytics/pkg/api/grpc/v1/admin"
	analyticsvendor "github.com/martketplace-vkr/analytics/pkg/api/grpc/v1/vendor"
	"github.com/martketplace-vkr/gateway/internal/common/httpx"
	"github.com/martketplace-vkr/gateway/internal/common/middleware"
	"github.com/martketplace-vkr/gateway/internal/services/analytics/models"
)

type Handler struct {
	analyticsAdminClient  analyticsadmin.AnalyticsAdminServiceClient
	analyticsVendorClient analyticsvendor.AnalyticsVendorServiceClient
	timeout               time.Duration
}

func New(analyticsVendorClient analyticsvendor.AnalyticsVendorServiceClient, analyticsAdminClient analyticsadmin.AnalyticsAdminServiceClient, timeout time.Duration) *Handler {
	return &Handler{
		analyticsAdminClient:  analyticsAdminClient,
		analyticsVendorClient: analyticsVendorClient,
		timeout:               timeout,
	}
}

func (h *Handler) GetOverview(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.analyticsVendorClient.GetOverview(ctx, &analyticsvendor.GetOverviewRequest{
		VendorId: user.ID,
		From:     c.Query("from"),
		To:       c.Query("to"),
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) GetNiches(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	limit, err := parseUint32Query(c, "limit")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid limit")
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.analyticsVendorClient.GetNiches(ctx, &analyticsvendor.GetNichesRequest{
		VendorId: user.ID,
		From:     c.Query("from"),
		To:       c.Query("to"),
		Sort:     c.Query("sort", "opportunity"),
		Limit:    limit,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) GetProducts(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.analyticsVendorClient.GetProducts(ctx, &analyticsvendor.GetProductsRequest{
		VendorId: user.ID,
		From:     c.Query("from"),
		To:       c.Query("to"),
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) RecordProductView(c *fiber.Ctx) error {
	productID, err := strconv.ParseInt(c.Params("product_id"), 10, 64)
	if err != nil || productID <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid product_id")
	}

	req := models.RecordProductViewRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	req.VisitorID = strings.TrimSpace(req.VisitorID)
	if req.VisitorID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "visitor_id is required")
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	if _, err := h.analyticsVendorClient.RecordProductView(ctx, &analyticsvendor.RecordProductViewRequest{
		ProductId: productID,
		VisitorId: req.VisitorID,
	}); err != nil {
		return httpx.MapGRPCError(err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) UpsertProductCost(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	productID, err := strconv.ParseInt(c.Params("product_id"), 10, 64)
	if err != nil || productID <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid product_id")
	}

	req := models.UpsertProductCostRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.analyticsVendorClient.UpsertProductCost(ctx, &analyticsvendor.UpsertProductCostRequest{
		VendorId:  user.ID,
		ProductId: productID,
		CostPrice: req.CostPrice,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) ExportSalesReport(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.analyticsVendorClient.ExportSalesReport(ctx, &analyticsvendor.ExportSalesReportRequest{
		VendorId: user.ID,
		From:     c.Query("from"),
		To:       c.Query("to"),
		Format:   c.Query("format", "csv"),
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	filename := resp.GetFilename()
	if filename == "" {
		filename = "sales-report"
	}

	c.Set(fiber.HeaderContentType, resp.GetContentType())
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf(`attachment; filename="%s"`, filename))

	return c.Status(fiber.StatusOK).Send(resp.GetContent())
}

func (h *Handler) ListTariffs(c *fiber.Ctx) error {
	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.analyticsAdminClient.ListTariffs(ctx, &analyticsadmin.ListTariffsRequest{})
	if err != nil {
		return httpx.MapGRPCError(err)
	}
	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) CreateTariff(c *fiber.Ctx) error {
	req := models.TariffRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.analyticsAdminClient.CreateTariff(ctx, &analyticsadmin.CreateTariffRequest{
		Name:              req.Name,
		CommissionPercent: req.CommissionPercent,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}
	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) UpdateTariff(c *fiber.Ctx) error {
	tariffID, err := parsePositiveInt64Param(c, "tariff_id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid tariff_id")
	}

	req := models.TariffRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.analyticsAdminClient.UpdateTariff(ctx, &analyticsadmin.UpdateTariffRequest{
		TariffId:          tariffID,
		Name:              req.Name,
		CommissionPercent: req.CommissionPercent,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}
	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) SetDefaultTariff(c *fiber.Ctx) error {
	tariffID, err := parsePositiveInt64Param(c, "tariff_id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid tariff_id")
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.analyticsAdminClient.SetDefaultTariff(ctx, &analyticsadmin.SetDefaultTariffRequest{
		TariffId: tariffID,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}
	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) AssignVendorTariff(c *fiber.Ctx) error {
	vendorID, err := parsePositiveInt64Param(c, "vendor_id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid vendor_id")
	}

	req := models.AssignVendorTariffRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.analyticsAdminClient.AssignVendorTariff(ctx, &analyticsadmin.AssignVendorTariffRequest{
		VendorId: vendorID,
		TariffId: req.TariffID,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}
	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) GetVendorTariff(c *fiber.Ctx) error {
	vendorID, err := parsePositiveInt64Param(c, "vendor_id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid vendor_id")
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.analyticsAdminClient.GetVendorTariff(ctx, &analyticsadmin.GetVendorTariffRequest{
		VendorId: vendorID,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}
	return httpx.WriteProtoJSON(c, resp)
}

func parseUint32Query(c *fiber.Ctx, key string) (uint32, error) {
	raw := c.Query(key)
	if raw == "" {
		return 0, nil
	}

	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, err
	}

	return uint32(value), nil
}

func parsePositiveInt64Param(c *fiber.Ctx, key string) (int64, error) {
	value, err := strconv.ParseInt(c.Params(key), 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("invalid %s", key)
	}
	return value, nil
}
