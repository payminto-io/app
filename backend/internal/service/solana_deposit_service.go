package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"slices"
	"strings"
	"time"

	"github.com/payminto/payminto/backend/internal/blockchain"
	"github.com/payminto/payminto/backend/internal/blockchain/solana"
	"github.com/payminto/payminto/backend/internal/metrics"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/realtime"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// SolanaDepositConfig tunes the watcher; zero values take the defaults below.
type SolanaDepositConfig struct {
	// AccountsPerPoll bounds how many due accounts one PollOnce touches.
	AccountsPerPoll int
	// SignaturePageSize is the getSignaturesForAddress page; more pages follow when a page is full.
	SignaturePageSize int
	// DropGrace is the minimum age before a deposit unknown to the cluster can be failed.
	DropGrace time.Duration
	// ReviveWindow is how long a failed deposit is re-checked for a late finalization.
	ReviveWindow time.Duration
	// FetchNodes is how many pool picks a transaction fetch tries before the signature is held.
	FetchNodes int
	// MaxHeldAttempts is how many ticks a listed-but-unreadable signature holds the cursor before
	// it is recorded as unresolved and tracked on its own.
	MaxHeldAttempts int
	// OwnerCadence is how often the owner address is polled while the payment is open (anomalies only).
	OwnerCadence time.Duration
	// LateCadence is the poll interval after payment expiry until watch_until.
	LateCadence time.Duration
	// LateWindow is how long after payment expiry an account stays watched (watch_until).
	LateWindow time.Duration
	// MaxAnomaliesPerAccountPerDay bounds missed_deposits rows per destination (dust spam).
	MaxAnomaliesPerAccountPerDay int
	// ExpiredScan is how often an expired account's balance is read for late money.
	ExpiredScan time.Duration
	// ExpiredScanFor is how long after watch_until the expired scan keeps reading an account; it bounds
	// the scanned set to the accounts that expired within it instead of every account ever created.
	ExpiredScanFor time.Duration
	// MaxUnresolvedPolls is how many polls unresolved signatures may keep an account from expiring;
	// past it the account expires with an anomaly and the expired scan keeps retrying them.
	MaxUnresolvedPolls int
}

func (c SolanaDepositConfig) withDefaults() SolanaDepositConfig {
	if c.AccountsPerPoll <= 0 {
		c.AccountsPerPoll = 500
	}
	if c.SignaturePageSize <= 0 {
		c.SignaturePageSize = 100
	}
	if c.DropGrace <= 0 {
		c.DropGrace = 3 * time.Minute
	}
	if c.ReviveWindow <= 0 {
		c.ReviveWindow = 7 * 24 * time.Hour
	}
	if c.FetchNodes <= 0 {
		c.FetchNodes = 2
	}
	if c.MaxHeldAttempts <= 0 {
		c.MaxHeldAttempts = 12
	}
	if c.OwnerCadence < 0 {
		c.OwnerCadence = 0
	} else if c.OwnerCadence == 0 {
		c.OwnerCadence = time.Minute
	}
	if c.LateCadence <= 0 {
		c.LateCadence = 10 * time.Minute
	}
	if c.LateWindow <= 0 {
		c.LateWindow = 7 * 24 * time.Hour
	}
	if c.MaxAnomaliesPerAccountPerDay <= 0 {
		c.MaxAnomaliesPerAccountPerDay = 5
	}
	if c.ExpiredScan <= 0 {
		c.ExpiredScan = 24 * time.Hour
	}
	if c.ExpiredScanFor <= 0 {
		c.ExpiredScanFor = 90 * 24 * time.Hour
	}
	if c.MaxUnresolvedPolls <= 0 {
		c.MaxUnresolvedPolls = 36
	}
	return c
}

// Anomaly reasons the watcher writes beyond the parser's kinds.
const (
	anomalyUnexplainedBalance   = "solana_unexplained_balance"
	anomalyLateBalance          = "solana_late_balance"
	anomalyLatePayment          = "late_payment"
	anomalyDropCheckUnavailable = "solana_evidence_unavailable"
	anomalyUnresolvedAtExpiry   = "solana_unresolved_at_expiry"
)

// SolanaDepositService detects SPL deposits by polling signatures per watched account, records
// them through DepositService, finalizes them on the finalized commitment and posts the ledger
// journal in the same transaction when a ledger is wired. Design: .scratch/payments-v1/issues/09-solana-usdc.md.
type SolanaDepositService struct {
	db         *gorm.DB
	client     *solana.Client
	chain      *models.Blockchain
	accounts   repository.SolanaDepositAccountRepository
	deposits   repository.DepositRepository
	depositSvc *DepositService
	missed     repository.MissedDepositRepository
	currencies repository.BlockchainCurrencyRepository
	// ledger may be nil: then the deposit journal is left to the switch (see solana wiring).
	ledger *LedgerService
	broker realtime.Broker
	cfg    SolanaDepositConfig
	now    func() time.Time
}

