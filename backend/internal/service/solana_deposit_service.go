package service

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	// AccountsPerPoll bounds how many watched accounts one PollOnce touches.
	AccountsPerPoll int
	// SignaturePageSize is the getSignaturesForAddress page; more pages follow when a page is full.
	SignaturePageSize int
	// DropGrace is how long a pending deposit may be unknown to the cluster before it is marked failed.
	DropGrace time.Duration
}

func (c SolanaDepositConfig) withDefaults() SolanaDepositConfig {
	if c.AccountsPerPoll <= 0 {
		c.AccountsPerPoll = 200
	}
	if c.SignaturePageSize <= 0 {
		c.SignaturePageSize = 100
	}
	if c.DropGrace <= 0 {
		c.DropGrace = 3 * time.Minute
	}
	return c
}

// SolanaDepositService detects SPL deposits by polling signatures per watched account, records
// them through DepositService, finalizes them on the finalized commitment and posts the ledger
// journal in the same transaction. Design: .scratch/payments-v1/issues/09-solana-usdc.md.
type SolanaDepositService struct {
	db         *gorm.DB
	client     *solana.Client
	chain      *models.Blockchain
	accounts   repository.SolanaDepositAccountRepository
	deposits   repository.DepositRepository
	depositSvc *DepositService
	missed     repository.MissedDepositRepository
	currencies repository.BlockchainCurrencyRepository
	ledger     *LedgerService
	broker     realtime.Broker
	cfg        SolanaDepositConfig
	now        func() time.Time
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

// PollOnce fetches new signatures for every watched account and records credits and anomalies.
// Returns how many deposits were newly recorded. Per-account errors are logged and skipped.
func (s *SolanaDepositService) PollOnce(ctx context.Context) (int, error) {
	accounts, err := s.accounts.ListByStatus(models.SolanaDepositAccountWatching, s.cfg.AccountsPerPoll)
	if err != nil {
		return 0, fmt.Errorf("list watched accounts: %w", err)
	}
	recorded := 0
	for i := range accounts {
		if ctx.Err() != nil {
			return recorded, ctx.Err()
		}
		n, err := s.pollAccount(ctx, &accounts[i])
		if err != nil {
			log.Printf("[solana] account %s poll: %v", accounts[i].TokenAccount, err)
			continue
		}
		recorded += n
	}
	return recorded, nil
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

func (s *SolanaDepositService) pollAccount(ctx context.Context, acct *models.SolanaDepositAccount) (int, error) {
	watch, err := watchFor(acct)
	if err != nil {
		return 0, err
	}
	seen := map[string]bool{}
	recorded := 0
	var newATACursor, newOwnerCursor string
	var maxSlot int64
	for _, target := range []struct {
		address string
		cursor  string
		dst     *string
	}{
		{acct.TokenAccount, acct.TokenAccountCursor, &newATACursor},
		{acct.OwnerAddress, acct.OwnerCursor, &newOwnerCursor},
	} {
		sigs, err := s.newSignatures(ctx, target.address, target.cursor)
		if err != nil {
			return recorded, err
		}
		if len(sigs) == 0 {
			continue
		}
		*target.dst = sigs[0].Signature
		// Oldest first so cursors never skip a signature when a later one fails.
		for i := len(sigs) - 1; i >= 0; i-- {
			info := sigs[i]
			if seen[info.Signature] || info.Failed() {
				continue
			}
			seen[info.Signature] = true
			n, err := s.processSignature(ctx, acct, watch, info.Signature)
			if err != nil {
				// Stop at the failure so the cursor does not move past it.
				*target.dst = ""
				if i+1 < len(sigs) {
					*target.dst = sigs[i+1].Signature
				}
				log.Printf("[solana] %s: %v", info.Signature, err)
				break
			}
			recorded += n
			if int64(info.Slot) > maxSlot {
				maxSlot = int64(info.Slot)
			}
		}
	}
	return recorded, s.accounts.UpdateCursors(acct.ID, newATACursor, newOwnerCursor, maxSlot, s.now())
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
	tx, err := s.client.GetTransaction(ctx, signature, solana.CommitmentConfirmed)
	if err != nil {
		return 0, fmt.Errorf("getTransaction: %w", err)
	}
	if tx == nil {
		return 0, nil
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
	before, _ := s.deposits.GetByTxIDAndToAddress(signature, acct.TokenAccount, acct.BlockchainCurrencyID)
	dep, err := s.depositSvc.RecordDeposit(ctx, blockchain.Transaction{
		TxHash: credit.Signature, FromAddress: credit.From, ToAddress: acct.TokenAccount, Amount: credit.Amount,
		Token: credit.Mint, BlockNumber: credit.Slot,
	}, acct.BlockchainCurrencyID)
	if err != nil {
		return 0, fmt.Errorf("record deposit: %w", err)
	}
	if before != nil {
		return 0, nil
	}
	log.Printf("[solana] deposit %d seen: %s %s -> %s (%s)", dep.ID, credit.Amount, bc.CurrencyCode, acct.TokenAccount, signature)
	s.publish(dep, models.PaymentStateOpen)
	return 1, nil
}

// recordAnomaly writes a missed_deposits row once per (signature, destination, reason).
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

// ConfirmOnce advances pending and confirming Solana deposits: confirmed commitment is "confirming",
// finalized is "confirmed" (with the ledger journal), an error or a drop is "failed".
func (s *SolanaDepositService) ConfirmOnce(ctx context.Context) (int, error) {
	var deposits []models.Deposit
	err := s.db.WithContext(ctx).
		Joins("JOIN blockchain_currencies ON blockchain_currencies.id = deposits.blockchain_currency_id").
		Where("deposits.status IN ? AND blockchain_currencies.blockchain_id = ?", []string{models.DepositStatusPending, models.DepositStatusConfirming}, s.chain.ID).
		Order("deposits.id ASC").Limit(200).
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
		if s.now().Sub(d.CreatedAt) < s.cfg.DropGrace {
			return false, nil
		}
		tx, err := s.client.GetTransaction(ctx, d.TxID, solana.CommitmentConfirmed)
		if err != nil {
			return false, err
		}
		if tx != nil {
			return false, nil
		}
		log.Printf("[solana] deposit %d dropped before finalization (%s); reversing", d.ID, d.TxID)
		return false, s.fail(d)
	case st.Failed():
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
		_, err := s.deposits.ConfirmIfPending(d.ID, confs)
		return false, err
	}
}

func (s *SolanaDepositService) fail(d *models.Deposit) error {
	res := s.db.Model(&models.Deposit{}).
		Where("id = ? AND status IN ?", d.ID, []string{models.DepositStatusPending, models.DepositStatusConfirming}).
		Update("status", models.DepositStatusFailed)
	return res.Error
}

// finalize marks the deposit confirmed and posts its journal atomically, then finalizes the payment.
func (s *SolanaDepositService) finalize(ctx context.Context, d *models.Deposit) (bool, error) {
	won := false
	post := func(tx *gorm.DB) error {
		res := tx.Model(&models.Deposit{}).
			Where("id = ? AND status IN ?", d.ID, []string{models.DepositStatusPending, models.DepositStatusConfirming}).
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
