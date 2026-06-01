package v1

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/gofiber/contrib/websocket"
	balanceclient "github.com/martketplace-vkr/balance/pkg/api/grpc/v1/client"
	balancedomain "github.com/martketplace-vkr/balance/pkg/api/grpc/v1/domain"
	pkghttp "github.com/martketplace-vkr/pkg/server/http"
	"github.com/martketplace-vkr/pkg/utils/currency"
)

const (
	notificationPollInterval = 5 * time.Second
	notificationPingInterval = 25 * time.Second
)

type wsMessage struct {
	Type string `json:"type"`
}

type cryptoTopUpNotification struct {
	Type         string `json:"type"`
	ID           string `json:"id"`
	Amount       string `json:"amount"`
	CurrencyCode int64  `json:"currency_code"`
	Asset        string `json:"asset"`
	Network      string `json:"network"`
	TxHash       string `json:"tx_hash"`
	TopUpID      int64  `json:"top_up_id"`
}

func (h *Handler) HandleNotificationsWS(conn *websocket.Conn) {
	user, ok := conn.Locals(pkghttp.UserLocalsKey).(*pkghttp.User)
	if !ok || user == nil || user.ID <= 0 {
		_ = conn.WriteJSON(wsMessage{Type: "unauthorized"})
		_ = conn.Close()
		return
	}

	seen, err := h.confirmedCryptoTopUpIDs(user.ID)
	if err != nil {
		_ = conn.WriteJSON(wsMessage{Type: "error"})
		_ = conn.Close()
		return
	}
	lastUSDTBalance, err := h.currentUSDTAvailableBalance(user.ID)
	if err != nil {
		_ = conn.WriteJSON(wsMessage{Type: "error"})
		_ = conn.Close()
		return
	}

	if err := conn.WriteJSON(wsMessage{Type: "ready"}); err != nil {
		return
	}

	closed := make(chan struct{})
	go func() {
		defer close(closed)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	pollTicker := time.NewTicker(notificationPollInterval)
	defer pollTicker.Stop()
	pingTicker := time.NewTicker(notificationPingInterval)
	defer pingTicker.Stop()

	for {
		select {
		case <-closed:
			return
		case <-pollTicker.C:
			nextUSDTBalance, err := h.writeNewCryptoTopUpNotifications(conn, user.ID, seen, lastUSDTBalance)
			if err != nil {
				return
			}
			lastUSDTBalance = nextUSDTBalance
		case <-pingTicker.C:
			if err := conn.WriteJSON(wsMessage{Type: "ping"}); err != nil {
				return
			}
		}
	}
}

func (h *Handler) writeNewCryptoTopUpNotifications(conn *websocket.Conn, userID int64, seen map[string]struct{}, lastUSDTBalance *big.Rat) (*big.Rat, error) {
	sentSpecificNotification := false

	topUps, err := h.listClientTopUps(userID)
	if err != nil {
		return lastUSDTBalance, err
	}

	for index := len(topUps) - 1; index >= 0; index-- {
		topUp := topUps[index]
		if !isConfirmedCryptoTopUp(topUp) {
			continue
		}

		id := notificationTopUpID(topUp)
		if _, ok := seen[id]; ok {
			continue
		}

		seen[id] = struct{}{}
		if err := conn.WriteJSON(cryptoTopUpNotification{
			Type:         "crypto_top_up_confirmed",
			ID:           id,
			Amount:       topUp.GetMoney().GetAmount(),
			CurrencyCode: topUp.GetMoney().GetCurrencyCode(),
			Asset:        "USDT",
			Network:      "TRC-20",
			TxHash:       topUp.GetTxHash(),
			TopUpID:      topUp.GetId(),
		}); err != nil {
			return lastUSDTBalance, err
		}
		sentSpecificNotification = true
	}

	transactions, err := h.listClientUSDTTransactions(userID)
	if err != nil {
		return lastUSDTBalance, err
	}

	for index := len(transactions) - 1; index >= 0; index-- {
		transaction := transactions[index]
		if !isConfirmedCryptoTopUpTransaction(transaction) {
			continue
		}

		id := notificationTransactionID(transaction)
		if _, ok := seen[id]; ok {
			continue
		}

		amount := cryptoTopUpTransactionAmount(transaction)
		if amount == "" {
			continue
		}

		seen[id] = struct{}{}
		if err := conn.WriteJSON(cryptoTopUpNotification{
			Type:         "crypto_top_up_confirmed",
			ID:           id,
			Amount:       amount,
			CurrencyCode: int64(currency.USDTinTRC),
			Asset:        "USDT",
			Network:      "TRC-20",
			TxHash:       transactionTxHash(transaction),
		}); err != nil {
			return lastUSDTBalance, err
		}
		sentSpecificNotification = true
	}

	nextUSDTBalance, err := h.currentUSDTAvailableBalance(userID)
	if err != nil {
		return lastUSDTBalance, err
	}

	delta := new(big.Rat).Sub(nextUSDTBalance, lastUSDTBalance)
	if !sentSpecificNotification && delta.Sign() > 0 {
		id := fmt.Sprintf("balance-delta:%d:%s", userID, time.Now().UTC().Format(time.RFC3339Nano))
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			if err := conn.WriteJSON(cryptoTopUpNotification{
				Type:         "crypto_top_up_confirmed",
				ID:           id,
				Amount:       formatRatAmount(delta, 8),
				CurrencyCode: int64(currency.USDTinTRC),
				Asset:        "USDT",
				Network:      "TRC-20",
			}); err != nil {
				return lastUSDTBalance, err
			}
		}
	}

	return nextUSDTBalance, nil
}

