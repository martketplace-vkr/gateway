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
	analytics := router.Group("/vendor/analytics", b.auth.Require(roles.Vendor))

	analytics.Get("/overview", b.handler.GetOverview)
	analytics.Get("/niches", b.handler.GetNiches)
	analytics.Get("/products", b.handler.GetProducts)
	analytics.Put("/products/:product_id/cost", b.handler.UpsertProductCost)
	analytics.Get("/reports/sales", b.handler.ExportSalesReport)
}