// NewSolanaDepositService wires the watcher; ledger and broker may be nil.
func NewSolanaDepositService(
	db *gorm.DB,
	client *solana.Client,
	chain *models.Blockchain,
	accounts repository.SolanaDepositAccountRepository,
	deposits repository.DepositRepository,
	depositSvc *DepositService,
	missed repository.MissedDepositRepository,
	currencies repository.BlockchainCurrencyRepository,
	ledger *LedgerService,
	broker realtime.Broker,
	cfg SolanaDepositConfig,
) *SolanaDepositService {
	return &SolanaDepositService{
		db: db, client: client, chain: chain, accounts: accounts, deposits: deposits, depositSvc: depositSvc,
		missed: missed, currencies: currencies, ledger: ledger, broker: broker, cfg: cfg.withDefaults(), now: time.Now,
	}
}

// errTransientFetch marks a listed signature no node returned a transaction for.
var errTransientFetch = errors.New("solana: transaction not yet available from any node")

// PollOnce expires accounts past their window (after a final balance read; held or unresolved
// signatures keep an account watched), scans expired accounts for late money on a slow cadence,
// reads every due token account in getMultipleAccounts batches, and polls signatures only for
// accounts whose balance moved or whose cadence is due. Returns how many deposits were newly
// recorded. Per-account errors are logged and skipped.
func (s *SolanaDepositService) PollOnce(ctx context.Context) (int, error) {
	now := s.now()
	if err := s.expireAccounts(ctx, now); err != nil {
		log.Printf("[solana] expire accounts: %v", err)
	}
	if err := s.scanExpired(ctx, now); err != nil {
		log.Printf("[solana] expired scan: %v", err)
	}
	accounts, err := s.accounts.ListDue(now, s.cfg.AccountsPerPoll)
	if err != nil {
		return 0, fmt.Errorf("list due accounts: %w", err)
	}
	if len(accounts) == 0 {
		return 0, nil
	}
	balances, err := s.tokenBalances(ctx, accounts)
	if err != nil {
		return 0, err
	}
	recorded := 0
	for i := range accounts {
		if ctx.Err() != nil {
			return recorded, ctx.Err()
		}
		acct := &accounts[i]
		n, err := s.pollAccount(ctx, acct, balances[acct.TokenAccount], now)
		if err != nil {
			log.Printf("[solana] account %s poll: %v", acct.TokenAccount, err)
			continue
		}
		recorded += n
	}
	return recorded, nil
}

// expireAccounts reads each account past watch_until once more: a balance with no deposit recorded
// is late money and an anomaly. A held signature keeps the account watched; unresolved ones do so
// only within MaxUnresolvedPolls, then the account expires with an anomaly and they stay tracked.
func (s *SolanaDepositService) expireAccounts(ctx context.Context, now time.Time) error {
	due, err := s.accounts.ListExpiring(now, s.cfg.MaxUnresolvedPolls, 200)
	if err != nil {
		return err
	}
	if len(due) == 0 {
		return nil
	}
	balances, err := s.tokenBalances(ctx, due)
	if err != nil {
		return err
	}
	for i := range due {
		acct := &due[i]
		if unresolved := decodeSignatures(acct.UnresolvedSignatures); len(unresolved) > 0 {
			for _, sig := range unresolved {
				a := solana.Anomaly{Kind: anomalyUnresolvedAtExpiry, Signature: sig, To: acct.TokenAccount, Mint: acct.Mint,
					Detail: fmt.Sprintf("still unresolved after %d polls; account expired, retried on the expired scan", acct.UnresolvedAttempts)}
				if err := s.recordAnomaly(acct, a); err != nil {
					log.Printf("[solana] unresolved-at-expiry anomaly for %s: %v", sig, err)
				}
			}
		}
		bal := balances[acct.TokenAccount]
		s.flagLateBalance(acct, bal)
		if err := s.accounts.Update(acct.ID, map[string]any{"status": models.SolanaDepositAccountExpired, "last_balance_raw": bal, "token_poll_after": now.Add(s.cfg.ExpiredScan)}); err != nil {
			return err
		}
	}
	log.Printf("[solana] %d accounts passed watch_until; now on the expired scan", len(due))
	return nil
}