func (h *Handler) confirmedCryptoTopUpIDs(userID int64) (map[string]struct{}, error) {
	topUps, err := h.listClientTopUps(userID)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(topUps))
	for _, topUp := range topUps {
		if isConfirmedCryptoTopUp(topUp) {
			seen[notificationTopUpID(topUp)] = struct{}{}
		}
	}

	transactions, err := h.listClientUSDTTransactions(userID)
	if err != nil {
		return nil, err
	}
	for _, transaction := range transactions {
		if isConfirmedCryptoTopUpTransaction(transaction) {
			seen[notificationTransactionID(transaction)] = struct{}{}
		}
	}

	return seen, nil
}

func (h *Handler) listClientTopUps(userID int64) ([]*balancedomain.TopUp, error) {
	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	resp, err := h.balanceClient.GetTopUpList(ctx, &balanceclient.GetTopUpListRequest{
		UserId: userID,
		Limit:  50,
		Offset: 0,
	})
	if err != nil {
		return nil, err
	}

	return resp.GetTopUps(), nil
}

func (h *Handler) currentUSDTAvailableBalance(userID int64) (*big.Rat, error) {
	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	resp, err := h.balanceClient.GetWallet(ctx, &balanceclient.GetWalletRequest{UserId: userID})
	if err != nil {
		return nil, err
	}

	for _, account := range resp.GetWallet().GetAccounts() {
		if account.GetCurrencyCode() == int64(currency.USDTinTRC) &&
			account.GetAccountType() == balancedomain.AccountType_ACCOUNT_TYPE_AVAILABLE {
			return parseRatAmount(account.GetBalance()), nil
		}
	}

	return new(big.Rat), nil
}

func (h *Handler) listClientUSDTTransactions(userID int64) ([]*balancedomain.LedgerTransaction, error) {
	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	currencyCode := int64(currency.USDTinTRC)
	resp, err := h.balanceClient.GetWalletTransactions(ctx, &balanceclient.GetWalletTransactionsRequest{
		UserId:       userID,
		CurrencyCode: &currencyCode,
		Limit:        50,
		Offset:       0,
	})
	if err != nil {
		return nil, err
	}

	return resp.GetTransactions(), nil
}

