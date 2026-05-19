package v1

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/martketplace-vkr/cart/pkg/api/grpc/v1/client"
	"github.com/martketplace-vkr/gateway/internal/common/httpx"
	"github.com/martketplace-vkr/gateway/internal/common/middleware"
	"github.com/martketplace-vkr/gateway/internal/services/cart/models"
)

type Handler struct {
	cartClient client.CartClientServiceClient
	timeout    time.Duration
}

func New(cartClient client.CartClientServiceClient, timeout time.Duration) *Handler {
	return &Handler{
		cartClient: cartClient,
		timeout:    timeout,
	}
}

func (h *Handler) GetCart(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.cartClient.GetCart(ctx, &client.GetCartRequest{
		UserId: user.ID,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) AddItem(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	req := models.AddItemRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if req.ProductID <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "product_id is required")
	}
	if req.VendorID <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "vendor_id is required")
	}
	if req.Quantity == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "quantity must be greater than zero")
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.cartClient.AddItem(ctx, &client.AddItemRequest{
		UserId:    user.ID,
		ProductId: req.ProductID,
		VendorId:  req.VendorID,
		Quantity:  req.Quantity,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) UpdateItemQuantity(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	productID, err := parseProductID(c)
	if err != nil {
		return err
	}

	req := models.UpdateItemQuantityRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if req.Quantity == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "quantity must be greater than zero")
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.cartClient.UpdateItemQuantity(ctx, &client.UpdateItemQuantityRequest{
		UserId:    user.ID,
		ProductId: productID,
		Quantity:  req.Quantity,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) RemoveItem(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	productID, err := parseProductID(c)
	if err != nil {
		return err
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.cartClient.RemoveItem(ctx, &client.RemoveItemRequest{
		UserId:    user.ID,
		ProductId: productID,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) ClearCart(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.cartClient.ClearCart(ctx, &client.ClearCartRequest{
		UserId: user.ID,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func parseProductID(c *fiber.Ctx) (int64, error) {
	productID, err := strconv.ParseInt(c.Params("product_id"), 10, 64)
	if err != nil || productID <= 0 {
		return 0, fiber.NewError(fiber.StatusBadRequest, "invalid product_id")
	}

	return productID, nil
}