// scanExpired reads expired accounts' balances on ExpiredScan and flags anything that arrived.
func (s *SolanaDepositService) scanExpired(ctx context.Context, now time.Time) error {
	list, err := s.accounts.ListExpiredDue(now, now.Add(-s.cfg.ExpiredScanFor), 500)
	if err != nil || len(list) == 0 {
		return err
	}
	balances, err := s.tokenBalances(ctx, list)
	if err != nil {
		return err
	}
	for i := range list {
		acct := &list[i]
		updates := map[string]any{"token_poll_after": now.Add(s.cfg.ExpiredScan), "last_polled_at": now}
		if unresolved := decodeSignatures(acct.UnresolvedSignatures); len(unresolved) > 0 {
			updates["unresolved_signatures"] = encodeSignatures(s.retryUnresolved(ctx, acct, unresolved))
		}
		bal := balances[acct.TokenAccount]
		if bal != acct.LastBalanceRaw {
			s.flagLateBalance(acct, bal)
		}
		updates["last_balance_raw"] = bal
		if err := s.accounts.Update(acct.ID, updates); err != nil {
			return err
		}
	}
	return nil
}

// retryUnresolved processes an expired account's unresolved signatures once and returns those still unresolved.
func (s *SolanaDepositService) retryUnresolved(ctx context.Context, acct *models.SolanaDepositAccount, sigs []string) []string {
	watch, err := watchFor(acct)
	if err != nil {
		return sigs
	}
	var still []string
	for _, sig := range sigs {
		if _, err := s.processSignature(ctx, acct, watch, sig); err != nil {
			if !errors.Is(err, errTransientFetch) {
				log.Printf("[solana] expired %s unresolved %s: %v", acct.TokenAccount, sig, err)
			}
			still = append(still, sig)
		}
	}
	return still
}

// flagLateBalance records an anomaly when an account holds more than its recorded deposits.
func (s *SolanaDepositService) flagLateBalance(acct *models.SolanaDepositAccount, raw string) {
	if raw == "" || raw == "0" {
		return
	}
	bal, ok := new(big.Int).SetString(raw, 10)
	if !ok {
		return
	}
	var recorded decimal.Decimal
	s.db.Model(&models.Deposit{}).Where("to_address = ? AND blockchain_currency_id = ? AND status IN ?", acct.TokenAccount, acct.BlockchainCurrencyID,
		[]string{models.DepositStatusPending, models.DepositStatusConfirming, models.DepositStatusConfirmed}).
		Select("COALESCE(SUM(amount), 0)").Scan(&recorded)
	have := decimal.NewFromBigInt(bal, -int32(acct.Decimals))
	if !have.GreaterThan(recorded) {
		return
	}
	a := solana.Anomaly{Kind: anomalyLateBalance, Signature: "late:" + acct.TokenAccount + ":" + raw, To: acct.TokenAccount, Mint: acct.Mint, Amount: have.Sub(recorded),
		Detail: "balance after watch_until exceeds recorded deposits"}
	if err := s.recordAnomaly(acct, a); err != nil {
		log.Printf("[solana] late balance anomaly for %s: %v", acct.TokenAccount, err)
	}
}

// tokenBalances reads the token accounts in batches of 100; a missing account reads as "".
func (s *SolanaDepositService) tokenBalances(ctx context.Context, accounts []models.SolanaDepositAccount) (map[string]string, error) {
	out := make(map[string]string, len(accounts))
	for start := 0; start < len(accounts); start += 100 {
		end := min(start+100, len(accounts))
		addrs := make([]string, 0, end-start)
		for _, a := range accounts[start:end] {
			addrs = append(addrs, a.TokenAccount)
		}
		infos, err := s.client.GetMultipleAccounts(ctx, addrs, solana.CommitmentConfirmed)
		if err != nil {
			return nil, fmt.Errorf("getMultipleAccounts: %w", err)
		}
		for i, addr := range addrs {
			if i < len(infos) {
				if raw, ok := infos[i].TokenAmountRaw(); ok {
					out[addr] = raw
				}
			}
		}
	}
	return out, nil
}

func watchFor(acct *models.SolanaDepositAccount) (solana.Watch, error) {
	owner, err := solana.ParsePublicKey(acct.OwnerAddress)
	if err != nil {
		return solana.Watch{}, err
	}
	ata, err := solana.ParsePublicKey(acct.TokenAccount)
	if err != nil {
		return solana.Watch{}, err
	}
	mint, err := solana.ParsePublicKey(acct.Mint)
	if err != nil {
		return solana.Watch{}, err
	}
	program, err := solana.ParsePublicKey(acct.TokenProgram)
	if err != nil {
		return solana.Watch{}, err
	}
	return solana.Watch{Owner: owner, TokenAccount: ata, Mint: mint, TokenProgram: program, Decimals: acct.Decimals}, nil
}

