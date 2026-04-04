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
	bindUserRoutes := func(group fiber.Router) {
		group.Get("/me", b.auth.Require(roles.Client), b.handler.GetMe)
		group.Put("/me", b.auth.Require(roles.Client), b.handler.UpdateMe)
		group.Patch("/me", b.auth.Require(roles.Client), b.handler.UpdateMe)
		group.Get("/me/addresses", b.auth.Require(roles.Client), b.handler.GetAddresses)
		group.Post("/me/addresses", b.auth.Require(roles.Client), b.handler.AddAddress)
	}

	bindUserRoutes(router.Group("/users"))
	bindUserRoutes(router.Group("/user"))
}
