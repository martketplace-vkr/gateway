package v1

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/martketplace-vkr/gateway/internal/common/httpx"
	"github.com/martketplace-vkr/gateway/internal/common/middleware"
	"github.com/martketplace-vkr/gateway/internal/services/user/models"
	userclient "github.com/martketplace-vkr/user/pkg/api/grpc/v1/client"
	"google.golang.org/grpc/codes"
)

type Handler struct {
	userClient userclient.UserClientServiceClient
	timeout    time.Duration
}

func New(userClient userclient.UserClientServiceClient, timeout time.Duration) *Handler {
	return &Handler{
		userClient: userClient,
		timeout:    timeout,
	}
}

func (h *Handler) GetMe(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.userClient.GetUser(ctx, &userclient.GetUserRequest{
		UserId: user.ID,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) UpdateMe(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	req := models.UpdateUserRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.userClient.UpdateUser(ctx, &userclient.UpdateUserRequest{
		UserId:    user.ID,
		Email:     user.Login,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		AvatarUrl: req.AvatarURL,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) GetAddresses(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.userClient.GetUserAddresses(ctx, &userclient.GetUserAddressesRequest{
		UserID: user.ID,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) AddAddress(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	req := models.AddAddressRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.userClient.AddUserAddress(ctx, &userclient.AddUserAddressRequest{
		Address: &userclient.Address{
			UserId:     user.ID,
			Country:    req.Country,
			City:       req.City,
			Street:     req.Street,
			PostalCode: req.PostalCode,
		},
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) resolveEmail(c *fiber.Ctx, userID int64, email *string) (string, error) {
	if email != nil {
		trimmed := strings.TrimSpace(*email)
		if trimmed != "" {
			return trimmed, nil
		}
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.userClient.GetUser(ctx, &userclient.GetUserRequest{
		UserId: userID,
	})
	if err != nil {
		if httpx.IsGRPCCode(err, codes.NotFound) {
			return "", fiber.NewError(fiber.StatusBadRequest, "email is required for initial profile creation")
		}

		return "", httpx.MapGRPCError(err)
	}

	if strings.TrimSpace(resp.GetEmail()) == "" {
		return "", fiber.NewError(fiber.StatusBadRequest, "email is required for initial profile creation")
	}

	return strings.TrimSpace(resp.GetEmail()), nil
}
