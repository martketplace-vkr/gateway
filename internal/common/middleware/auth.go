package middleware

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	authadmin "github.com/martketplace-vkr/auth/pkg/api/grpc/v1/admin"
	authclient "github.com/martketplace-vkr/auth/pkg/api/grpc/v1/client"
	authvendor "github.com/martketplace-vkr/auth/pkg/api/grpc/v1/vendor"
	"github.com/martketplace-vkr/gateway/internal/common/httpx"
	"github.com/martketplace-vkr/gateway/pkg/consts"
	"github.com/martketplace-vkr/gateway/pkg/roles"
	pkghttp "github.com/martketplace-vkr/pkg/server/http"
	"google.golang.org/grpc/codes"
)

const bearerPrefix = "Bearer "

type Auth struct {
	client       authclient.AuthClientServiceClient
	adminClient  authadmin.AuthAdminServiceClient
	vendorClient authvendor.AuthVendorServiceClient
	timeout      time.Duration
}

func NewAuth(
	client authclient.AuthClientServiceClient,
	adminClient authadmin.AuthAdminServiceClient,
	vendorClient authvendor.AuthVendorServiceClient,
	timeout time.Duration,
) *Auth {
	return &Auth{
		client:       client,
		adminClient:  adminClient,
		vendorClient: vendorClient,
		timeout:      timeout,
	}
}

func (a *Auth) Require(required ...int64) fiber.Handler {
	requiredAccess := roles.GetAccess(required...)

	return func(c *fiber.Ctx) error {
		token, err := extractAccessToken(c)
		if err != nil {
			return err
		}

		user, refreshedAccessToken, err := a.validateOrRefresh(c, token, requiredAccess)
		if err != nil {
			return fiber.ErrUnauthorized
		}

		if user.ID <= 0 {
			return fiber.ErrUnauthorized
		}

		if requiredAccess != 0 && !roles.CheckAccess(user.PermissionKey, requiredAccess) {
			return fiber.ErrForbidden
		}

		if refreshedAccessToken != "" {
			c.Set(consts.AccessTokenHeader, refreshedAccessToken)
			c.Append(fiber.HeaderAccessControlExposeHeaders, consts.AccessTokenHeader)
		}

		c.Locals(pkghttp.UserLocalsKey, user)

		return c.Next()
	}
}

func (a *Auth) validateOrRefresh(c *fiber.Ctx, token string, requiredAccess int64) (*pkghttp.User, string, error) {
	user, err := a.validateToken(c, token, requiredAccess)
	if err == nil {
		return user, "", nil
	}

	if err != fiber.ErrUnauthorized && !httpx.IsGRPCCode(err, codes.Unauthenticated) {
		return nil, "", err
	}

	refreshedAccessToken, refreshErr := a.tryRefresh(c, requiredAccess)
	if refreshErr != nil {
		return nil, "", err
	}

	user, err = a.validateToken(c, refreshedAccessToken, requiredAccess)
	if err != nil {
		return nil, "", err
	}

	return user, refreshedAccessToken, nil
}

func (a *Auth) validateToken(c *fiber.Ctx, token string, requiredAccess int64) (*pkghttp.User, error) {
	if requiredAccess == roles.Admin {
		if user, err := a.validateAdminToken(c, token); err == nil {
			return user, nil
		}

		return nil, fiber.ErrUnauthorized
	}

	validateSequence := a.resolveValidationSequence(requiredAccess)
	for _, validator := range validateSequence {
		user, err := validator(c, token)
		if err == nil {
			return user, nil
		}
		if !httpx.IsGRPCCode(err, codes.Unauthenticated) {
			return nil, err
		}
	}

	return nil, fiber.ErrUnauthorized
}