// pollAccount decides what this tick costs: the token account's signatures when its balance moved
// or a held/unresolved signature is pending, the owner's on its cadence, and schedules the next tick.
func (s *SolanaDepositService) pollAccount(ctx context.Context, acct *models.SolanaDepositAccount, balance string, now time.Time) (int, error) {
	watch, err := watchFor(acct)
	if err != nil {
		return 0, err
	}
	late := acct.PaymentExpiresAt != nil && now.After(*acct.PaymentExpiresAt)
	updates := map[string]any{"last_polled_at": now}
	next := now
	if late {
		next = now.Add(s.cfg.LateCadence)
	}
	updates["token_poll_after"] = next

	recorded := 0
	seen := map[string]bool{}
	var pollErr error

	// Unresolved signatures are retried on their own before anything else.
	unresolved := decodeSignatures(acct.UnresolvedSignatures)
	if len(unresolved) > 0 {
		// Changing the list here is what the held-budget branch below reads back.
		var still []string
		for _, sig := range unresolved {
			n, err := s.processSignature(ctx, acct, watch, sig)
			if errors.Is(err, errTransientFetch) {
				// Tracked on its own: it must not hold the cursor again when the address is listed.
				seen[sig] = true
				still = append(still, sig)
				continue
			}
			if err != nil {
				return recorded, err
			}
			seen[sig] = true
			recorded += n
		}
		updates["unresolved_signatures"] = encodeSignatures(still)
		if len(still) > 0 {
			updates["unresolved_attempts"] = acct.UnresolvedAttempts + 1
		} else {
			updates["unresolved_attempts"] = 0
		}
	}

	balanceMoved := balance != acct.LastBalanceRaw
	ownerDue := acct.OwnerPollAfter == nil || !now.Before(*acct.OwnerPollAfter)
	if late {
		ownerDue = true
	}
	// The new balance is persisted only once the signature poll explained the movement; an
	// unexplained movement keeps re-triggering the poll under its own budget (NEW-C2).
	explained := !balanceMoved
	if balanceMoved || acct.HeldSignature != "" || late || acct.BalanceHoldAttempts > 0 {
		n, cursor, listed, err := s.pollAddress(ctx, acct, watch, acct.TokenAccount, acct.TokenAccountCursor, seen)
		recorded += n
		held, attempts := "", 0
		// Any error other than a hold (transport, rate limit, timeout) says nothing about the held
		// signature: it is carried unchanged and only a completed poll clears it (NEW2-C1).
		failed := false
		var hold *heldError
		if err != nil && !errors.As(err, &hold) {
			held, attempts, failed = acct.HeldSignature, acct.HeldAttempts, true
		}
		if hold != nil {
			held, attempts = hold.signature, 1
			if acct.HeldSignature == hold.signature {
				attempts = acct.HeldAttempts + 1
			}
			if attempts >= s.cfg.MaxHeldAttempts {
				// Budget spent: record it, track it on its own, and let later signatures proceed.
				s.recordUnresolved(acct, hold.signature, attempts)
				current := decodeSignatures(acct.UnresolvedSignatures)
				if v, ok := updates["unresolved_signatures"].(string); ok {
					current = decodeSignatures(v)
				}
				updates["unresolved_signatures"] = encodeSignatures(append(current, hold.signature))
				seen[hold.signature] = true
				var n2 int
				var hold2 *heldError
				n2, cursor, listed, err = s.pollAddress(ctx, acct, watch, acct.TokenAccount, acct.TokenAccountCursor, seen)
				recorded += n2
				held, attempts = "", 0
				if errors.As(err, &hold2) {
					held, attempts = hold2.signature, 1
				} else if err != nil {
					failed = true
				}
			}
		}
		if cursor != "" {
			updates["token_account_cursor"] = cursor
		}
		updates["held_signature"] = held
		updates["held_attempts"] = attempts
		if err != nil {
			pollErr = err
		}
		// Listed signatures or a transient hold explain the movement; nothing listed and no error
		// means the node that reported the balance and the node that listed signatures disagree. A
		// failed poll explains nothing, so a moved balance keeps the next poll alive under its budget.
		explained = explained || (!failed && (listed > 0 || held != ""))
	}
	if explained {
		updates["last_balance_raw"] = balance
		updates["balance_hold_attempts"] = 0
	} else {
		holdAttempts := acct.BalanceHoldAttempts + 1
		updates["balance_hold_attempts"] = holdAttempts
		if holdAttempts >= s.cfg.MaxHeldAttempts {
			s.recordBalanceHold(acct, balance, holdAttempts)
			updates["last_balance_raw"] = balance
			updates["balance_hold_attempts"] = 0
		}
	}
	if ownerDue {
		n, cursor, _, err := s.pollAddress(ctx, acct, watch, acct.OwnerAddress, acct.OwnerCursor, seen)
		recorded += n
		if cursor != "" {
			updates["owner_cursor"] = cursor
		}
		if err != nil && pollErr == nil {
			pollErr = err
		}
		ownerNext := now.Add(s.cfg.OwnerCadence)
		if late {
			ownerNext = next
		}
		updates["owner_poll_after"] = ownerNext
	}
	if err := s.accounts.Update(acct.ID, updates); err != nil {
		return recorded, err
	}
	if pollErr != nil && !errors.Is(pollErr, errTransientFetch) {
		return recorded, pollErr
	}
	return recorded, nil
}

