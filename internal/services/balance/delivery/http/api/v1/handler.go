package v1

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	stdhttp "net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	balanceadmin "github.com/martketplace-vkr/balance/pkg/api/grpc/v1/admin"
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
	mockRubSBPProvider       = "MOCK_RUB_SBP"
	mockRubCardProvider      = "MOCK_RUB_CARD"
)

type Handler struct {
	balanceClient balanceclient.BalanceClientServiceClient
	balanceAdmin  balanceadmin.BalanceAdminServiceClient
	timeout       time.Duration
	providerURL   string
	webhookSecret string
	httpClient    *stdhttp.Client
}

func New(balanceClient balanceclient.BalanceClientServiceClient, balanceAdmin balanceadmin.BalanceAdminServiceClient, timeout time.Duration, providerURL string, webhookSecret string) *Handler {
	providerURL = strings.TrimRight(strings.TrimSpace(providerURL), "/")
	if providerURL == "" {
		providerURL = "http://mock-payment-provider:8010"
	}

	return &Handler{
		balanceClient: balanceClient,
		balanceAdmin:  balanceAdmin,
		timeout:       timeout,
		providerURL:   providerURL,
		webhookSecret: strings.TrimSpace(webhookSecret),
		httpClient:    &stdhttp.Client{Timeout: timeout},
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

func (h *Handler) GetDepositAddressList(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.balanceClient.GetDepositAddressList(ctx, &balanceclient.GetDepositAddressListRequest{
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

func (h *Handler) CreateRubTopUp(c *fiber.Ctx) error {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		return err
	}

	req := models.CreateRubTopUpRequest{}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	amount := strings.TrimSpace(req.Amount)
	if amount == "" {
		return fiber.NewError(fiber.StatusBadRequest, "amount is required")
	}

	providerName, err := rubProviderName(req.Method)
	if err != nil {
		return err
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.balanceClient.CreateTopUp(ctx, &balanceclient.CreateTopUpRequest{
		UserId: user.ID,
		Money: &balancedomain.Money{
			Amount:       amount,
			CurrencyCode: int64(currency.RUB),
		},
		ProviderType:   balancedomain.ProviderType_PROVIDER_TYPE_ACQUIRING,
		ProviderName:   providerName,
		IdempotencyKey: newIdempotencyKey("rub-topup", user.ID),
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
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

func (h *Handler) ListAdminTopUps(c *fiber.Ctx) error {
	limit, err := parseOptionalUint32Query(c, "limit", 50)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid limit")
	}
	offset, err := parseOptionalUint64Query(c, "offset", 0)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid offset")
	}

	req := &balanceadmin.ListTopUpsRequest{
		Limit:  limit,
		Offset: offset,
	}
	if currencyCode, err := parseOptionalInt64PtrQuery(c, "currency_code"); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid currency_code")
	} else if currencyCode != nil {
		req.CurrencyCode = currencyCode
	}
	if providerType, ok, err := parseOptionalProviderTypeQuery(c, "provider_type"); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid provider_type")
	} else if ok {
		req.ProviderType = &providerType
	}
	if statusValue, ok, err := parseOptionalTopUpStatusQuery(c, "status"); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid status")
	} else if ok {
		req.Status = &statusValue
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.balanceAdmin.ListTopUps(ctx, req)
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	return httpx.WriteProtoJSON(c, resp)
}

func (h *Handler) ConfirmAdminTopUp(c *fiber.Ctx) error {
	externalID := strings.TrimSpace(c.Params("external_id"))
	if externalID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "external_id is required")
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	currencyCode := int64(currency.RUB)
	providerType := balancedomain.ProviderType_PROVIDER_TYPE_ACQUIRING
	statusValue := balancedomain.TopUpStatus_TOP_UP_STATUS_PENDING
	topUpsResp, err := h.balanceAdmin.ListTopUps(ctx, &balanceadmin.ListTopUpsRequest{
		CurrencyCode: &currencyCode,
		ProviderType: &providerType,
		Status:       &statusValue,
		Limit:        1000,
	})
	if err != nil {
		return httpx.MapGRPCError(err)
	}

	var topUpAmount string
	var providerName string
	for _, topUp := range topUpsResp.GetTopUps() {
		if strings.TrimSpace(topUp.GetExternalId()) == externalID {
			topUpAmount = topUp.GetMoney().GetAmount()
			providerName = topUp.GetProviderName()
			break
		}
	}
	if topUpAmount == "" || providerName == "" {
		return fiber.NewError(fiber.StatusNotFound, "top up not found")
	}

	body, err := json.Marshal(map[string]any{
		"amount":        topUpAmount,
		"currency_code": int64(currency.RUB),
		"provider_name": providerName,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	providerReq, err := stdhttp.NewRequestWithContext(c.UserContext(), stdhttp.MethodPost, h.providerURL+"/admin/top-ups/"+externalID+"/confirm", bytes.NewReader(body))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	providerReq.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(providerReq)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fiber.NewError(fiber.StatusBadGateway, "mock provider confirmation failed")
	}

	return c.SendStatus(fiber.StatusAccepted)
}

type mockProviderWebhook struct {
	ExternalID     string `json:"external_id"`
	ProviderName   string `json:"provider_name"`
	Status         string `json:"status"`
	Amount         string `json:"amount"`
	CurrencyCode   int64  `json:"currency_code"`
	WebhookEventID string `json:"webhook_event_id"`
	OccurredAt     string `json:"occurred_at"`
}

func (h *Handler) HandleMockProviderWebhook(c *fiber.Ctx) error {
	body := c.Body()
	if !h.validMockProviderSignature(body, c.Get("X-Mock-Pay-Signature")) {
		return fiber.ErrUnauthorized
	}

	var payload mockProviderWebhook
	if err := json.Unmarshal(body, &payload); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if strings.TrimSpace(payload.Status) != "confirmed" {
		return fiber.NewError(fiber.StatusBadRequest, "unsupported webhook status")
	}

	ctx, cancel := httpx.RPCContext(c, h.timeout)
	defer cancel()

	resp, err := h.balanceAdmin.ConfirmTopUp(ctx, &balanceadmin.ConfirmTopUpRequest{
		ExternalId:     strings.TrimSpace(payload.ExternalID),
		ProviderName:   strings.TrimSpace(payload.ProviderName),
		WebhookEventId: strings.TrimSpace(payload.WebhookEventID),
		Money: &balancedomain.Money{
			Amount:       strings.TrimSpace(payload.Amount),
			CurrencyCode: payload.CurrencyCode,
		},
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

func parseOptionalProviderTypeQuery(c *fiber.Ctx, key string) (balancedomain.ProviderType, bool, error) {
	raw := strings.TrimSpace(strings.ToLower(c.Query(key)))
	if raw == "" {
		return 0, false, nil
	}
	switch raw {
	case "1", "acquiring":
		return balancedomain.ProviderType_PROVIDER_TYPE_ACQUIRING, true, nil
	case "2", "crypto":
		return balancedomain.ProviderType_PROVIDER_TYPE_CRYPTO, true, nil
	default:
		return 0, false, fmt.Errorf("unknown provider_type")
	}
}

func parseOptionalTopUpStatusQuery(c *fiber.Ctx, key string) (balancedomain.TopUpStatus, bool, error) {
	raw := strings.TrimSpace(strings.ToLower(c.Query(key)))
	if raw == "" {
		return 0, false, nil
	}
	switch raw {
	case "1", "created":
		return balancedomain.TopUpStatus_TOP_UP_STATUS_CREATED, true, nil
	case "2", "pending":
		return balancedomain.TopUpStatus_TOP_UP_STATUS_PENDING, true, nil
	case "3", "paid":
		return balancedomain.TopUpStatus_TOP_UP_STATUS_PAID, true, nil
	case "4", "confirmed":
		return balancedomain.TopUpStatus_TOP_UP_STATUS_CONFIRMED, true, nil
	case "5", "failed":
		return balancedomain.TopUpStatus_TOP_UP_STATUS_FAILED, true, nil
	case "6", "canceled", "cancelled":
		return balancedomain.TopUpStatus_TOP_UP_STATUS_CANCELED, true, nil
	case "7", "expired":
		return balancedomain.TopUpStatus_TOP_UP_STATUS_EXPIRED, true, nil
	default:
		return 0, false, fmt.Errorf("unknown status")
	}
}

func rubProviderName(method string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(method)) {
	case "", "sbp", "сбп":
		return mockRubSBPProvider, nil
	case "card", "карта":
		return mockRubCardProvider, nil
	default:
		return "", fiber.NewError(fiber.StatusBadRequest, "method must be sbp or card")
	}
}

func (h *Handler) validMockProviderSignature(body []byte, signature string) bool {
	secret := strings.TrimSpace(h.webhookSecret)
	if secret == "" || strings.TrimSpace(signature) == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(strings.TrimSpace(signature)))
}

func newIdempotencyKey(prefix string, userID int64) string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%s-%d-%d", prefix, userID, time.Now().UnixNano())
	}

	return fmt.Sprintf("%s-%d-%d-%s", prefix, userID, time.Now().UnixNano(), hex.EncodeToString(buf[:]))
}
