package v1

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	cartclient "github.com/martketplace-vkr/cart/pkg/api/grpc/v1/client"
	catalogclient "github.com/martketplace-vkr/catalog/pkg/api/grpc/v1/client"
	"github.com/martketplace-vkr/gateway/internal/common/httpx"
	"github.com/martketplace-vkr/gateway/internal/common/middleware"
	"github.com/martketplace-vkr/gateway/internal/services/order/models"
	orderclient "github.com/martketplace-vkr/order/pkg/api/grpc/v1/client"
	ordervendor "github.com/martketplace-vkr/order/pkg/api/grpc/v1/vendor"
	"google.golang.org/grpc/codes"
)

type Handler struct {
	orderClient       orderclient.OrderClientServiceClient
	orderVendorClient ordervendor.OrderVendorServiceClient
	cartClient        cartclient.CartClientServiceClient
	catalogClient     catalogclient.CatalogClientServiceClient
	timeout           time.Duration
}

func New(
	orderClient orderclient.OrderClientServiceClient,
	orderVendorClient ordervendor.OrderVendorServiceClient,
	cartClient cartclient.CartClientServiceClient,
	catalogClient catalogclient.CatalogClientServiceClient,
	timeout time.Duration,
) *Handler {
	return &Handler{
		orderClient:       orderClient,
		orderVendorClient: orderVendorClient,
		cartClient:        cartClient,
		catalogClient:     catalogClient,
		timeout:           timeout,
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

	if err := h.syncCheckoutCart(c, req.ProductIDs); err != nil {
		return err
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

func (h *Handler) syncCheckoutCart(c *fiber.Ctx, productIDs []int64) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	quantities := make(map[int64]uint32, len(productIDs))
	for _, productID := range productIDs {
		quantities[productID]++
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	for productID, quantity := range quantities {
		_, err = h.cartClient.RemoveItem(ctx, &cartclient.RemoveItemRequest{
			UserId:    user.ID,
			ProductId: productID,
		})
		if err != nil && !httpx.IsGRPCCode(err, codes.NotFound) {
			return httpx.MapGRPCError(err)
		}

		productResp, err := h.catalogClient.GetProduct(ctx, &catalogclient.GetProductRequest{
			ProductId: productID,
		})
		if err != nil {
			return httpx.MapGRPCError(err)
		}
		if productResp.GetProduct() == nil || productResp.GetProduct().GetVendorId() <= 0 {
			return fiber.NewError(fiber.StatusPreconditionFailed, "product vendor_id is required")
		}

		_, err = h.cartClient.AddItem(ctx, &cartclient.AddItemRequest{
			UserId:    user.ID,
			ProductId: productID,
			VendorId:  productResp.GetProduct().GetVendorId(),
			Quantity:  quantity,
		})
		if err != nil {
			return httpx.MapGRPCError(err)
		}
	}

	return nil
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

func (h *Handler) GetVendorOrderList(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.orderVendorClient.GetOrderList(ctx, &ordervendor.GetOrderListRequest{
		VendorId: user.ID,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) UpdateVendorOrder(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	orderID, err := strconv.ParseInt(c.Params("order_id"), 10, 64)
	if err != nil || orderID <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid order_id")
	}

	req := models.UpdateVendorOrderRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if strings.TrimSpace(req.Status) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "status is required")
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.orderVendorClient.UpdateOrder(ctx, &ordervendor.UpdateOrderRequest{
		VendorId: user.ID,
		OrderId:  orderID,
		Status:   strings.TrimSpace(req.Status),
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}