func (a *Auth) tryRefresh(c *fiber.Ctx, requiredAccess int64) (string, error) {
	for _, refresher := range a.resolveRefreshSequence(requiredAccess) {
		refreshToken := strings.TrimSpace(c.Cookies(refresher.cookieName))
		if refreshToken == "" {
			continue
		}

		accessToken, rotatedRefreshToken, err := refresher.refresh(c, refreshToken)
		if err == nil {
			a.setRefreshCookie(c, refresher.cookieName, rotatedRefreshToken)
			return accessToken, nil
		}

		if !httpx.IsGRPCCode(err, codes.Unauthenticated) {
			return "", err
		}
	}

	return "", fiber.ErrUnauthorized
}

func (a *Auth) validateClientToken(c *fiber.Ctx, token string) (*pkghttp.User, error) {
	ctx, cancel := httpx.RPCContext(c, a.timeout)
	defer cancel()

	resp, err := a.client.ValidateToken(ctx, &authclient.ValidateTokenRequest{
		Token: token,
	})
	if err != nil {
		return nil, err
	}

	roleName, permission := resolveRole(resp.GetRole())

	return &pkghttp.User{
		ID:            resp.GetUserId(),
		Role:          roleName,
		Login:         resp.Login,
		PermissionKey: permission,
	}, nil
}

func (a *Auth) validateAdminToken(c *fiber.Ctx, token string) (*pkghttp.User, error) {
	ctx, cancel := httpx.RPCContext(c, a.timeout)
	defer cancel()

	resp, err := a.adminClient.ValidateToken(ctx, &authadmin.ValidateTokenRequest{
		Token: token,
	})
	if err != nil {
		return nil, err
	}

	roleName, permission := resolveRole(resp.GetRole())

	return &pkghttp.User{
		ID:            resp.GetUserId(),
		Role:          roleName,
		Login:         resp.Login,
		PermissionKey: permission,
	}, nil
}

func (a *Auth) validateVendorToken(c *fiber.Ctx, token string) (*pkghttp.User, error) {
	ctx, cancel := httpx.RPCContext(c, a.timeout)
	defer cancel()

	resp, err := a.vendorClient.ValidateToken(ctx, &authvendor.ValidateTokenRequest{
		Token: token,
	})
	if err != nil {
		return nil, err
	}

	roleName, permission := resolveRole(resp.GetRole())

	return &pkghttp.User{
		ID:            resp.GetVendorId(),
		Role:          roleName,
		Login:         resp.Login,
		PermissionKey: permission,
	}, nil
}

func CurrentUser(c *fiber.Ctx) (*pkghttp.User, error) {
	user, ok := c.Locals(pkghttp.UserLocalsKey).(*pkghttp.User)
	if !ok || user == nil || user.ID <= 0 {
		return nil, fiber.ErrUnauthorized
	}

	return user, nil
}

func extractAccessToken(c *fiber.Ctx) (string, error) {
	token := strings.TrimSpace(c.Get(fiber.HeaderAuthorization))
	if token == "" {
		return "", fiber.ErrUnauthorized
	}

	if len(token) >= len(bearerPrefix) && strings.EqualFold(token[:len(bearerPrefix)], bearerPrefix) {
		token = strings.TrimSpace(token[len(bearerPrefix):])
	}

	if token == "" {
		return "", fiber.ErrUnauthorized
	}

	return token, nil
}

func resolveRole(roleName string) (string, int64) {
	roleName = strings.TrimSpace(strings.ToLower(roleName))

	switch roleName {
	case roles.RoleAdmin:
		return roles.RoleAdmin, roles.Admin
	case roles.RoleAll:
		return roles.RoleAll, roles.All
	case roles.RoleVendor:
		return roles.RoleVendor, roles.Vendor
	case roles.RoleClient, "":
		return roles.RoleClient, roles.Client
	default:
		if permission, ok := roles.RoleMap[roleName]; ok {
			return roleName, permission
		}

		return roles.RoleClient, roles.Client
	}
}