type heldError struct{ signature string }

func (e *heldError) Error() string        { return "solana: holding cursor below " + e.signature }
func (e *heldError) Is(target error) bool { return target == errTransientFetch }

// pollAddress fetches new signatures for one address and processes them oldest first. It returns
// the newest fully processed signature as the cursor (a transient failure holds it below that
// signature) and how many new signatures the node listed.
func (s *SolanaDepositService) pollAddress(ctx context.Context, acct *models.SolanaDepositAccount, watch solana.Watch, address, cursor string, seen map[string]bool) (recorded int, newCursor string, listed int, err error) {
	sigs, err := s.newSignatures(ctx, address, cursor)
	if err != nil {
		return 0, "", 0, err
	}
	if len(sigs) == 0 {
		return 0, "", 0, nil
	}
	newCursor = sigs[0].Signature
	for i := len(sigs) - 1; i >= 0; i-- {
		info := sigs[i]
		if seen[info.Signature] || info.Failed() {
			continue
		}
		seen[info.Signature] = true
		n, err := s.processSignature(ctx, acct, watch, info.Signature)
		if err != nil {
			newCursor = ""
			if i+1 < len(sigs) {
				newCursor = sigs[i+1].Signature
			}
			if errors.Is(err, errTransientFetch) {
				return recorded, newCursor, len(sigs), &heldError{signature: info.Signature}
			}
			log.Printf("[solana] %s: %v", info.Signature, err)
			return recorded, newCursor, len(sigs), err
		}
		recorded += n
	}
	return recorded, newCursor, len(sigs), nil
}

// recordBalanceHold writes the anomaly for a balance movement no signature explained within the budget.
func (s *SolanaDepositService) recordBalanceHold(acct *models.SolanaDepositAccount, balance string, attempts int) {
	a := solana.Anomaly{Kind: anomalyUnexplainedBalance, Signature: "balance:" + acct.TokenAccount + ":" + balance, To: acct.TokenAccount, Mint: acct.Mint,
		Detail: fmt.Sprintf("balance %s -> %s with no signature listed in %d polls", acct.LastBalanceRaw, balance, attempts)}
	if err := s.recordAnomaly(acct, a); err != nil {
		log.Printf("[solana] unexplained balance anomaly for %s: %v", acct.TokenAccount, err)
	}
}

// newSignatures pages getSignaturesForAddress from the tip back to the cursor, newest first.
func (s *SolanaDepositService) newSignatures(ctx context.Context, address, cursor string) ([]solana.SignatureInfo, error) {
	var all []solana.SignatureInfo
	before := ""
	for page := 0; page < 20; page++ {
		sigs, err := s.client.GetSignaturesForAddressPage(ctx, address, cursor, before, s.cfg.SignaturePageSize, solana.CommitmentConfirmed)
		if err != nil {
			return nil, fmt.Errorf("getSignaturesForAddress %s: %w", address, err)
		}
		all = append(all, sigs...)
		if len(sigs) < s.cfg.SignaturePageSize {
			return all, nil
		}
		before = sigs[len(sigs)-1].Signature
	}
	return all, nil
}

