package v1

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/martketplace-vkr/pkg/server/http"
)

type Binder struct {
	server  *http.Server
	handler *Handler
}

func NewBinder(server *http.Server, handler *Handler) *Binder {
	return &Binder{
		server:  server,
		handler: handler,
	}
}

func (b *Binder) BindRoutes(_ context.Context) {
	for _, prefix := range []string{"/api/v1", "/client/v1"} {
		b.bind(b.server.Group(prefix))
	}
}

func (b *Binder) bind(router fiber.Router) {
	auth := router.Group("/auth")

	auth.Post("/register", b.handler.Register)
	auth.Post("/sign-up", b.handler.Register)
	auth.Post("/login", b.handler.Login)
	auth.Post("/sign-in", b.handler.Login)
	auth.Post("/refresh", b.handler.Refresh)
	auth.Post("/logout", b.handler.Logout)
	auth.Post("/sign-out", b.handler.Logout)

	adminAuth := router.Group("/admin/auth")
	adminAuth.Post("/register", b.handler.AdminRegister)
	adminAuth.Post("/sign-up", b.handler.AdminRegister)
	adminAuth.Post("/login", b.handler.AdminLogin)
	adminAuth.Post("/sign-in", b.handler.AdminLogin)
	adminAuth.Post("/refresh", b.handler.AdminRefresh)
	adminAuth.Post("/logout", b.handler.AdminLogout)
	adminAuth.Post("/sign-out", b.handler.AdminLogout)
	adminAuth.Get("/me", b.handler.AdminMe)

	vendorAuth := router.Group("/vendor/auth")
	vendorAuth.Post("/register", b.handler.VendorRegister)
	vendorAuth.Post("/sign-up", b.handler.VendorRegister)
	vendorAuth.Post("/login", b.handler.VendorLogin)
	vendorAuth.Post("/sign-in", b.handler.VendorLogin)
	vendorAuth.Post("/refresh", b.handler.VendorRefresh)
	vendorAuth.Post("/logout", b.handler.VendorLogout)
	vendorAuth.Post("/sign-out", b.handler.VendorLogout)
	vendorAuth.Get("/me", b.handler.VendorMe)
}
