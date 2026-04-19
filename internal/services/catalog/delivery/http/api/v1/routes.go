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
	catalog := router.Group("/catalog")

	catalog.Get("/categories", b.handler.GetCategories)
	catalog.Post("/categories", b.auth.Require(roles.Admin), b.handler.CreateCategory)
	catalog.Delete("/categories/:category_id", b.auth.Require(roles.Admin), b.handler.DeleteCategory)
	catalog.Get("/products", b.handler.ListProducts)
	catalog.Get("/products/:product_id", b.handler.GetProduct)
	catalog.Post("/products", b.auth.Require(roles.Client), b.handler.CreateProduct)
	catalog.Put("/products/:product_id", b.auth.Require(roles.Client), b.handler.UpdateProduct)
	catalog.Patch("/products/:product_id", b.auth.Require(roles.Client), b.handler.UpdateProduct)

	adminCatalog := router.Group("/admin/catalog", b.auth.Require(roles.Admin))
	adminCatalog.Post("/categories", b.handler.CreateCategory)
	adminCatalog.Delete("/categories/:category_id", b.handler.DeleteCategory)
}