func (s *SolanaDepositService) processSignature(ctx context.Context, acct *models.SolanaDepositAccount, watch solana.Watch, signature string) (int, error) {
	// Already recorded signatures are not fetched again (unresolved retries and held cursors re-list them).
	if existing, err := s.deposits.GetByTxIDAndToAddress(signature, acct.TokenAccount, acct.BlockchainCurrencyID); err == nil && existing != nil {
		return 0, nil
	}
	tx, err := s.client.GetTransactionFromNodes(ctx, signature, solana.CommitmentConfirmed, s.cfg.FetchNodes)
	if err != nil {
		return 0, fmt.Errorf("getTransaction: %w", err)
	}
	if tx == nil {
		return 0, errTransientFetch
	}
	credit, anomalies := solana.ExtractDeposits(tx, watch)
	for _, a := range anomalies {
		if err := s.recordAnomaly(acct, a); err != nil {
			log.Printf("[solana] record anomaly %s on %s: %v", a.Kind, a.Signature, err)
		}
	}
	if credit == nil || credit.Amount.IsZero() {
		return 0, nil
	}
	bc, err := s.currencies.GetByID(acct.BlockchainCurrencyID)
	if err != nil {
		return 0, fmt.Errorf("blockchain currency %d: %w", acct.BlockchainCurrencyID, err)
	}
	if bc.MinDepositAmount != nil && !bc.MinDepositAmount.IsZero() && credit.Amount.LessThan(*bc.MinDepositAmount) {
		log.Printf("[solana] dust ignored: %s %s < %s (%s)", credit.Amount, bc.CurrencyCode, bc.MinDepositAmount, signature)
		return 0, nil
	}
	dep, err := s.depositSvc.RecordDeposit(ctx, blockchain.Transaction{
		TxHash: credit.Signature, FromAddress: credit.From, ToAddress: acct.TokenAccount, Amount: credit.Amount,
		Token: credit.Mint, BlockNumber: credit.Slot,
	}, acct.BlockchainCurrencyID)
	if err != nil {
		return 0, fmt.Errorf("record deposit: %w", err)
	}
	if err := s.accounts.Update(acct.ID, map[string]any{"last_seen_slot": gorm.Expr("CASE WHEN last_seen_slot > ? THEN last_seen_slot ELSE ? END", credit.Slot, credit.Slot)}); err != nil {
		log.Printf("[solana] last seen slot for %s: %v", acct.TokenAccount, err)
	}
	log.Printf("[solana] deposit %d seen: %s %s -> %s (%s)", dep.ID, credit.Amount, bc.CurrencyCode, acct.TokenAccount, signature)
	s.publish(dep, models.PaymentStateOpen)
	return 1, nil
}

// recordUnresolved writes the anomaly for a signature no node returned within the budget.
func (s *SolanaDepositService) recordUnresolved(acct *models.SolanaDepositAccount, signature string, attempts int) {
	a := solana.Anomaly{Kind: "solana_unresolved_signature", Signature: signature, To: acct.TokenAccount, Detail: fmt.Sprintf("no node returned it in %d polls", attempts)}
	if err := s.recordAnomaly(acct, a); err != nil {
		log.Printf("[solana] record unresolved %s: %v", signature, err)
	}
}

// recordAnomaly writes a missed_deposits row once per (signature, destination, reason), bounded
// per destination per day so owner-address dust cannot grow the table without limit.
func (s *SolanaDepositService) recordAnomaly(acct *models.SolanaDepositAccount, a solana.Anomaly) error {
	reason := a.Kind
	if a.Detail != "" {
		reason = a.Kind + ": " + a.Detail
	}
	if len(reason) > 100 {
		reason = reason[:100]
	}
	existing, err := s.missed.GetByTxHash(a.Signature)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	for _, m := range existing {
		if m.ToAddress == a.To && m.Reason == reason {
			return nil
		}
	}
	var recent int64
	if err := s.db.Model(&models.MissedDeposit{}).
		Where("blockchain_id = ? AND to_address = ? AND reason LIKE ? AND created_at > ?", s.chain.ID, a.To, a.Kind+"%", s.now().Add(-24*time.Hour)).
		Count(&recent).Error; err != nil {
		return err
	}
	if recent >= int64(s.cfg.MaxAnomaliesPerAccountPerDay) {
		log.Printf("[solana] anomaly %s on %s suppressed: %d rows in 24h (%s)", a.Kind, a.To, recent, a.Signature)
		return nil
	}
	var currencyID *uint
	if a.Mint != "" {
		if rows, err := s.currencies.ListByBlockchainID(s.chain.ID); err == nil {
			for _, row := range rows {
				if row.Address == a.Mint {
					id := row.ID
					currencyID = &id
					break
				}
			}
		}
	}
	log.Printf("[solana] anomaly %s on %s (%s): %s %s -> %s", a.Kind, acct.TokenAccount, a.Signature, a.Amount, a.Mint, a.To)
	return s.missed.Create(&models.MissedDeposit{
		TxHash:               a.Signature,
		BlockchainID:         s.chain.ID,
		BlockchainCurrencyID: currencyID,
		FromAddress:          a.From,
		ToAddress:            a.To,
		Amount:               a.Amount,
		BlockNumber:          int64(a.Slot),
		Reason:               reason,
		Status:               models.MissedDepositStatusPending,
	})
}