func (a *Auth) resolveValidationSequence(requiredAccess int64) []func(*fiber.Ctx, string) (*pkghttp.User, error) {
	switch {
	case requiredAccess == roles.Vendor:
		return []func(*fiber.Ctx, string) (*pkghttp.User, error){
			a.validateVendorToken,
			a.validateAdminToken,
		}
	case requiredAccess == roles.Client:
		return []func(*fiber.Ctx, string) (*pkghttp.User, error){
			a.validateClientToken,
			a.validateAdminToken,
		}
	case requiredAccess&(roles.Client|roles.Vendor) == roles.All:
		return []func(*fiber.Ctx, string) (*pkghttp.User, error){
			a.validateClientToken,
			a.validateVendorToken,
			a.validateAdminToken,
		}
	default:
		return []func(*fiber.Ctx, string) (*pkghttp.User, error){
			a.validateClientToken,
			a.validateVendorToken,
			a.validateAdminToken,
		}
	}
}

type refreshHandler struct {
	cookieName string
	refresh    func(*fiber.Ctx, string) (string, string, error)
}

func (a *Auth) resolveRefreshSequence(requiredAccess int64) []refreshHandler {
	switch {
	case requiredAccess == roles.Vendor:
		return []refreshHandler{
			{cookieName: consts.VendorSessionKey, refresh: a.refreshVendorToken},
			{cookieName: consts.AdminSessionKey, refresh: a.refreshAdminToken},
		}
	case requiredAccess == roles.Client:
		return []refreshHandler{
			{cookieName: consts.SessionKey, refresh: a.refreshClientToken},
			{cookieName: consts.AdminSessionKey, refresh: a.refreshAdminToken},
		}
	case requiredAccess&(roles.Client|roles.Vendor) == roles.All:
		return []refreshHandler{
			{cookieName: consts.SessionKey, refresh: a.refreshClientToken},
			{cookieName: consts.VendorSessionKey, refresh: a.refreshVendorToken},
			{cookieName: consts.AdminSessionKey, refresh: a.refreshAdminToken},
		}
	default:
		return []refreshHandler{
			{cookieName: consts.SessionKey, refresh: a.refreshClientToken},
			{cookieName: consts.VendorSessionKey, refresh: a.refreshVendorToken},
			{cookieName: consts.AdminSessionKey, refresh: a.refreshAdminToken},
		}
	}
}

func (a *Auth) refreshClientToken(c *fiber.Ctx, refreshToken string) (string, string, error) {
	ctx, cancel := httpx.RPCContext(c, a.timeout)
	defer cancel()

	resp, err := a.client.RefreshToken(ctx, &authclient.RefreshTokenRequest{
		RefreshToken: refreshToken,
	})
	if err != nil {
		return "", "", err
	}

	return strings.TrimSpace(resp.GetAccessToken()), strings.TrimSpace(resp.GetRefreshToken()), nil
}

func (a *Auth) refreshAdminToken(c *fiber.Ctx, refreshToken string) (string, string, error) {
	ctx, cancel := httpx.RPCContext(c, a.timeout)
	defer cancel()

	resp, err := a.adminClient.RefreshToken(ctx, &authadmin.RefreshTokenRequest{
		RefreshToken: refreshToken,
	})
	if err != nil {
		return "", "", err
	}

	return strings.TrimSpace(resp.GetAccessToken()), strings.TrimSpace(resp.GetRefreshToken()), nil
}

func (a *Auth) refreshVendorToken(c *fiber.Ctx, refreshToken string) (string, string, error) {
	ctx, cancel := httpx.RPCContext(c, a.timeout)
	defer cancel()

	resp, err := a.vendorClient.RefreshToken(ctx, &authvendor.RefreshTokenRequest{
		RefreshToken: refreshToken,
	})
	if err != nil {
		return "", "", err
	}

	return strings.TrimSpace(resp.GetAccessToken()), strings.TrimSpace(resp.GetRefreshToken()), nil
}

func (a *Auth) setRefreshCookie(c *fiber.Ctx, name string, refreshToken string) {
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
