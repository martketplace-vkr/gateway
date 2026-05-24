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
	router.Post("/balance/webhooks/mock-provider", b.handler.HandleMockProviderWebhook)

	balance := router.Group("/balance", b.auth.Require(roles.Client))

	balance.Get("/wallet", b.handler.GetWallet)
	balance.Get("/deposit-addresses", b.handler.GetDepositAddressList)
	balance.Get("/transactions", b.handler.GetWalletTransactions)
	balance.Post("/top-ups", b.handler.CreateTopUp)
	balance.Post("/top-ups/crypto", b.handler.CreateCryptoTopUp)
	balance.Post("/top-ups/rub", b.handler.CreateRubTopUp)
	balance.Get("/top-ups", b.handler.GetTopUpList)
	balance.Get("/top-ups/:top_up_id", b.handler.GetTopUp)
	balance.Post("/withdrawals", b.handler.CreateWithdrawal)
	balance.Get("/withdrawals/:withdrawal_id", b.handler.GetWithdrawal)

	vendorBalance := router.Group("/vendor/balance", b.auth.Require(roles.Vendor))
	vendorBalance.Get("/wallet", b.handler.GetVendorWallet)
	vendorBalance.Get("/transactions", b.handler.GetVendorWalletTransactions)

	adminBalance := router.Group("/admin/balance", b.auth.Require(roles.Admin))
	adminBalance.Get("/wallet", b.handler.GetAdminWallet)
	adminBalance.Get("/transactions", b.handler.GetAdminWalletTransactions)
	adminBalance.Get("/top-ups", b.handler.ListAdminTopUps)
	adminBalance.Post("/top-ups/:external_id/confirm", b.handler.ConfirmAdminTopUp)
}