// ConfirmOnce advances pending and confirming Solana deposits, and re-checks recently failed ones
// so a late finalization revives them: confirmed commitment is "confirming", finalized is
// "confirmed" (with the ledger journal when wired), an on-chain error or an evidenced drop is "failed".
func (s *SolanaDepositService) ConfirmOnce(ctx context.Context) (int, error) {
	var deposits []models.Deposit
	err := s.db.WithContext(ctx).
		Joins("JOIN blockchain_currencies ON blockchain_currencies.id = deposits.blockchain_currency_id").
		Where("blockchain_currencies.blockchain_id = ? AND (deposits.status IN ? OR (deposits.status = ? AND deposits.created_at > ? AND deposits.updated_at < ?))",
			s.chain.ID, []string{models.DepositStatusPending, models.DepositStatusConfirming}, models.DepositStatusFailed, s.now().Add(-s.cfg.ReviveWindow), s.now().Add(-s.cfg.LateCadence)).
		Order("deposits.updated_at ASC").Limit(200).
		Find(&deposits).Error
	if err != nil {
		return 0, fmt.Errorf("list open solana deposits: %w", err)
	}
	if len(deposits) == 0 {
		return 0, nil
	}
	finalized := 0
	for start := 0; start < len(deposits); start += 100 {
		end := min(start+100, len(deposits))
		batch := deposits[start:end]
		sigs := make([]string, len(batch))
		for i := range batch {
			sigs[i] = batch[i].TxID
		}
		statuses, err := s.client.GetSignatureStatuses(ctx, sigs)
		if err != nil {
			return finalized, fmt.Errorf("getSignatureStatuses: %w", err)
		}
		for i := range batch {
			var st *solana.SignatureStatus
			if i < len(statuses) {
				st = statuses[i]
			}
			done, err := s.advance(ctx, &batch[i], st)
			if err != nil {
				log.Printf("[solana] deposit %d advance: %v", batch[i].ID, err)
				continue
			}
			if done {
				finalized++
			}
		}
	}
	return finalized, nil
}

func (s *SolanaDepositService) advance(ctx context.Context, d *models.Deposit, st *solana.SignatureStatus) (bool, error) {
	switch {
	case st == nil:
		if d.Status == models.DepositStatusFailed || s.now().Sub(d.CreatedAt) < s.cfg.DropGrace {
			// Touch so the next ConfirmOnce round orders it after fresher rows.
			return false, s.db.Model(&models.Deposit{}).Where("id = ?", d.ID).Update("updated_at", s.now()).Error
		}
		dropped, err := s.evidencedDrop(ctx, d)
		if err != nil || !dropped {
			return false, err
		}
		log.Printf("[solana] deposit %d dropped before finalization (%s); reversing", d.ID, d.TxID)
		return false, s.fail(d)
	case st.Failed():
		if d.Status == models.DepositStatusFailed {
			return false, nil
		}
		log.Printf("[solana] deposit %d transaction failed on chain (%s); reversing", d.ID, d.TxID)
		return false, s.fail(d)
	case st.ConfirmationStatus == solana.CommitmentFinalized:
		return s.finalize(ctx, d)
	default:
		confs := 1
		if st.Confirmations != nil && *st.Confirmations > 0 {
			confs = int(*st.Confirmations)
		}
		if confs >= d.RequiredConfirmations {
			confs = d.RequiredConfirmations - 1
		}
		if confs <= 0 {
			return false, nil
		}
		if d.Status == models.DepositStatusFailed {
			// Revived: the cluster knows it again.
			return false, s.db.Model(&models.Deposit{}).Where("id = ? AND status = ?", d.ID, models.DepositStatusFailed).
				Updates(map[string]any{"status": models.DepositStatusConfirming, "confirmations": confs}).Error
		}
		_, err := s.deposits.ConfirmIfPending(d.ID, confs)
		return false, err
	}
}

// evidencedDrop requires the cluster to have finalized past the deposit's slot and the signature
// to be absent at finalized commitment on more than one node; a single lagging node is not a drop.
func (s *SolanaDepositService) evidencedDrop(ctx context.Context, d *models.Deposit) (bool, error) {
	finalizedSlot, err := s.client.GetSlot(ctx, solana.CommitmentFinalized)
	if err != nil {
		return false, err
	}
	if d.BlockNumber <= 0 || finalizedSlot <= uint64(d.BlockNumber) {
		return false, nil
	}
	tx, reached, err := s.client.GetTransactionFromDistinctNodes(ctx, d.TxID, solana.CommitmentFinalized, max(2, s.cfg.FetchNodes))
	if err != nil {
		return false, err
	}
	if tx != nil {
		return false, nil
	}
	if reached < 2 {
		// One endpoint cannot evidence a drop: the deposit stays pending and the anomaly says why.
		acct, _ := s.accounts.GetByTokenAccount(d.ToAddress)
		if acct != nil {
			_ = s.recordAnomaly(acct, solana.Anomaly{Kind: anomalyDropCheckUnavailable, Signature: d.TxID, To: d.ToAddress, Slot: uint64(d.BlockNumber),
				Detail: "one rpc endpoint; a drop cannot be evidenced on two nodes"})
		}
		return false, nil
	}
	return true, nil
}

