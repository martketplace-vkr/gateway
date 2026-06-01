package v1

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	authadmin "github.com/martketplace-vkr/auth/pkg/api/grpc/v1/admin"
	authclient "github.com/martketplace-vkr/auth/pkg/api/grpc/v1/client"
	authvendor "github.com/martketplace-vkr/auth/pkg/api/grpc/v1/vendor"
	"github.com/martketplace-vkr/gateway/internal/common/httpx"
	"github.com/martketplace-vkr/gateway/internal/services/auth/models"
	"github.com/martketplace-vkr/gateway/pkg/consts"
	userclient "github.com/martketplace-vkr/user/pkg/api/grpc/v1/client"
)

func (h *Handler) AdminClients(c *fiber.Ctx) error {
	adminID, err := h.validateAdmin(c)
	if err != nil || adminID <= 0 {
		return err
	}
	limit, _ := strconv.ParseUint(c.Query("limit", "50"), 10, 32)
	offset, _ := strconv.ParseUint(c.Query("offset", "0"), 10, 32)
	ctx, cancel := httpx.RPCContext(c, h.authTimeout)
	defer cancel()
	resp, err := h.adminAuthClient.ListClients(ctx, &authadmin.ListClientsRequest{
		Query: c.Query("query"), Status: c.Query("status"), Limit: uint32(limit), Offset: uint32(offset),
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}
	result := models.ClientsResponse{Total: resp.GetTotal(), Clients: make([]models.ClientProfile, 0, len(resp.GetClients()))}
	for _, client := range resp.GetClients() {
		result.Clients = append(result.Clients, h.clientProfile(c, client, false))
	}
	return c.JSON(result)
}

func (h *Handler) AdminClient(c *fiber.Ctx) error {
	if _, err := h.validateAdmin(c); err != nil {
		return err
	}
	clientID, err := strconv.ParseInt(c.Params("client_id"), 10, 64)
	if err != nil || clientID <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid client_id")
	}
	ctx, cancel := httpx.RPCContext(c, h.authTimeout)
	defer cancel()
	client, err := h.adminAuthClient.GetClient(ctx, &authadmin.GetClientRequest{ClientId: clientID})
	if err != nil {
		return httpx.MapGRPCError(err)
	}
	return c.JSON(h.clientProfile(c, client, true))
}

func (h *Handler) UpdateAdminClientStatus(c *fiber.Ctx) error {
	adminID, err := h.validateAdmin(c)
	if err != nil {
		return err
	}
	clientID, err := strconv.ParseInt(c.Params("client_id"), 10, 64)
	if err != nil || clientID <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "invalid client_id")
	}
	req := models.UpdateClientStatusRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	ctx, cancel := httpx.RPCContext(c, h.authTimeout)
	defer cancel()
	client, err := h.adminAuthClient.UpdateClientStatus(ctx, &authadmin.UpdateClientStatusRequest{
		ClientId: clientID, AdminId: adminID, Status: strings.TrimSpace(req.Status), Reason: strings.TrimSpace(req.Reason),
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}
	return c.JSON(h.clientProfile(c, client, true))
}

func (h *Handler) validateAdmin(c *fiber.Ctx) (int64, error) {
	token, err := extractBearerToken(c)
	if err != nil {
		return 0, err
	}
	ctx, cancel := httpx.RPCContext(c, h.authTimeout)
	defer cancel()
	resp, err := h.adminAuthClient.ValidateToken(ctx, &authadmin.ValidateTokenRequest{Token: token})
	if err != nil {
		return 0, httpx.MapGRPCError(err)
	}
	return resp.GetUserId(), nil
}

func (h *Handler) clientProfile(c *fiber.Ctx, client *authadmin.Client, withAddresses bool) models.ClientProfile {
	result := models.ClientProfile{
		ID: client.GetClientId(), Email: client.GetEmail(), EmailVerified: client.GetEmailVerified(),
		Status: client.GetStatus(), StatusReason: client.GetStatusReason(), CreatedAt: client.GetCreatedAt(),
		UpdatedAt: client.GetUpdatedAt(), LastActivityAt: client.GetLastActivityAt(),
	}
	ctx, cancel := httpx.RPCContext(c, h.userTimeout)
	defer cancel()
	if profile, err := h.userClient.GetUser(ctx, &userclient.GetUserRequest{UserId: client.GetClientId()}); err == nil {
		result.FirstName = profile.GetFirstName()
		result.LastName = profile.GetLastName()
		result.AvatarURL = profile.GetAvatarUrl()
	}
	if withAddresses {
		ctx, cancel := httpx.RPCContext(c, h.userTimeout)
		defer cancel()
		if addresses, err := h.userClient.GetUserAddresses(ctx, &userclient.GetUserAddressesRequest{UserID: client.GetClientId()}); err == nil {
			result.Addresses = addresses.GetAddresses()
		}
	}
	for _, event := range client.GetModerationEvents() {
		result.ModerationEvents = append(result.ModerationEvents, models.ClientModerationEvent{
			ID: event.GetId(), AdminID: event.GetAdminId(), OldStatus: event.GetOldStatus(),
			NewStatus: event.GetNewStatus(), Reason: event.GetReason(), CreatedAt: event.GetCreatedAt(),
		})
	}
	return result
}

type Handler struct {
	authClient       authclient.AuthClientServiceClient
	adminAuthClient  authadmin.AuthAdminServiceClient
	vendorAuthClient authvendor.AuthVendorServiceClient
	userClient       userclient.UserClientServiceClient
	authTimeout      time.Duration
	userTimeout      time.Duration
}

