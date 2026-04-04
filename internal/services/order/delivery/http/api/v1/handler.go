package v1

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/martketplace-vkr/gateway/internal/common/httpx"
	"github.com/martketplace-vkr/gateway/internal/common/middleware"
	"github.com/martketplace-vkr/gateway/internal/services/order/models"
	orderclient "github.com/martketplace-vkr/order/pkg/api/grpc/v1/client"
)

type Handler struct {
	orderClient orderclient.OrderClientServiceClient
	timeout     time.Duration
}

func New(orderClient orderclient.OrderClientServiceClient, timeout time.Duration) *Handler {
	return &Handler{
		orderClient: orderClient,
		timeout:     timeout,
	}
}

func (h *Handler) Checkout(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	req := models.CheckoutRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if strings.TrimSpace(req.CheckoutID) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "checkout_id is required")
	}
	if len(req.ProductIDs) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "product_ids are required")
	}
	for _, productID := range req.ProductIDs {
		if productID <= 0 {
			return fiber.NewError(fiber.StatusBadRequest, "product_ids must be greater than zero")
		}
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.orderClient.Checkout(ctx, &orderclient.CheckoutRequest{
		UserId:              user.ID,
		CheckoutId:          strings.TrimSpace(req.CheckoutID),
		ProductIds:          req.ProductIDs,
		ExpectedCartVersion: req.ExpectedCartVersion,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) GetOrderList(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.orderClient.GetOrderList(ctx, &orderclient.GetOrderListRequest{
		UserId: user.ID,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) GetOrder(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	orderID, err := strconv.ParseInt(c.Params("order_id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid order_id")
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.orderClient.GetOrder(ctx, &orderclient.GetOrderRequest{
		UserId:  user.ID,
		OrderId: orderID,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) CancelOrder(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	orderID, err := strconv.ParseInt(c.Params("order_id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid order_id")
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.orderClient.CancellOrder(ctx, &orderclient.CancelOrderRequest{
		UserId:  user.ID,
		OrderId: orderID,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}
