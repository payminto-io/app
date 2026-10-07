package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/payminto/payminto/backend/internal/blockchain"
	"github.com/payminto/payminto/backend/internal/blockchain/solana"
	"github.com/payminto/payminto/backend/internal/metrics"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/realtime"
	"github.com/payminto/payminto/backend/internal/repository"
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
	return c
}

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

// PollOnce expires accounts past their window, reads every due token account in getMultipleAccounts
// batches, and polls signatures only for accounts whose balance moved or whose cadence is due.
// Returns how many deposits were newly recorded. Per-account errors are logged and skipped.
func (s *SolanaDepositService) PollOnce(ctx context.Context) (int, error) {
	now := s.now()
	if n, err := s.accounts.ExpireWatching(now); err != nil {
		return 0, fmt.Errorf("expire watched accounts: %w", err)
	} else if n > 0 {
		log.Printf("[solana] %d accounts passed watch_until and are no longer polled", n)
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
		var still []string
		for _, sig := range unresolved {
			n, err := s.processSignature(ctx, acct, watch, sig)
			if errors.Is(err, errTransientFetch) {
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
	}

	balanceMoved := balance != acct.LastBalanceRaw
	updates["last_balance_raw"] = balance
	ownerDue := acct.OwnerPollAfter == nil || !now.Before(*acct.OwnerPollAfter)
	if late {
		ownerDue = true
	}
	if balanceMoved || acct.HeldSignature != "" || late {
		n, cursor, err := s.pollAddress(ctx, acct, watch, acct.TokenAccount, acct.TokenAccountCursor, seen)
		recorded += n
		held, attempts := "", 0
		var hold *heldError
		if errors.As(err, &hold) {
			held, attempts = hold.signature, 1
			if acct.HeldSignature == hold.signature {
				attempts = acct.HeldAttempts + 1
			}
			if attempts >= s.cfg.MaxHeldAttempts {
				// Budget spent: record it, track it on its own, and let later signatures proceed.
				s.recordUnresolved(acct, hold.signature, attempts)
				updates["unresolved_signatures"] = encodeSignatures(append(decodeSignatures(fmt.Sprint(updates["unresolved_signatures"])), hold.signature))
				seen[hold.signature] = true
				n, cursor, err = s.pollAddress(ctx, acct, watch, acct.TokenAccount, acct.TokenAccountCursor, seen)
				recorded += n
				held, attempts = "", 0
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
	}
	if ownerDue {
		n, cursor, err := s.pollAddress(ctx, acct, watch, acct.OwnerAddress, acct.OwnerCursor, seen)
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

// pollAddress fetches new signatures for one address and processes them oldest first. The returned
// cursor is the newest fully processed signature; a transient failure holds it below that signature.
func (s *SolanaDepositService) pollAddress(ctx context.Context, acct *models.SolanaDepositAccount, watch solana.Watch, address, cursor string, seen map[string]bool) (int, string, error) {
	sigs, err := s.newSignatures(ctx, address, cursor)
	if err != nil {
		return 0, "", err
	}
	if len(sigs) == 0 {
		return 0, "", nil
	}
	recorded := 0
	newCursor := sigs[0].Signature
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
				return recorded, newCursor, &heldError{signature: info.Signature}
			}
			log.Printf("[solana] %s: %v", info.Signature, err)
			return recorded, newCursor, err
		}
		recorded += n
	}
	return recorded, newCursor, nil
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
		Where("blockchain_currencies.blockchain_id = ? AND (deposits.status IN ? OR (deposits.status = ? AND deposits.created_at > ?))",
			s.chain.ID, []string{models.DepositStatusPending, models.DepositStatusConfirming}, models.DepositStatusFailed, s.now().Add(-s.cfg.ReviveWindow)).
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
	tx, err := s.client.GetTransactionFromNodes(ctx, d.TxID, solana.CommitmentFinalized, max(2, s.cfg.FetchNodes))
	if err != nil {
		return false, err
	}
	return tx == nil, nil
}

func (s *SolanaDepositService) fail(d *models.Deposit) error {
	res := s.db.Model(&models.Deposit{}).
		Where("id = ? AND status IN ?", d.ID, []string{models.DepositStatusPending, models.DepositStatusConfirming}).
		Update("status", models.DepositStatusFailed)
	return res.Error
}

// finalize marks the deposit confirmed (from pending, confirming or a revived failed) and posts its
// journal atomically when a ledger is wired, then finalizes the payment.
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
