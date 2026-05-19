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
	cart := router.Group("/cart", b.auth.Require(roles.Client))

	cart.Get("", b.handler.GetCart)
	cart.Get("/", b.handler.GetCart)
	cart.Delete("", b.handler.ClearCart)
	cart.Delete("/", b.handler.ClearCart)
	cart.Post("/clear", b.handler.ClearCart)
	cart.Post("/items", b.handler.AddItem)
	cart.Patch("/items/:product_id", b.handler.UpdateItemQuantity)
	cart.Put("/items/:product_id", b.handler.UpdateItemQuantity)
	cart.Delete("/items/:product_id", b.handler.RemoveItem)
}
