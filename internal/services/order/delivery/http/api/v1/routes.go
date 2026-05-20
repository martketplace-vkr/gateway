package v1

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/martketplace-vkr/gateway/internal/common/middleware"
	"github.com/martketplace-vkr/gateway/pkg/roles"
	"github.com/martketplace-vkr/pkg/server/http"
)

type Binder struct {
	server  *http.Server
	auth    *middleware.Auth
	handler *Handler
}

func NewBinder(server *http.Server, auth *middleware.Auth, handler *Handler) *Binder {
	return &Binder{
		server:  server,
		auth:    auth,
		handler: handler,
	}
}

func (b *Binder) BindRoutes(_ context.Context) {
	for _, prefix := range []string{"/api/v1", "/client/v1"} {
		b.bind(b.server.Group(prefix))
	}
}

func (b *Binder) bind(router fiber.Router) {
	orders := router.Group("/orders", b.auth.Require(roles.Client))

	orders.Post("/checkout", b.handler.Checkout)
	orders.Get("", b.handler.GetOrderList)
	orders.Get("/", b.handler.GetOrderList)
	orders.Get("/:order_id", b.handler.GetOrder)
	orders.Post("/:order_id/cancel", b.handler.CancelOrder)

	vendorOrders := router.Group("/vendor/orders", b.auth.Require(roles.Vendor))
	vendorOrders.Get("", b.handler.GetVendorOrderList)
	vendorOrders.Get("/", b.handler.GetVendorOrderList)
	vendorOrders.Patch("/:order_id", b.handler.UpdateVendorOrder)

	adminOrders := router.Group("/admin/orders", b.auth.Require(roles.Admin))
	adminOrders.Get("", b.handler.GetAdminOrderList)
	adminOrders.Get("/", b.handler.GetAdminOrderList)
	adminOrders.Patch("/:order_id/payment-status", b.handler.UpdateAdminOrderPaymentStatus)
}
