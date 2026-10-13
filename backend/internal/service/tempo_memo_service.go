package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/payminto/payminto/backend/internal/blockchain/tempo"
	"github.com/payminto/payminto/backend/internal/metrics"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/realtime"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TempoReader is the chain surface the Tempo watcher reads; *tempo.Client implements it.
type TempoReader interface {
	ChainID(ctx context.Context) (*big.Int, error)
	Head(ctx context.Context) (uint64, error)
	MemoTransfersTo(ctx context.Context, tokens []common.Address, recipient common.Address, from, to uint64) ([]tempo.MemoTransfer, error)
	ReceiptStatus(ctx context.Context, hash common.Hash) (uint64, error)
}

// TempoMemoConfig tunes the Tempo watcher.
type TempoMemoConfig struct {
	// Merchant is the receiving address every Tempo payment is sent to.
	Merchant common.Address
	// MemberID, when non-zero, restricts credits to payments of this merchant member.
	MemberID uint
	// StartBlock is where scanning begins when the chain cursor is 0; 0 means head - Lookback.
	StartBlock uint64
	// Lookback is how far behind head a fresh cursor starts (default 2000 blocks, about 17 minutes).
	Lookback uint64
	// MaxRange caps one eth_getLogs block range (default 1000).
	MaxRange uint64
}

// Outcome values written to tempo_memo_credits.
const (
	TempoOutcomeCredited         = "credited"
	TempoOutcomeUnknownReference = "unknown_reference"
)

// TempoMemoService credits TIP-20 TransferWithMemo payments on Tempo: it scans logs to the merchant
// address from a block cursor (blockchains.height), verifies each candidate (token, recipient, amount,
// receipt status), resolves the memo to a payment reference, records one deposit per (tx hash, log
// index) and confirms it through finalizeDeposit, the same journal path the Solana watcher uses.
// Tempo finality is deterministic, so a log at or below head is final and is credited at once.
type TempoMemoService struct {
	db         *gorm.DB
	client     TempoReader
	chain      *models.Blockchain
	tokens     map[common.Address]tempo.Token
	depositSvc *DepositService
	ledger     *LedgerService
	broker     realtime.Broker
	cfg        TempoMemoConfig
}

// NewTempoMemoService wires the watcher; ledger and broker may be nil.
func NewTempoMemoService(db *gorm.DB, client TempoReader, chain *models.Blockchain, tokens map[common.Address]tempo.Token,
	depositSvc *DepositService, ledger *LedgerService, broker realtime.Broker, cfg TempoMemoConfig) *TempoMemoService {
	if cfg.Lookback == 0 {
		cfg.Lookback = 2000
	}
	if cfg.MaxRange == 0 {
		cfg.MaxRange = 1000
	}
	return &TempoMemoService{db: db, client: client, chain: chain, tokens: tokens, depositSvc: depositSvc, ledger: ledger, broker: broker, cfg: cfg}
}

// Merchant returns the receiving address.
func (s *TempoMemoService) Merchant() common.Address { return s.cfg.Merchant }

// Tokens returns the accepted tokens.
func (s *TempoMemoService) Tokens() map[common.Address]tempo.Token { return s.tokens }

// WithBroker attaches the checkout event broker during startup, before polling begins.
func (s *TempoMemoService) WithBroker(broker realtime.Broker) *TempoMemoService {
	s.broker = broker
	return s
}

// PollOnce scans from the cursor to head, credits every verified payment and advances the cursor past
// the blocks it fully handled. A transient failure (RPC, DB) stops the scan below the failing log's
// block so the next poll retries it; de-duplication on (tx hash, log index) makes the retry safe.
func (s *TempoMemoService) PollOnce(ctx context.Context) (int, error) {
	// Recheck the RPC on every poll, before retrying pending money writes: endpoints may be
	// reconfigured after boot, and a successful receipt on another chain proves nothing here.
	id, err := s.client.ChainID(ctx)
	if err != nil {
		return 0, fmt.Errorf("tempo chain id: %w", err)
	}
	if id == nil || id.Cmp(big.NewInt(tempo.ModeratoChainID)) != 0 || s.chain.ChainID == nil || *s.chain.ChainID != tempo.ModeratoChainID {
		return 0, fmt.Errorf("tempo: RPC and configured chain must be Moderato %d", tempo.ModeratoChainID)
	}
	if err := s.retryPending(ctx); err != nil {
		log.Printf("[tempo] retry pending: %v", err)
	}
	head, err := s.client.Head(ctx)
	if err != nil {
		return 0, fmt.Errorf("head: %w", err)
	}
	var cursor uint64
	if err := s.db.WithContext(ctx).Model(&models.Blockchain{}).Where("id = ?", s.chain.ID).Pluck("height", &cursor).Error; err != nil {
		return 0, fmt.Errorf("cursor: %w", err)
	}
	if cursor == 0 {
		switch {
		case s.cfg.StartBlock > 0:
			cursor = s.cfg.StartBlock - 1
		case head > s.cfg.Lookback:
			cursor = head - s.cfg.Lookback
		}
	}
	addrs := make([]common.Address, 0, len(s.tokens))
	for a := range s.tokens {
		addrs = append(addrs, a)
	}
	credited := 0
	for from := cursor + 1; from <= head; {
		to := min(from+s.cfg.MaxRange-1, head)
		transfers, err := s.client.MemoTransfersTo(ctx, addrs, s.cfg.Merchant, from, to)
		if err != nil {
			return credited, err
		}
		for _, t := range transfers {
			ok, err := s.handle(ctx, t)
			if err != nil {
				// Hold the cursor below this block so the log is retried.
				if t.BlockNumber > 0 {
					if cursorErr := s.setCursor(ctx, t.BlockNumber-1); cursorErr != nil {
						return credited, errors.Join(err, cursorErr)
					}
				}
				return credited, fmt.Errorf("%s: %w", t.Key(), err)
			}
			if ok {
				credited++
			}
		}
		if err := s.setCursor(ctx, to); err != nil {
			return credited, err
		}
		from = to + 1
	}
	return credited, nil
}