func New(
	authClient authclient.AuthClientServiceClient,
	adminAuthClient authadmin.AuthAdminServiceClient,
	vendorAuthClient authvendor.AuthVendorServiceClient,
	userClient userclient.UserClientServiceClient,
	authTimeout time.Duration,
	userTimeout time.Duration,
) *Handler {
	return &Handler{
		authClient:       authClient,
		adminAuthClient:  adminAuthClient,
		vendorAuthClient: vendorAuthClient,
		userClient:       userClient,
		authTimeout:      authTimeout,
		userTimeout:      userTimeout,
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

	h.setSessionCookie(c, consts.AdminSessionKey, resp.GetRefreshToken())

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) AdminRefresh(c *fiber.Ctx) error {
	req := models.RefreshRequest{}
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
	}

	refreshToken := resolveRefreshToken(req.RefreshToken, c.Cookies(consts.AdminSessionKey))
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

	h.setSessionCookie(c, consts.AdminSessionKey, resp.GetRefreshToken())

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) AdminLogout(c *fiber.Ctx) error {
	req := models.LogoutRequest{}
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
	}

	refreshToken := resolveRefreshToken(req.RefreshToken, c.Cookies(consts.AdminSessionKey))
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

	h.clearSessionCookie(c, consts.AdminSessionKey)

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) VendorRegister(c *fiber.Ctx) error {
	req := models.RegisterRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ctx, cancel := httpx.RPCContext(c, h.authTimeout)
	defer cancel()

	resp, err := h.vendorAuthClient.Register(ctx, &authvendor.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) VendorLogin(c *fiber.Ctx) error {
	req := models.LoginRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ctx, cancel := httpx.RPCContext(c, h.authTimeout)
	defer cancel()

	resp, err := h.vendorAuthClient.Login(ctx, &authvendor.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	h.setSessionCookie(c, consts.VendorSessionKey, resp.GetRefreshToken())

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) VendorRefresh(c *fiber.Ctx) error {
	req := models.RefreshRequest{}
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
	}

	refreshToken := resolveRefreshToken(req.RefreshToken, c.Cookies(consts.VendorSessionKey))
	if refreshToken == "" {
		return fiber.ErrUnauthorized
	}

	ctx, cancel := httpx.RPCContext(c, h.authTimeout)
	defer cancel()

	resp, err := h.vendorAuthClient.RefreshToken(ctx, &authvendor.RefreshTokenRequest{
		RefreshToken: refreshToken,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	h.setSessionCookie(c, consts.VendorSessionKey, resp.GetRefreshToken())

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) VendorLogout(c *fiber.Ctx) error {
	req := models.LogoutRequest{}
	if len(c.Body()) > 0 {
		if err := c.BodyParser(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
	}

	refreshToken := resolveRefreshToken(req.RefreshToken, c.Cookies(consts.VendorSessionKey))
	if refreshToken == "" {
		return fiber.ErrUnauthorized
	}

	ctx, cancel := httpx.RPCContext(c, h.authTimeout)
	defer cancel()

	resp, err := h.vendorAuthClient.Logout(ctx, &authvendor.LogoutRequest{
		RefreshToken: refreshToken,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	h.clearSessionCookie(c, consts.VendorSessionKey)

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) AdminMe(c *fiber.Ctx) error {
	token, err := extractBearerToken(c)
	if err != nil {
		return err
	}

	ctx, cancel := httpx.RPCContext(c, h.authTimeout)
	defer cancel()

	resp, err := h.adminAuthClient.ValidateToken(ctx, &authadmin.ValidateTokenRequest{
		Token: token,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return c.JSON(models.AuthMeResponse{
		ID:    resp.GetUserId(),
		Email: resp.GetLogin(),
		Role:  strings.TrimSpace(resp.GetRole()),
	})
}

func (h *Handler) AdminVendors(c *fiber.Ctx) error {
	token, err := extractBearerToken(c)
	if err != nil {
		return err
	}

	ctx, cancel := httpx.RPCContext(c, h.authTimeout)
	defer cancel()

	if _, err := h.adminAuthClient.ValidateToken(ctx, &authadmin.ValidateTokenRequest{Token: token}); err != nil {
		return httpx.MapGRPCError(err)
	}

	resp, err := h.adminAuthClient.ListVendors(ctx, &authadmin.ListVendorsRequest{})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	vendors := make([]models.VendorResponse, 0, len(resp.GetVendors()))
	for _, vendor := range resp.GetVendors() {
		vendors = append(vendors, models.VendorResponse{
			ID:    vendor.GetVendorId(),
			Email: vendor.GetEmail(),
		})
	}

	return c.JSON(models.VendorsResponse{Vendors: vendors})
}

func (h *Handler) VendorMe(c *fiber.Ctx) error {
	token, err := extractBearerToken(c)
	if err != nil {
		return err
	}

	ctx, cancel := httpx.RPCContext(c, h.authTimeout)
	defer cancel()

	resp, err := h.vendorAuthClient.ValidateToken(ctx, &authvendor.ValidateTokenRequest{
		Token: token,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return c.JSON(models.AuthMeResponse{
		ID:    resp.GetVendorId(),
		Email: resp.GetLogin(),
		Role:  strings.TrimSpace(resp.GetRole()),
	})
}

func extractBearerToken(c *fiber.Ctx) (string, error) {
	token := strings.TrimSpace(c.Get(fiber.HeaderAuthorization))
	if token == "" {
		return "", fiber.ErrUnauthorized
	}

	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[7:])
	}

	if token == "" {
		return "", fiber.ErrUnauthorized
	}

	return token, nil
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
