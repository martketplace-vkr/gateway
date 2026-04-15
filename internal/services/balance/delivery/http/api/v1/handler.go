package v1

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	balanceclient "github.com/martketplace-vkr/balance/pkg/api/grpc/v1/client"
	balancedomain "github.com/martketplace-vkr/balance/pkg/api/grpc/v1/domain"
	"github.com/martketplace-vkr/gateway/internal/common/httpx"
	"github.com/martketplace-vkr/gateway/internal/common/middleware"
	"github.com/martketplace-vkr/gateway/internal/services/balance/models"
	"github.com/martketplace-vkr/pkg/utils/currency"
)

const (
	defaultTopUpProviderName = "USDT-TRC20"
	defaultTopUpNetwork      = "TRON"
)

type Handler struct {
	balanceClient balanceclient.BalanceClientServiceClient
	timeout       time.Duration
}

func New(balanceClient balanceclient.BalanceClientServiceClient, timeout time.Duration) *Handler {
	return &Handler{
		balanceClient: balanceClient,
		timeout:       timeout,
	}
}

func (h *Handler) GetWallet(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.balanceClient.GetWallet(ctx, &balanceclient.GetWalletRequest{
		UserId: user.ID,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) GetWalletTransactions(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	limit, err := parseOptionalUint32Query(c, "limit", 50)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid limit")
	}
	offset, err := parseOptionalUint64Query(c, "offset", 0)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid offset")
	}
	currencyCode, err := parseOptionalInt64PtrQuery(c, "currency_code")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid currency_code")
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.balanceClient.GetWalletTransactions(ctx, &balanceclient.GetWalletTransactionsRequest{
		UserId:       user.ID,
		CurrencyCode: currencyCode,
		Limit:        limit,
		Offset:       offset,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) CreateCryptoTopUp(c *fiber.Ctx) error {
	return h.createTopUp(c, true)
}

func (h *Handler) CreateTopUp(c *fiber.Ctx) error {
	return h.createTopUp(c, false)
}

func (h *Handler) createTopUp(c *fiber.Ctx, forceCrypto bool) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	req := models.CreateTopUpRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	req.Amount = strings.TrimSpace(req.Amount)
	if req.Amount == "" {
		return fiber.NewError(fiber.StatusBadRequest, "amount is required")
	}
	if req.CurrencyCode == 0 {
		req.CurrencyCode = int64(currency.USDTinTRC)
	}
	if req.ProviderType == 0 || forceCrypto {
		req.ProviderType = int32(balancedomain.ProviderType_PROVIDER_TYPE_CRYPTO)
	}
	if req.ProviderName == "" {
		req.ProviderName = defaultTopUpProviderName
	}
	if req.Network == "" {
		req.Network = defaultTopUpNetwork
	}
	if req.IdempotencyKey == "" {
		req.IdempotencyKey = newIdempotencyKey("topup", user.ID)
	}

	network := strings.TrimSpace(req.Network)
	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.balanceClient.CreateTopUp(ctx, &balanceclient.CreateTopUpRequest{
		UserId: user.ID,
		Money: &balancedomain.Money{
			Amount:       req.Amount,
			CurrencyCode: req.CurrencyCode,
		},
		ProviderType:   balancedomain.ProviderType(req.ProviderType),
		ProviderName:   strings.TrimSpace(req.ProviderName),
		IdempotencyKey: strings.TrimSpace(req.IdempotencyKey),
		SuccessUrl:     strings.TrimSpace(req.SuccessURL),
		FailUrl:        strings.TrimSpace(req.FailURL),
		Network:        &network,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) GetTopUpList(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	limit, err := parseOptionalUint32Query(c, "limit", 50)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid limit")
	}
	offset, err := parseOptionalUint64Query(c, "offset", 0)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid offset")
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.balanceClient.GetTopUpList(ctx, &balanceclient.GetTopUpListRequest{
		UserId: user.ID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) GetTopUp(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	topUpID, err := strconv.ParseInt(c.Params("top_up_id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid top_up_id")
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.balanceClient.GetTopUp(ctx, &balanceclient.GetTopUpRequest{
		UserId:  user.ID,
		TopUpId: topUpID,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) CreateWithdrawal(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	req := models.CreateWithdrawalRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	req.Amount = strings.TrimSpace(req.Amount)
	if req.Amount == "" {
		return fiber.NewError(fiber.StatusBadRequest, "amount is required")
	}
	if req.CurrencyCode == 0 {
		req.CurrencyCode = int64(currency.USDTinTRC)
	}
	if req.DestinationType == 0 {
		req.DestinationType = int32(balancedomain.DestinationType_DESTINATION_TYPE_CRYPTO_WALLET)
	}
	if strings.TrimSpace(req.Destination) == "" {
		return fiber.NewError(fiber.StatusBadRequest, "destination is required")
	}
	if req.IdempotencyKey == "" {
		req.IdempotencyKey = newIdempotencyKey("withdrawal", user.ID)
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.balanceClient.CreateWithdrawal(ctx, &balanceclient.CreateWithdrawalRequest{
		UserId: user.ID,
		Money: &balancedomain.Money{
			Amount:       req.Amount,
			CurrencyCode: req.CurrencyCode,
		},
		DestinationType: balancedomain.DestinationType(req.DestinationType),
		Destination:     strings.TrimSpace(req.Destination),
		IdempotencyKey:  strings.TrimSpace(req.IdempotencyKey),
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) GetWithdrawal(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	withdrawalID, err := strconv.ParseInt(c.Params("withdrawal_id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid withdrawal_id")
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.balanceClient.GetWithdrawal(ctx, &balanceclient.GetWithdrawalRequest{
		UserId:       user.ID,
		WithdrawalId: withdrawalID,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func parseOptionalInt64PtrQuery(c *fiber.Ctx, key string) (*int64, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return nil, nil
	}

	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return nil, err
	}

	return &value, nil
}

func parseOptionalUint32Query(c *fiber.Ctx, key string, defaultValue uint32) (uint32, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return defaultValue, nil
	}

	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return 0, err
	}

	return uint32(value), nil
}

func parseOptionalUint64Query(c *fiber.Ctx, key string, defaultValue uint64) (uint64, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return defaultValue, nil
	}

	return strconv.ParseUint(raw, 10, 64)
}

func newIdempotencyKey(prefix string, userID int64) string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%s-%d-%d", prefix, userID, time.Now().UnixNano())
	}

	return fmt.Sprintf("%s-%d-%d-%s", prefix, userID, time.Now().UnixNano(), hex.EncodeToString(buf[:]))
}