func (s *SolanaDepositService) fail(d *models.Deposit) error {
	res := s.db.Model(&models.Deposit{}).
		Where("id = ? AND status IN ?", d.ID, []string{models.DepositStatusPending, models.DepositStatusConfirming}).
		Update("status", models.DepositStatusFailed)
	return res.Error
}

// switchOwnsPayment reports a payment opened by the switch's chaindeposit connector: its invoice_id is
// a switch attempt id, and the switch posts that payment's journal on Sync. Every other payment
// (legacy POST /payment) gets its one payment journal from the watcher.
func switchOwnsPayment(tx *gorm.DB, paymentRequestID *uint) (bool, error) {
	if paymentRequestID == nil {
		return false, nil
	}
	var p models.PaymentRequest
	if err := tx.Select("id", "invoice_id").First(&p, *paymentRequestID).Error; err != nil {
		return false, err
	}
	if p.InvoiceID == nil || *p.InvoiceID == "" || !tx.Migrator().HasTable("switch_payment_attempts") {
		return false, nil
	}
	var n int64
	if err := tx.Table("switch_payment_attempts").Where("id = ?", *p.InvoiceID).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// finalize marks the deposit confirmed (from pending, confirming or a revived failed) and, in the same
// transaction, posts the payment journal unless the switch owns the payment; then finalizes the payment.
func (s *SolanaDepositService) finalize(ctx context.Context, d *models.Deposit) (bool, error) {
	won := false
	post := func(tx *gorm.DB) error {
		res := tx.Model(&models.Deposit{}).
			Where("id = ? AND status IN ?", d.ID, []string{models.DepositStatusPending, models.DepositStatusConfirming, models.DepositStatusFailed}).
			Updates(map[string]any{"confirmations": d.RequiredConfirmations, "status": models.DepositStatusConfirmed})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		won = true
		if s.ledger == nil {
			return nil
		}
		owned, err := switchOwnsPayment(tx, d.PaymentRequestID)
		if err != nil || owned {
			return err
		}
		return s.ledger.RecordDepositIn(ctx, tx, d.ID, d.BlockchainCurrencyID, d.Amount)
	}
	var err error
	if s.ledger != nil {
		err = s.ledger.InTransaction(ctx, post)
	} else {
		err = s.db.WithContext(ctx).Transaction(post)
	}
	if err != nil {
		return false, fmt.Errorf("finalize deposit %d: %w", d.ID, err)
	}
	if !won {
		return false, nil
	}
	state := models.PaymentStateFilled
	if d.PaymentRequestID != nil {
		if err := s.depositSvc.FinalizePayment(*d.PaymentRequestID); err != nil {
			log.Printf("[solana] finalize payment %d: %v", *d.PaymentRequestID, err)
		}
		var pr models.PaymentRequest
		if err := s.db.Select("state").First(&pr, *d.PaymentRequestID).Error; err == nil && pr.State != "" {
			state = pr.State
		}
		if state != models.PaymentStateFilled && state != models.PaymentStatePartiallyFilled && state != models.PaymentStateOverFilled {
			// Money for a payment that already closed: the late window exists to catch exactly this.
			if acct, err := s.accounts.GetByTokenAccount(d.ToAddress); err == nil {
				_ = s.recordAnomaly(acct, solana.Anomaly{Kind: anomalyLatePayment, Signature: d.TxID, From: d.FromAddress, To: d.ToAddress, Slot: uint64(d.BlockNumber),
					Amount: d.Amount, Mint: acct.Mint, Detail: "deposit finalized while the payment is " + state})
			}
		}
		metrics.PaymentConfirmed()
	}
	d.Confirmations = d.RequiredConfirmations
	s.publish(d, state)
	log.Printf("[solana] deposit %d finalized (%s)", d.ID, d.TxID)
	return true, nil
}

func (s *SolanaDepositService) publish(d *models.Deposit, state string) {
	if s.broker == nil || d.PaymentRequestID == nil {
		return
	}
	var pr models.PaymentRequest
	if err := s.db.Select("reference_id").First(&pr, *d.PaymentRequestID).Error; err != nil {
		return
	}
	s.broker.Publish(s.broker.PaymentTopic(pr.ReferenceID), realtime.PaymentEvent{
		ReferenceID: pr.ReferenceID, State: strings.ToUpper(state), Confirmations: d.Confirmations,
		Required: d.RequiredConfirmations, TxID: d.TxID,
	})
}

func decodeSignatures(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

func encodeSignatures(sigs []string) string {
	sigs = slices.DeleteFunc(slices.Clone(sigs), func(s string) bool { return s == "" })
	if len(sigs) == 0 {
		return ""
	}
	raw, _ := json.Marshal(sigs)
	return string(raw)
}
