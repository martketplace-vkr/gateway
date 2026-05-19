package v1

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/martketplace-vkr/gateway/internal/common/httpx"
	"github.com/martketplace-vkr/gateway/internal/common/middleware"
	"github.com/martketplace-vkr/gateway/internal/services/review/models"
	reviewadmin "github.com/martketplace-vkr/review/pkg/api/grpc/v1/admin"
	reviewclient "github.com/martketplace-vkr/review/pkg/api/grpc/v1/client"
	reviewvendor "github.com/martketplace-vkr/review/pkg/api/grpc/v1/vendor"
)

type Handler struct {
	client  reviewclient.ReviewClientServiceClient
	vendor  reviewvendor.ReviewVendorServiceClient
	admin   reviewadmin.ReviewAdminServiceClient
	timeout time.Duration
}

func New(
	client reviewclient.ReviewClientServiceClient,
	vendor reviewvendor.ReviewVendorServiceClient,
	admin reviewadmin.ReviewAdminServiceClient,
	timeout time.Duration,
) *Handler {
	return &Handler{
		client:  client,
		vendor:  vendor,
		admin:   admin,
		timeout: timeout,
	}
}

func (h *Handler) ListProductReviews(c *fiber.Ctx) error {
	productID, err := parsePositiveIntParam(c, "product_id")
	if err != nil {
		return err
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.client.ListProductReviews(ctx, &reviewclient.ListProductReviewsRequest{
		ProductId: productID,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) GetMyProductReview(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	productID, err := parsePositiveIntParam(c, "product_id")
	if err != nil {
		return err
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.client.ListProductReviews(ctx, &reviewclient.ListProductReviewsRequest{
		ProductId: productID,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	for _, review := range resp.GetReviews() {
		if review.GetAuthorUserId() == user.ID {
			return httpx.WriteProtoJSON(c, &reviewclient.CreateReviewResponse{
				Review:  review,
				Summary: resp.GetSummary(),
			})
		}
	}

	return httpx.WriteProtoJSON(c, &reviewclient.CreateReviewResponse{
		Summary: resp.GetSummary(),
	})
}

func (h *Handler) CreateReview(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	productID, err := parsePositiveIntParam(c, "product_id")
	if err != nil {
		return err
	}

	req := models.CreateReviewRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.client.CreateReview(ctx, &reviewclient.CreateReviewRequest{
		ProductId:    productID,
		AuthorUserId: user.ID,
		AuthorName:   user.Login,
		Rating:       req.Rating,
		Comment:      strings.TrimSpace(req.Comment),
		ImageUrls:    req.ImageURLs,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) VoteReview(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	reviewID, err := parsePositiveIntParam(c, "review_id")
	if err != nil {
		return err
	}

	req := models.VoteReviewRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.client.VoteReview(ctx, &reviewclient.VoteReviewRequest{
		ReviewId: reviewID,
		UserId:   user.ID,
		Vote:     strings.TrimSpace(req.Vote),
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) ReportReview(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	reviewID, err := parsePositiveIntParam(c, "review_id")
	if err != nil {
		return err
	}

	req := models.ReportReviewRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.client.ReportReview(ctx, &reviewclient.ReportReviewRequest{
		ReviewId:       reviewID,
		ReporterUserId: user.ID,
		Reason:         strings.TrimSpace(req.Reason),
		Details:        strings.TrimSpace(req.Details),
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) ListVendorReviews(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.vendor.ListVendorReviews(ctx, &reviewvendor.ListVendorReviewsRequest{
		VendorId: user.ID,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) ReplyReview(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	reviewID, err := parsePositiveIntParam(c, "review_id")
	if err != nil {
		return err
	}

	req := models.ReplyReviewRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.vendor.ReplyReview(ctx, &reviewvendor.ReplyReviewRequest{
		VendorId: user.ID,
		ReviewId: reviewID,
		Comment:  strings.TrimSpace(req.Comment),
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) DisputeReview(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	reviewID, err := parsePositiveIntParam(c, "review_id")
	if err != nil {
		return err
	}

	req := models.DisputeReviewRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.vendor.DisputeReview(ctx, &reviewvendor.DisputeReviewRequest{
		VendorId: user.ID,
		ReviewId: reviewID,
		Reason:   strings.TrimSpace(req.Reason),
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) ListReviewDisputes(c *fiber.Ctx) error {
	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.admin.ListReviewDisputes(ctx, &reviewadmin.ListReviewDisputesRequest{
		Status: strings.TrimSpace(c.Query("status")),
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) ListReviewReports(c *fiber.Ctx) error {
	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.admin.ListReviewReports(ctx, &reviewadmin.ListReviewReportsRequest{
		Status: strings.TrimSpace(c.Query("status")),
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) ResolveReviewDispute(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	reviewID, err := parsePositiveIntParam(c, "review_id")
	if err != nil {
		return err
	}
	disputeID, err := parsePositiveIntParam(c, "dispute_id")
	if err != nil {
		return err
	}

	req := models.ResolveReviewDisputeRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.admin.ResolveReviewDispute(ctx, &reviewadmin.ResolveReviewDisputeRequest{
		AdminId:   user.ID,
		ReviewId:  reviewID,
		DisputeId: disputeID,
		Decision:  strings.TrimSpace(req.Decision),
		Comment:   strings.TrimSpace(req.Comment),
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) DeleteReview(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	reviewID, err := parsePositiveIntParam(c, "review_id")
	if err != nil {
		return err
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.admin.DeleteReview(ctx, &reviewadmin.DeleteReviewRequest{
		AdminId:  user.ID,
		ReviewId: reviewID,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func parsePositiveIntParam(c *fiber.Ctx, key string) (int64, error) {
	value, err := strconv.ParseInt(c.Params(key), 10, 64)
	if err != nil || value <= 0 {
		return 0, fiber.NewError(fiber.StatusBadRequest, "invalid "+key)
	}

	return value, nil
}