func (s *TempoMemoService) setCursor(ctx context.Context, height uint64) error {
	if err := s.db.WithContext(ctx).Model(&models.Blockchain{}).Where("id = ? AND height < ?", s.chain.ID, height).
		Updates(map[string]any{"height": height, "updated_at": time.Now()}).Error; err != nil {
		return fmt.Errorf("tempo cursor %d: %w", height, err)
	}
	return nil
}

// handle decides one log. It returns true when the log was newly credited.
func (s *TempoMemoService) handle(ctx context.Context, t tempo.MemoTransfer) (bool, error) {
	var n int64
	if err := s.db.WithContext(ctx).Model(&models.TempoMemoCredit{}).
		Where("tx_hash = ? AND log_index = ?", strings.ToLower(t.TxHash.Hex()), t.LogIndex).Count(&n).Error; err != nil {
		return false, err
	}
	if n > 0 {
		return false, nil
	}
	status, err := s.client.ReceiptStatus(ctx, t.TxHash)
	if err != nil {
		return false, err
	}
	tok, amount, rej := tempo.Verify(t, s.tokens, s.cfg.Merchant, status)
	credit := &models.TempoMemoCredit{
		TxHash: strings.ToLower(t.TxHash.Hex()), LogIndex: t.LogIndex, BlockNumber: int64(t.BlockNumber),
		Token: t.Token.Hex(), FromAddress: t.From.Hex(), ToAddress: t.To.Hex(), Amount: auditAmount(t, s.tokens),
		Memo: fmt.Sprintf("0x%x", t.Memo), Outcome: string(rej),
	}
	var payment *models.PaymentRequest
	if rej == tempo.RejectNone {
		payment, err = s.paymentForMemo(ctx, t.Memo, tok, t.To)
		if err != nil {
			return false, err
		}
		if payment == nil {
			credit.Outcome = TempoOutcomeUnknownReference
		} else {
			credit.Outcome = TempoOutcomeCredited
		}
	}
	if payment == nil {
		log.Printf("[tempo] not credited %s: %s (memo %s)", t.Key(), credit.Outcome, credit.Memo)
		return false, s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(credit).Error
	}

	required := max(s.chain.MinConfirmations, 1)
	dep := &models.Deposit{
		TxID: strings.ToLower(t.TxHash.Hex()), Amount: amount, Status: models.DepositStatusPending,
		RequiredConfirmations: required, FromAddress: t.From.Hex(), ToAddress: t.To.Hex(),
		BlockNumber: int64(t.BlockNumber), BlockHash: t.BlockHash.Hex(),
		BlockchainCurrencyID: tok.BlockchainCurrencyID, MemberID: payment.MemberID, PaymentRequestID: &payment.ID,
	}
	inserted := false
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(credit)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil // a concurrent poll recorded this log first
		}
		if err := tx.Create(dep).Error; err != nil {
			return err
		}
		inserted = true
		return tx.Model(&models.TempoMemoCredit{}).Where("id = ?", credit.ID).
			Updates(map[string]any{"deposit_id": dep.ID, "payment_request_id": payment.ID}).Error
	})
	if err != nil || !inserted {
		return false, err
	}
	log.Printf("[tempo] deposit %d seen: %s %s for payment %s (%s)", dep.ID, amount, tok.Code, payment.ReferenceID, tempo.ExplorerTxURL(dep.TxID))
	if _, err := s.confirm(ctx, dep); err != nil {
		// The deposit stays pending; retryPending finishes it next poll.
		log.Printf("[tempo] confirm deposit %d: %v", dep.ID, err)
	}
	return true, nil
}

