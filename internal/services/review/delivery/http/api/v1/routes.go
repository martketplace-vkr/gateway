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
	router.Get("/products/:product_id/reviews", b.handler.ListProductReviews)
	router.Get("/products/:product_id/reviews/my", b.auth.Require(roles.Client), b.handler.GetMyProductReview)
	router.Post("/products/:product_id/reviews", b.auth.Require(roles.Client), b.handler.CreateReview)
	router.Get("/catalog/products/:product_id/reviews", b.handler.ListProductReviews)
	router.Get("/catalog/products/:product_id/reviews/my", b.auth.Require(roles.Client), b.handler.GetMyProductReview)
	router.Post("/catalog/products/:product_id/reviews", b.auth.Require(roles.Client), b.handler.CreateReview)

	reviews := router.Group("/reviews", b.auth.Require(roles.Client))
	reviews.Put("/:review_id/vote", b.handler.VoteReview)
	reviews.Post("/:review_id/reports", b.handler.ReportReview)

	vendorReviews := router.Group("/vendor/reviews", b.auth.Require(roles.Vendor))
	vendorReviews.Get("", b.handler.ListVendorReviews)
	vendorReviews.Get("/", b.handler.ListVendorReviews)
	vendorReviews.Post("/:review_id/reply", b.handler.ReplyReview)
	vendorReviews.Delete("/:review_id/reply", b.handler.DeleteReviewReply)
	vendorReviews.Post("/:review_id/disputes", b.handler.DisputeReview)
	vendorReviews.Delete("/:review_id/disputes", b.handler.CancelReviewDispute)

	adminReviews := router.Group("/admin/reviews", b.auth.Require(roles.Admin))
	adminReviews.Get("/disputes", b.handler.ListReviewDisputes)
	adminReviews.Get("/reports", b.handler.ListReviewReports)
	adminReviews.Post("/:review_id/disputes/:dispute_id/resolve", b.handler.ResolveReviewDispute)
	adminReviews.Delete("/:review_id", b.handler.DeleteReview)
}
