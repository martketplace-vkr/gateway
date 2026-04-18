package middleware

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	authadmin "github.com/martketplace-vkr/auth/pkg/api/grpc/v1/admin"
	authclient "github.com/martketplace-vkr/auth/pkg/api/grpc/v1/client"
	"github.com/martketplace-vkr/gateway/internal/common/httpx"
	"github.com/martketplace-vkr/gateway/pkg/roles"
	pkghttp "github.com/martketplace-vkr/pkg/server/http"
	"google.golang.org/grpc/codes"
)

const bearerPrefix = "Bearer "

type Auth struct {
	client      authclient.AuthClientServiceClient
	adminClient authadmin.AuthAdminServiceClient
	timeout     time.Duration
}

func NewAuth(
	client authclient.AuthClientServiceClient,
	adminClient authadmin.AuthAdminServiceClient,
	timeout time.Duration,
) *Auth {
	return &Auth{
		client:      client,
		adminClient: adminClient,
		timeout:     timeout,
	}
}

func (a *Auth) Require(required ...int64) fiber.Handler {
	requiredAccess := roles.GetAccess(required...)

	return func(c *fiber.Ctx) error {
		token, err := extractAccessToken(c)
		if err != nil {
			return err
		}

		user, err := a.validateToken(c, token, requiredAccess)
		if err != nil {
			return fiber.ErrUnauthorized
		}

		if user.ID <= 0 {
			return fiber.ErrUnauthorized
		}

		if requiredAccess != 0 && !roles.CheckAccess(user.PermissionKey, requiredAccess) {
			return fiber.ErrForbidden
		}

		c.Locals(pkghttp.UserLocalsKey, user)

		return c.Next()
	}
}

func (a *Auth) validateToken(c *fiber.Ctx, token string, requiredAccess int64) (*pkghttp.User, error) {
	if requiredAccess == roles.Admin {
		if user, err := a.validateAdminToken(c, token); err == nil {
			return user, nil
		}

		return nil, fiber.ErrUnauthorized
	}

	if user, err := a.validateClientToken(c, token); err == nil {
		return user, nil
	} else if !httpx.IsGRPCCode(err, codes.Unauthenticated) {
		return nil, err
	}

	return a.validateAdminToken(c, token)
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
	case roles.RoleClient, "":
		return roles.RoleClient, roles.Client
	default:
		if permission, ok := roles.RoleMap[roleName]; ok {
			return roleName, permission
		}

		return roles.RoleClient, roles.Client
	}
}