// paymentForMemo resolves a memo to the payment whose reference it encodes, or nil.
func (s *TempoMemoService) paymentForMemo(ctx context.Context, memo [32]byte, token tempo.Token, recipient common.Address) (*models.PaymentRequest, error) {
	refs := tempo.MemoCandidates(memo)
	if len(refs) == 0 {
		return nil, nil
	}
	// A memo is public invoice data, not proof that this invoice accepted this rail.
	// Require the issued leg with the same owner, receiving address, token, and active chain.
	q := s.db.WithContext(ctx).Where("reference_id IN ?", refs).
		Where(`EXISTS (SELECT 1 FROM deposit_addresses da
			JOIN blockchain_currencies bc ON bc.id = da.blockchain_currency_id
			JOIN blockchains b ON b.id = bc.blockchain_id
			JOIN currencies c ON c.id = bc.currency_id
			WHERE da.payment_request_id = payment_requests.id AND da.member_id = payment_requests.member_id
			AND da.blockchain_currency_id = ? AND LOWER(da.address) = LOWER(?)
			AND bc.blockchain_id = ? AND bc.blockchain_code = ? AND bc.deposit_enabled = ?
			AND LOWER(bc.address) = LOWER(?) AND bc.wallet_precision = ? AND bc.standard = 'TIP20'
			AND b.status = 'active' AND b.chain_id = ? AND c.deposit_enabled = ?
			AND da.deleted_at IS NULL AND bc.deleted_at IS NULL AND b.deleted_at IS NULL AND c.deleted_at IS NULL)`,
			token.BlockchainCurrencyID, recipient.Hex(), s.chain.ID, tempo.ChainCode, true, token.Address.Hex(), token.Decimals, tempo.ModeratoChainID, true)
	if s.cfg.MemberID != 0 {
		q = q.Where("member_id = ?", s.cfg.MemberID)
	}
	var payments []models.PaymentRequest
	if err := q.Limit(2).Find(&payments).Error; err != nil {
		return nil, err
	}
	if len(payments) != 1 {
		return nil, nil // No issued leg or ambiguous compact UUID: never guess an owner.
	}
	return &payments[0], nil
}

// confirm finalizes the deposit (journal in the same transaction) and the payment, then publishes.
func (s *TempoMemoService) confirm(ctx context.Context, d *models.Deposit) (bool, error) {
	var payment models.PaymentRequest
	// An invoice write failure must roll back both confirmation and its journal so retryPending
	// can finish the same deposit on the next poll without losing or double-booking money.
	won, err := finalizeDeposit(ctx, s.db, s.ledger, d, func(tx *gorm.DB) error {
		if d.PaymentRequestID == nil {
			return nil
		}
		_, _, _, err := repository.NewPaymentRepository(tx).FinalizeFromConfirmedDeposits(*d.PaymentRequestID)
		if err != nil {
			return err
		}
		return tx.Select("state", "reference_id").First(&payment, *d.PaymentRequestID).Error
	})
	if err != nil || !won {
		return false, err
	}
	state := "unlinked"
	if d.PaymentRequestID != nil {
		state = payment.State
		if s.broker != nil {
			s.broker.Publish(s.broker.PaymentTopic(payment.ReferenceID), realtime.PaymentEvent{
				ReferenceID: payment.ReferenceID, State: strings.ToUpper(state), Confirmations: d.RequiredConfirmations,
				Required: d.RequiredConfirmations, TxID: d.TxID,
			})
		}
		if state == models.PaymentStateFilled || state == models.PaymentStateOverFilled {
			metrics.PaymentConfirmed()
		}
	}
	log.Printf("[tempo] deposit %d confirmed, payment state %s (%s)", d.ID, state, d.TxID)
	return true, nil
}

// retryPending confirms Tempo deposits a previous poll recorded but could not confirm.
func (s *TempoMemoService) retryPending(ctx context.Context) error {
	ids := make([]uint, 0, len(s.tokens))
	for _, t := range s.tokens {
		ids = append(ids, t.BlockchainCurrencyID)
	}
	var pending []models.Deposit
	if err := s.db.WithContext(ctx).Where("status = ? AND blockchain_currency_id IN ?", models.DepositStatusPending, ids).
		Limit(50).Find(&pending).Error; err != nil {
		return err
	}
	for i := range pending {
		if _, err := s.confirm(ctx, &pending[i]); err != nil {
			return err
		}
	}
	return nil
}

// TempoTokensFromCurrencies builds the accepted-token map from TEMPO blockchain_currencies rows.
func TempoTokensFromCurrencies(rows []models.BlockchainCurrency) map[common.Address]tempo.Token {
	out := make(map[common.Address]tempo.Token, len(rows))
	for _, r := range rows {
		if !r.DepositEnabled || r.BlockchainCode != tempo.ChainCode || r.WalletPrecision != 6 || r.Standard != "TIP20" || !common.IsHexAddress(r.Address) {
			continue
		}
		a := common.HexToAddress(r.Address)
		out[a] = tempo.Token{Address: a, Decimals: int32(r.WalletPrecision), BlockchainCurrencyID: r.ID, Code: r.CurrencyCode}
	}
	return out
}

// auditAmount is the transfer amount in whole units for the audit row, also for rejected logs
// (TIP-20 stablecoins have 6 decimals when the token is not one we accept).
func auditAmount(t tempo.MemoTransfer, tokens map[common.Address]tempo.Token) decimal.Decimal {
	if t.Amount == nil {
		return decimal.Zero
	}
	dec := int32(6)
	if tok, ok := tokens[t.Token]; ok {
		dec = tok.Decimals
	}
	return decimal.NewFromBigInt(t.Amount, -dec)
}