func isConfirmedCryptoTopUp(topUp *balancedomain.TopUp) bool {
	return topUp != nil &&
		topUp.GetStatus() == balancedomain.TopUpStatus_TOP_UP_STATUS_CONFIRMED &&
		topUp.GetProviderType() == balancedomain.ProviderType_PROVIDER_TYPE_CRYPTO &&
		topUp.GetMoney() != nil &&
		topUp.GetMoney().GetCurrencyCode() == int64(currency.USDTinTRC)
}

func notificationTopUpID(topUp *balancedomain.TopUp) string {
	if topUp.GetIdempotencyKey() != "" {
		return topUp.GetIdempotencyKey()
	}
	if topUp.GetTxHash() != "" {
		return fmt.Sprintf("%s:%d", topUp.GetTxHash(), topUp.GetId())
	}

	return fmt.Sprintf("top-up:%d", topUp.GetId())
}

func isConfirmedCryptoTopUpTransaction(transaction *balancedomain.LedgerTransaction) bool {
	return transaction != nil &&
		transaction.GetStatus() == balancedomain.LedgerTransactionStatus_LEDGER_TRANSACTION_STATUS_POSTED &&
		isCryptoTopUpLikeTransaction(transaction) &&
		cryptoTopUpTransactionAmount(transaction) != ""
}

func isCryptoTopUpLikeTransaction(transaction *balancedomain.LedgerTransaction) bool {
	if transaction.GetType() == balancedomain.LedgerTransactionType_LEDGER_TRANSACTION_TYPE_TOP_UP ||
		transaction.GetReferenceType() == balancedomain.ReferenceType_REFERENCE_TYPE_TOP_UP {
		return true
	}

	if transaction.GetType() != balancedomain.LedgerTransactionType_LEDGER_TRANSACTION_TYPE_ADJUSTMENT {
		return false
	}

	reason := strings.ToLower(transaction.GetReason())
	idempotencyKey := strings.ToLower(transaction.GetIdempotencyKey())
	referenceID := strings.ToLower(transaction.GetReferenceId())

	return strings.Contains(reason, "deposit") ||
		strings.Contains(reason, "top") ||
		strings.Contains(reason, "пополн") ||
		strings.HasPrefix(idempotencyKey, "crypto-deposit:") ||
		strings.Contains(referenceID, "crypto-deposit")
}

func cryptoTopUpTransactionAmount(transaction *balancedomain.LedgerTransaction) string {
	for _, entry := range transaction.GetEntries() {
		if entry.GetDirection() == balancedomain.EntryDirection_ENTRY_DIRECTION_CREDIT &&
			entry.GetMoney() != nil &&
			entry.GetMoney().GetCurrencyCode() == int64(currency.USDTinTRC) {
			return entry.GetMoney().GetAmount()
		}
	}

	return ""
}

func notificationTransactionID(transaction *balancedomain.LedgerTransaction) string {
	if transaction.GetIdempotencyKey() != "" {
		return transaction.GetIdempotencyKey()
	}

	return fmt.Sprintf("transaction:%d", transaction.GetId())
}

func transactionTxHash(transaction *balancedomain.LedgerTransaction) string {
	key := transaction.GetIdempotencyKey()
	if strings.HasPrefix(key, "crypto-deposit:") {
		parts := strings.Split(key, ":")
		if len(parts) >= 3 {
			return parts[2]
		}
	}

	referenceID := transaction.GetReferenceId()
	if hash, _, ok := strings.Cut(referenceID, ":"); ok {
		return hash
	}

	return referenceID
}

func parseRatAmount(value string) *big.Rat {
	amount, ok := new(big.Rat).SetString(strings.TrimSpace(value))
	if !ok {
		return new(big.Rat)
	}

	return amount
}

func formatRatAmount(value *big.Rat, scale int) string {
	if value == nil {
		return "0"
	}

	return value.FloatString(scale)
}
