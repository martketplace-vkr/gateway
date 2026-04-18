package v1

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	authadmin "github.com/martketplace-vkr/auth/pkg/api/grpc/v1/admin"
	authclient "github.com/martketplace-vkr/auth/pkg/api/grpc/v1/client"
	"github.com/martketplace-vkr/gateway/internal/common/httpx"
	"github.com/martketplace-vkr/gateway/internal/services/auth/models"
	"github.com/martketplace-vkr/gateway/pkg/consts"
	userclient "github.com/martketplace-vkr/user/pkg/api/grpc/v1/client"
)

const adminSessionKey = "marketplace_admin_session"

type Handler struct {
	authClient      authclient.AuthClientServiceClient
	adminAuthClient authadmin.AuthAdminServiceClient
	userClient      userclient.UserClientServiceClient
	authTimeout     time.Duration
	userTimeout     time.Duration
}

func New(
	authClient authclient.AuthClientServiceClient,
	adminAuthClient authadmin.AuthAdminServiceClient,
	userClient userclient.UserClientServiceClient,
	authTimeout time.Duration,
	userTimeout time.Duration,
) *Handler {
	return &Handler{
		authClient:      authClient,
		adminAuthClient: adminAuthClient,
		userClient:      userClient,
		authTimeout:     authTimeout,
		userTimeout:     userTimeout,
	}
}

func (h *Handler) Register(c *fiber.Ctx) error {
	req := models.RegisterRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ctx, cancel := httpx.RPCContext(c, h.authTimeout)
	defer cancel()

	resp, err := h.authClient.Register(ctx, &authclient.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	h.syncUserProfile(c, resp.GetUserId(), req.Email)

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) Login(c *fiber.Ctx) error {
	req := models.LoginRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ctx, cancel := httpx.RPCContext(c, h.authTimeout)
	defer cancel()

	resp, err := h.authClient.Login(ctx, &authclient.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	h.setSessionCookie(c, consts.SessionKey, resp.GetRefreshToken())

	if userID, resolveErr := h.resolveUserID(c, resp.GetAccessToken()); resolveErr == nil {
		h.syncUserProfile(c, userID, req.Email)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) Refresh(c *fiber.Ctx) error {
	req := models.RefreshRequest{}
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
	}

	refreshToken := resolveRefreshToken(req.RefreshToken, c.Cookies(consts.SessionKey))
	if refreshToken == "" {
		return fiber.ErrUnauthorized
	}

	ctx, cancel := httpx.RPCContext(c, h.authTimeout)
	defer cancel()

	resp, err := h.authClient.RefreshToken(ctx, &authclient.RefreshTokenRequest{
		RefreshToken: refreshToken,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	h.setSessionCookie(c, consts.SessionKey, resp.GetRefreshToken())

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) Logout(c *fiber.Ctx) error {
	req := models.LogoutRequest{}
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
	}

	refreshToken := resolveRefreshToken(req.RefreshToken, c.Cookies(consts.SessionKey))
	if refreshToken == "" {
		return fiber.ErrUnauthorized
	}

	ctx, cancel := httpx.RPCContext(c, h.authTimeout)
	defer cancel()

	resp, err := h.authClient.Logout(ctx, &authclient.LogoutRequest{
		RefreshToken: refreshToken,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	h.clearSessionCookie(c, consts.SessionKey)

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) AdminRegister(c *fiber.Ctx) error {
	req := models.RegisterRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ctx, cancel := httpx.RPCContext(c, h.authTimeout)
	defer cancel()

	resp, err := h.adminAuthClient.Register(ctx, &authadmin.RegisterRequest{
		Email:       req.Email,
		Password:    req.Password,
		InviteToken: req.InviteToken,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) AdminLogin(c *fiber.Ctx) error {
	req := models.LoginRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ctx, cancel := httpx.RPCContext(c, h.authTimeout)
	defer cancel()

	resp, err := h.adminAuthClient.Login(ctx, &authadmin.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	h.setSessionCookie(c, adminSessionKey, resp.GetRefreshToken())

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) AdminRefresh(c *fiber.Ctx) error {
	req := models.RefreshRequest{}
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
	}

	refreshToken := resolveRefreshToken(req.RefreshToken, c.Cookies(adminSessionKey))
	if refreshToken == "" {
		return fiber.ErrUnauthorized
	}

	ctx, cancel := httpx.RPCContext(c, h.authTimeout)
	defer cancel()

	resp, err := h.adminAuthClient.RefreshToken(ctx, &authadmin.RefreshTokenRequest{
		RefreshToken: refreshToken,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	h.setSessionCookie(c, adminSessionKey, resp.GetRefreshToken())

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) AdminLogout(c *fiber.Ctx) error {
	req := models.LogoutRequest{}
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
	}

	refreshToken := resolveRefreshToken(req.RefreshToken, c.Cookies(adminSessionKey))
	if refreshToken == "" {
		return fiber.ErrUnauthorized
	}

	ctx, cancel := httpx.RPCContext(c, h.authTimeout)
	defer cancel()

	resp, err := h.adminAuthClient.Logout(ctx, &authadmin.LogoutRequest{
		RefreshToken: refreshToken,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	h.clearSessionCookie(c, adminSessionKey)

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) resolveUserID(c *fiber.Ctx, accessToken string) (int64, error) {
	ctx, cancel := httpx.RPCContext(c, h.authTimeout)
	defer cancel()

	resp, err := h.authClient.ValidateToken(ctx, &authclient.ValidateTokenRequest{
		Token: accessToken,
	})
	if err != nil {
		return 0, err
	}

	return resp.GetUserId(), nil
}

func (h *Handler) syncUserProfile(c *fiber.Ctx, userID int64, email string) {
	if h.userClient == nil || userID <= 0 || strings.TrimSpace(email) == "" {
		return
	}

	ctx, cancel := httpx.RPCContext(c, h.userTimeout)
	defer cancel()

	_, _ = h.userClient.UpdateUser(ctx, &userclient.UpdateUserRequest{
		UserId: userID,
		Email:  strings.TrimSpace(email),
	})
}

func (h *Handler) setSessionCookie(c *fiber.Ctx, name string, refreshToken string) {
	if strings.TrimSpace(refreshToken) == "" {
		return
	}

	c.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    refreshToken,
		Path:     "/",
		HTTPOnly: true,
		SameSite: fiber.CookieSameSiteLaxMode,
		Secure:   strings.EqualFold(c.Protocol(), "https"),
	})
}

func (h *Handler) clearSessionCookie(c *fiber.Ctx, name string) {
	c.Cookie(&fiber.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		SameSite: fiber.CookieSameSiteLaxMode,
		Secure:   strings.EqualFold(c.Protocol(), "https"),
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

func resolveRefreshToken(bodyToken string, cookieToken string) string {
	bodyToken = strings.TrimSpace(bodyToken)
	if bodyToken != "" {
		return bodyToken
	}

	return strings.TrimSpace(cookieToken)
}
