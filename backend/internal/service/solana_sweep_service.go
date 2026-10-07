package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	"sort"
	"strconv"
	"time"

	"github.com/payminto/payminto/backend/internal/blockchain/solana"
	"github.com/payminto/payminto/backend/internal/ledger"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// SolanaSweepConfig tunes sweeps; zero values take the defaults below.
type SolanaSweepConfig struct {
	BatchSize                int
	ComputeUnitLimit         uint32
	PriorityFeeMicroLamports uint64
	CloseAccounts            bool
	// MaxAttempts bounds rebuilds after blockhash expiry before the sweep is failed and released.
	MaxAttempts int
	// FetchNodes is how many distinct endpoints confirm a signature's absence before an attempt is called expired.
	FetchNodes int
	// ValidityWindow is how long a blockhash stays usable; a sweep or attempt with no evidence after it is resolved by age.
	ValidityWindow time.Duration
	// HistoryPages bounds how far explainDrain reads an account's signature history (pages of 100).
	HistoryPages int
	// DrainWait is how long a sweep waits on an account below its claim, with nothing finalized, before
	// the account's history must explain the drain or the sweep fails with an anomaly.
	DrainWait time.Duration
}

func (c SolanaSweepConfig) withDefaults() SolanaSweepConfig {
	if c.BatchSize <= 0 || c.BatchSize > solana.DefaultSweepBatchSize {
		c.BatchSize = solana.DefaultSweepBatchSize
	}
	if c.ComputeUnitLimit == 0 {
		c.ComputeUnitLimit = 120_000
	}
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = 3
	}
	if c.FetchNodes <= 0 {
		c.FetchNodes = 2
	}
	if c.ValidityWindow <= 0 {
		c.ValidityWindow = 2 * time.Minute
	}
	if c.HistoryPages <= 0 {
		c.HistoryPages = 5
	}
	if c.DrainWait <= 0 {
		c.DrainWait = time.Hour
	}
	return c
}

// Anomaly reasons the sweeper writes to missed_deposits.
const (
	anomalyUnexplainedDrain    = "solana_unexplained_drain"
	anomalySweepMismatch       = "solana_sweep_mismatch"
	anomalyFailedSweepLanded   = "solana_failed_sweep_landed"
	anomalyEvidenceUnavailable = "solana_evidence_unavailable"
	anomalySweepNeverSent      = "solana_sweep_never_sent"
	anomalyUntrackedSweep      = "solana_untracked_sweep"
)

// SolanaSweepService drains confirmed SPL deposits into the hot wallet's ATA with a sponsored fee
// payer. SweepConfirmed writes the rows and the lock, signs, persists the signature, then sends.
// trackPageSize bounds how many in-flight sweeps one query loads; TrackConfirmations pages by id.
const trackPageSize = 200

// TrackConfirmations runs the reconciler of SOLANA_SWEEPS.md over every in-flight sweep: every
// decision is derived from the attempt rows, the deposit links and the chain, never from memory.
type SolanaSweepService struct {
	db         *gorm.DB
	client     *solana.Client
	chain      *models.Blockchain
	deposits   repository.DepositRepository
	accounts   repository.SolanaDepositAccountRepository
	currencies repository.BlockchainCurrencyRepository
	missed     repository.MissedDepositRepository
	sweepRepo  repository.SweepRepository
	sweepTxs   repository.SweepTransactionRepository
	sweepSvc   *SweepService
	sweepTxSvc *SweepTransactionService
	keys       KeyProvider
	feePayer   solana.Signer
	hotWallet  solana.PublicKey
	journal    *ledger.Service
	cfg        SolanaSweepConfig
	now        func() time.Time
}

// NewSolanaSweepService wires the sweeper; journal may be nil (rent then goes unbooked and is logged).
func NewSolanaSweepService(
	db *gorm.DB,
	client *solana.Client,
	chain *models.Blockchain,
	deposits repository.DepositRepository,
	accounts repository.SolanaDepositAccountRepository,
	currencies repository.BlockchainCurrencyRepository,
	missed repository.MissedDepositRepository,
	sweepRepo repository.SweepRepository,
	sweepTxs repository.SweepTransactionRepository,
	sweepSvc *SweepService,
	sweepTxSvc *SweepTransactionService,
	keys KeyProvider,
	feePayer solana.Signer,
	hotWallet solana.PublicKey,
	journal *ledger.Service,
	cfg SolanaSweepConfig,
) *SolanaSweepService {
	return &SolanaSweepService{
		db: db, client: client, chain: chain, deposits: deposits, accounts: accounts, currencies: currencies, missed: missed,
		sweepRepo: sweepRepo, sweepTxs: sweepTxs, sweepSvc: sweepSvc, sweepTxSvc: sweepTxSvc, keys: keys,
		feePayer: feePayer, hotWallet: hotWallet, journal: journal, cfg: cfg.withDefaults(), now: time.Now,
	}
}

type sweepGroup struct {
	tokenAccount string
	deposits     []models.Deposit
}

// SweepConfirmed starts one sweep per batch of confirmed token deposits on accounts with no sweep
// in flight and returns how many were started. It does not wait for confirmation. Per-batch
// errors are logged; the batch's deposits return to confirmed.
func (s *SolanaSweepService) SweepConfirmed(ctx context.Context) (int, error) {
	s.releaseOrphanLocks()
	confirmed, err := s.deposits.ListByStatus(models.DepositStatusConfirmed)
	if err != nil {
		return 0, err
	}
	byCurrency := map[uint]map[string]*sweepGroup{}
	for _, d := range confirmed {
		bc := d.BlockchainCurrency
		if bc == nil || bc.BlockchainCode != solana.ChainCode || bc.Address == "" {
			continue
		}
		groups := byCurrency[bc.ID]
		if groups == nil {
			groups = map[string]*sweepGroup{}
			byCurrency[bc.ID] = groups
		}
		g := groups[d.ToAddress]
		if g == nil {
			g = &sweepGroup{tokenAccount: d.ToAddress}
			groups[d.ToAddress] = g
		}
		g.deposits = append(g.deposits, d)
	}
	sent := 0
	for bcID, groups := range byCurrency {
		list := make([]*sweepGroup, 0, len(groups))
		for _, g := range groups {
			if s.skipAccount(g.tokenAccount) {
				continue
			}
			list = append(list, g)
		}
		sort.Slice(list, func(i, j int) bool { return list[i].deposits[0].ID < list[j].deposits[0].ID })
		for start := 0; start < len(list); start += s.cfg.BatchSize {
			if ctx.Err() != nil {
				return sent, ctx.Err()
			}
			end := min(start+s.cfg.BatchSize, len(list))
			ok, err := s.sweepBatch(ctx, bcID, list[start:end])
			if err != nil {
				log.Printf("[solana sweep] batch for currency %d: %v", bcID, err)
			}
			if ok {
				sent++
			}
		}
	}
	return sent, nil
}

// releaseOrphanLocks deletes locks whose sweep is no longer in flight (completed, failed or gone).
func (s *SolanaSweepService) releaseOrphanLocks() {
	if err := s.db.Exec("DELETE FROM solana_sweep_locks WHERE sweep_id NOT IN (SELECT id FROM sweeps WHERE status IN ? AND deleted_at IS NULL)",
		[]string{SweepStatusPending, SweepStatusProcessing}).Error; err != nil {
		log.Printf("[solana sweep] release orphan locks: %v", err)
	}
}

// skipAccount reports accounts that must not be swept now: drained (anomaly recorded, operator
// owns it) or holding a lock because a sweep is in flight.
func (s *SolanaSweepService) skipAccount(tokenAccount string) bool {
	acct, err := s.accounts.GetByTokenAccount(tokenAccount)
	if err != nil || acct.Status == models.SolanaDepositAccountDrained {
		return true
	}
	var locks int64
	s.db.Model(&models.SolanaSweepLock{}).Where("token_account = ?", tokenAccount).Count(&locks)
	return locks > 0
}

type sweepItem struct {
	group  *sweepGroup
	acct   *models.SolanaDepositAccount
	amount uint64
	close  bool
}

// claimedSum is the base units the sweep's deposits add up to for one account.
func claimedSum(deposits []models.Deposit, decimals uint8) uint64 {
	total := decimal.Zero
	for _, d := range deposits {
		total = total.Add(d.Amount)
	}
	return total.Shift(int32(decimals)).BigInt().Uint64()
}

// amountFor decides what one account's sweep moves: the claimed deposits' sum, read against the
// finalized balance. Money beyond the claim (a deposit landing mid-sweep) stays for the next sweep
// and the account is not closed; a balance below the claim means something drained it.
func amountFor(balance, claimed uint64) (amount uint64, close bool, short bool) {
	switch {
	case balance == 0 || balance < claimed:
		return 0, false, true
	case balance == claimed:
		return claimed, true, false
	default:
		return claimed, false, false
	}
}

func (s *SolanaSweepService) sweepBatch(ctx context.Context, bcID uint, groups []*sweepGroup) (bool, error) {
	bc, err := s.currencies.GetByID(bcID)
	if err != nil {
		return false, err
	}
	params, err := s.paramsFor(bc)
	if err != nil {
		return false, err
	}

	// Claim first so no second worker can start a sweep for the same deposits.
	var claimed []*sweepGroup
	for _, g := range groups {
		if s.claim(g) {
			claimed = append(claimed, g)
		}
	}
	if len(claimed) == 0 {
		return false, nil
	}
	release := func() {
		for _, g := range claimed {
			s.release(g.deposits)
		}
	}

	var items []sweepItem
	for _, g := range claimed {
		acct, err := s.accounts.GetByTokenAccount(g.tokenAccount)
		if err != nil {
			release()
			return false, fmt.Errorf("solana deposit account %s: %w", g.tokenAccount, err)
		}
		balance, err := s.balance(ctx, g.tokenAccount)
		if err != nil {
			release()
			return false, fmt.Errorf("balance of %s: %w", g.tokenAccount, err)
		}
		amount, close, short := amountFor(balance, claimedSum(g.deposits, params.Decimals))
		if short {
			// Nothing to move, or less than the claim: never mark swept here. Either a tracked sweep
			// drained it (the tracker books it) or someone else did (anomaly). Deposits go back to confirmed.
			s.release(g.deposits)
			s.explainDrain(ctx, acct, balance, nil)
			continue
		}
		items = append(items, sweepItem{group: g, acct: acct, amount: amount, close: close})
	}
	if len(items) == 0 {
		return false, nil
	}

	// Rows and locks first: a signature can never exist without the rows that let the tracker find it,
	// and the lock's unique index refuses a second sweep on an account already in flight.
	hotATA, err := solana.AssociatedTokenAddress(params.HotWalletOwner, params.Mint, params.TokenProgram)
	if err != nil {
		for _, it := range items {
			s.release(it.group.deposits)
		}
		return false, err
	}
	sweep := &models.Sweep{Status: SweepStatusProcessing, BlockchainID: s.chain.ID}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(sweep).Error; err != nil {
			return err
		}
		for _, it := range items {
			amount := decimal.NewFromBigInt(new(big.Int).SetUint64(it.amount), -int32(params.Decimals))
			st := &models.SweepTransaction{Amount: amount, FromAddress: it.acct.TokenAccount, ToAddress: hotATA.String(),
				Status: SweepTxStatusPending, SweepID: sweep.ID, BlockchainCurrencyID: bc.ID}
			if err := tx.Create(st).Error; err != nil {
				return err
			}
			for _, d := range it.group.deposits {
				if err := tx.Create(&models.SolanaSweepDeposit{SweepID: sweep.ID, DepositID: d.ID}).Error; err != nil {
					return err
				}
			}
			if err := tx.Create(&models.SolanaSweepLock{TokenAccount: it.acct.TokenAccount, SweepID: sweep.ID}).Error; err != nil {
				return fmt.Errorf("account %s already has a sweep in flight: %w", it.acct.TokenAccount, err)
			}
		}
		return nil
	})
	if err != nil {
		for _, it := range items {
			s.release(it.group.deposits)
		}
		return false, fmt.Errorf("sweep rows: %w", err)
	}
	if err := s.signAndSend(ctx, sweep, params, items, 1); err != nil {
		return false, err
	}
	log.Printf("[solana sweep] sweep %d started: %d accounts of %s to %s", sweep.ID, len(items), bc.CurrencyCode, hotATA)
	return true, nil
}

// signAndSend builds and signs an attempt, persists its signature and last valid block height,
// then sends. A node rejection fails the attempt (and the sweep, when it is the first); a
// transport error leaves the attempt "signed" for the reconciler, which resolves it by lookup.
func (s *SolanaSweepService) signAndSend(ctx context.Context, sw *models.Sweep, params solana.SweepParams, items []sweepItem, attemptNo int) error {
	signed, err := s.sign(ctx, params, items)
	if err != nil {
		if attemptNo == 1 {
			if ferr := s.failSweep(sw, "could not sign: "+err.Error()); ferr != nil {
				log.Printf("[solana sweep] sweep %d fail after sign error: %v", sw.ID, ferr)
			}
		}
		return fmt.Errorf("sign: %w", err)
	}
	var seen int64
	if err := s.db.WithContext(ctx).Model(&models.SolanaSweepAttempt{}).Where("signature = ?", signed.Signature).Count(&seen).Error; err != nil {
		return fmt.Errorf("check attempt %s: %w", signed.Signature, err)
	}
	if seen > 0 {
		// Same message against the same blockhash: nothing new to send. A fresh blockhash next pass differs.
		if attemptNo == 1 {
			if ferr := s.failSweep(sw, "blockhash not fresh, transaction identical to an earlier attempt"); ferr != nil {
				log.Printf("[solana sweep] sweep %d fail on a repeated signature: %v", sw.ID, ferr)
			}
		}
		return fmt.Errorf("attempt %s already exists; waiting for a fresh blockhash", signed.Signature)
	}
	attempt := &models.SolanaSweepAttempt{SweepID: sw.ID, AttemptNo: attemptNo, Signature: signed.Signature, Blockhash: signed.Blockhash,
		LastValidBlockHeight: signed.LastValidBlockHeight, Status: models.SolanaSweepAttemptSigned}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Only a sweep still in flight takes an attempt: one the reconciler failed meanwhile must not be revived here.
		res := tx.Model(&models.Sweep{}).Where("id = ? AND status IN ?", sw.ID, []string{SweepStatusProcessing, SweepStatusPending}).Update("status", SweepStatusPending)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return fmt.Errorf("sweep %d is no longer in flight", sw.ID)
		}
		// The (sweep_id, attempt_no) unique index refuses a second worker's attempt with the same number.
		if err := tx.Create(attempt).Error; err != nil {
			return err
		}
		return tx.Model(&models.SweepTransaction{}).Where("sweep_id = ?", sw.ID).
			Updates(map[string]any{"tx_hash": signed.Signature, "status": SweepTxStatusBroadcast}).Error
	})
	if err != nil {
		// Not sent: the sweep stays processing and the reconciler resolves it from the chain.
		return fmt.Errorf("persist attempt %s before send: %w", signed.Signature, err)
	}
	err = solana.SendSigned(ctx, s.client, signed)
	switch {
	case err == nil:
		s.db.Model(attempt).Update("status", models.SolanaSweepAttemptSent)
		return nil
	case solana.IsRejection(err):
		s.db.Model(attempt).Update("status", models.SolanaSweepAttemptFailed)
		if attemptNo == 1 {
			if ferr := s.failSweep(sw, "node rejected the transaction: "+err.Error()); ferr != nil {
				log.Printf("[solana sweep] sweep %d fail after rejection: %v", sw.ID, ferr)
			}
		}
		return fmt.Errorf("send rejected: %w", err)
	default:
		log.Printf("[solana sweep] sweep %d attempt %s: send transport error, signature persisted, tracker resolves it: %v", sw.ID, signed.Signature, err)
		return nil
	}
}

func (s *SolanaSweepService) paramsFor(bc *models.BlockchainCurrency) (solana.SweepParams, error) {
	mint, err := solana.ParsePublicKey(bc.Address)
	if err != nil {
		return solana.SweepParams{}, fmt.Errorf("mint %q: %w", bc.Address, err)
	}
	program, err := solana.TokenProgramFor(bc.Standard)
	if err != nil {
		return solana.SweepParams{}, err
	}
	return solana.SweepParams{
		FeePayer: s.feePayer.PublicKey(), HotWalletOwner: s.hotWallet, Mint: mint, TokenProgram: program, Decimals: uint8(bc.WalletPrecision),
		ComputeUnitLimit: s.cfg.ComputeUnitLimit, PriorityFeeMicroLamports: s.cfg.PriorityFeeMicroLamports, CloseAccounts: s.cfg.CloseAccounts,
	}, nil
}

// balance reads a token account's finalized base units; a missing account is 0.
func (s *SolanaSweepService) balance(ctx context.Context, tokenAccount string) (uint64, error) {
	bal, err := s.client.GetTokenAccountBalance(ctx, tokenAccount, solana.CommitmentFinalized)
	if errors.Is(err, solana.ErrAccountNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(bal.Amount, 10, 64)
}

// sign resolves the owner keys and signs with the fee payer; keys are zeroed before returning.
func (s *SolanaSweepService) sign(ctx context.Context, params solana.SweepParams, items []sweepItem) (solana.Signed, error) {
	signers := []solana.Signer{s.feePayer}
	var owners []solana.Ed25519Signer
	defer func() {
		for i := range owners {
			owners[i].Zero()
		}
	}()
	params.Items = nil
	for _, it := range items {
		owner, err := solana.ParsePublicKey(it.acct.OwnerAddress)
		if err != nil {
			return solana.Signed{}, err
		}
		ata, _ := solana.ParsePublicKey(it.acct.TokenAccount)
		priv, _, err := s.keys.PrivateKeyForAddress(it.acct.OwnerAddress)
		if err != nil {
			return solana.Signed{}, fmt.Errorf("key for %s: %w", it.acct.OwnerAddress, err)
		}
		signer, err := solana.NewEd25519Signer(priv)
		zeroBytes(priv)
		if err != nil {
			return solana.Signed{}, err
		}
		owners = append(owners, signer)
		signers = append(signers, signer)
		params.Items = append(params.Items, solana.SweepItem{Owner: owner, TokenAccount: ata, Amount: it.amount, Close: it.close})
	}
	build := func(blockhash string) (solana.Message, error) {
		ixs, _, err := solana.SweepInstructions(params)
		if err != nil {
			return solana.Message{}, err
		}
		bh, err := solana.ParsePublicKey(blockhash)
		if err != nil {
			return solana.Message{}, err
		}
		return solana.CompileMessage(params.FeePayer, bh, ixs)
	}
	return solana.SignForSend(ctx, s.client, build, signers)
}

func (s *SolanaSweepService) claim(g *sweepGroup) bool {
	var won []models.Deposit
	for _, d := range g.deposits {
		n, err := s.deposits.ClaimForSweep(d.ID)
		if err != nil || n == 0 {
			s.release(won)
			return false
		}
		won = append(won, d)
	}
	return true
}

func (s *SolanaSweepService) release(deposits []models.Deposit) {
	for _, d := range deposits {
		if err := s.deposits.UpdateStatus(d.ID, models.DepositStatusConfirmed); err != nil {
			log.Printf("[solana sweep] release deposit %d: %v", d.ID, err)
		}
	}
}

// history pages an account's signatures, newest first, up to HistoryPages pages.
func (s *SolanaSweepService) history(ctx context.Context, tokenAccount string) ([]solana.SignatureInfo, error) {
	var all []solana.SignatureInfo
	before := ""
	for page := 0; page < s.cfg.HistoryPages; page++ {
		sigs, err := s.client.GetSignaturesForAddressPage(ctx, tokenAccount, "", before, 100, solana.CommitmentConfirmed)
		if err != nil {
			return all, err
		}
		all = append(all, sigs...)
		if len(sigs) < 100 {
			break
		}
		before = sigs[len(sigs)-1].Signature
	}
	return all, nil
}

// explainDrain looks for one of our sweep signatures in the account's history. A signature of a
// sweep still tracked is left to the tracker; a signature of a sweep booked failed revives it
// (the failure was wrong); a transaction our fee payer signed but no attempt row knows is recorded
// for the given in-flight sweep when there is one, else flagged. Nothing of ours explaining the
// drain is an anomaly and the account is set aside for an operator.
func (s *SolanaSweepService) explainDrain(ctx context.Context, acct *models.SolanaDepositAccount, balance uint64, inflight *models.Sweep) (explained bool) {
	sigs, err := s.history(ctx, acct.TokenAccount)
	if err != nil {
		log.Printf("[solana sweep] %s holds %d; history lookup failed: %v", acct.TokenAccount, balance, err)
		return true
	}
	for _, info := range sigs {
		if info.Failed() {
			continue
		}
		var attempt models.SolanaSweepAttempt
		if err := s.db.Where("signature = ?", info.Signature).First(&attempt).Error; err != nil {
			continue
		}
		var sw models.Sweep
		if err := s.db.First(&sw, attempt.SweepID).Error; err != nil {
			continue
		}
		switch sw.Status {
		case SweepStatusPending, SweepStatusProcessing:
			log.Printf("[solana sweep] %s drained by tracked sweep %d (%s); the tracker books it", acct.TokenAccount, sw.ID, info.Signature)
			return true
		case SweepStatusFailed:
			log.Printf("[solana sweep] %s drained by sweep %d (%s) that was booked failed; reviving it", acct.TokenAccount, sw.ID, info.Signature)
			s.recordAnomaly(info.Signature, acct.TokenAccount, decimal.Zero, fmt.Sprintf("%s: sweep %d", anomalyFailedSweepLanded, sw.ID))
			err := s.db.Transaction(func(tx *gorm.DB) error {
				if err := tx.Model(&models.SolanaSweepAttempt{}).Where("id = ?", attempt.ID).Update("status", models.SolanaSweepAttemptSent).Error; err != nil {
					return err
				}
				if err := tx.Model(&models.Sweep{}).Where("id = ?", sw.ID).Update("status", SweepStatusPending).Error; err != nil {
					return err
				}
				return tx.Create(&models.SolanaSweepLock{TokenAccount: acct.TokenAccount, SweepID: sw.ID}).Error
			})
			if err != nil {
				log.Printf("[solana sweep] revive sweep %d: %v", sw.ID, err)
			}
			return true
		}
	}
	// Last resort: a transaction our fee payer signed that no attempt row knows.
	for i, info := range sigs {
		if i >= 20 || info.Failed() {
			continue
		}
		tx, err := s.client.GetTransaction(ctx, info.Signature, solana.CommitmentConfirmed)
		if err != nil || tx == nil || len(tx.Transaction.Message.AccountKeys) == 0 || tx.Transaction.Message.AccountKeys[0].Pubkey != s.feePayer.PublicKey().String() {
			continue
		}
		if inflight != nil {
			// The transaction already executed, so its blockhash cannot outlive the current height by more
			// than the validity span: an upper bound that can only delay expiry evidence, never hasten it.
			height, err := s.client.GetBlockHeight(ctx, solana.CommitmentFinalized)
			if err != nil || height == 0 {
				log.Printf("[solana sweep] %s: untracked transaction %s found; block height unavailable, retrying next pass: %v", acct.TokenAccount, info.Signature, err)
				return true
			}
			log.Printf("[solana sweep] %s drained by our untracked transaction %s; recording it as sweep %d's attempt", acct.TokenAccount, info.Signature, inflight.ID)
			err = s.db.Transaction(func(dbTx *gorm.DB) error {
				var n int64
				if err := dbTx.Model(&models.SolanaSweepAttempt{}).Where("sweep_id = ?", inflight.ID).Count(&n).Error; err != nil {
					return err
				}
				if err := dbTx.Create(&models.SolanaSweepAttempt{SweepID: inflight.ID, AttemptNo: int(n) + 1, Signature: info.Signature, Blockhash: tx.Transaction.Message.RecentBlockhash,
					LastValidBlockHeight: height + solana.BlockhashValidityBlocks, Status: models.SolanaSweepAttemptSent}).Error; err != nil {
					return err
				}
				if err := dbTx.Model(&models.SweepTransaction{}).Where("sweep_id = ?", inflight.ID).Updates(map[string]any{"tx_hash": info.Signature, "status": SweepTxStatusBroadcast}).Error; err != nil {
					return err
				}
				return dbTx.Model(&models.Sweep{}).Where("id = ?", inflight.ID).Update("status", SweepStatusPending).Error
			})
			if err != nil {
				log.Printf("[solana sweep] record recovered attempt %s: %v", info.Signature, err)
			}
			return true
		}
		s.recordAnomaly(info.Signature, acct.TokenAccount, decimal.Zero, anomalyUntrackedSweep+": our fee payer signed it, no attempt row")
		return true
	}
	last := ""
	if len(sigs) > 0 {
		last = sigs[0].Signature
	}
	log.Printf("[solana sweep] %s holds %d, below its claim, and no sweep of ours explains it (last signature %s); anomaly recorded", acct.TokenAccount, balance, last)
	_ = s.accounts.UpdateStatus(acct.ID, models.SolanaDepositAccountDrained)
	s.recordAnomaly(last, acct.TokenAccount, decimal.Zero, anomalyUnexplainedDrain+": token account below its claimed deposits without a booked sweep")
	return false
}

func (s *SolanaSweepService) recordAnomaly(signature, to string, amount decimal.Decimal, reason string) {
	if len(reason) > 100 {
		reason = reason[:100]
	}
	if signature == "" {
		signature = "none"
	}
	existing, _ := s.missed.GetByTxHash(signature)
	for _, m := range existing {
		if m.ToAddress == to && m.Reason == reason {
			return
		}
	}
	if err := s.missed.Create(&models.MissedDeposit{TxHash: signature, BlockchainID: s.chain.ID, ToAddress: to, Amount: amount, Reason: reason, Status: models.MissedDepositStatusPending}); err != nil {
		log.Printf("[solana sweep] record anomaly %s: %v", reason, err)
	}
}

// TrackConfirmations runs the reconciler over every in-flight sweep and releases orphan locks.
func (s *SolanaSweepService) TrackConfirmations(ctx context.Context) (int, error) {
	completed := 0
	var after uint
	for {
		var sweeps []models.Sweep
		if err := s.db.Where("blockchain_id = ? AND status IN ? AND id > ?", s.chain.ID, []string{SweepStatusPending, SweepStatusProcessing}, after).
			Order("id").Limit(trackPageSize).Find(&sweeps).Error; err != nil {
			return completed, err
		}
		for i := range sweeps {
			if ctx.Err() != nil {
				return completed, ctx.Err()
			}
			done, err := s.reconcileSweep(ctx, &sweeps[i])
			if err != nil {
				log.Printf("[solana sweep] reconcile sweep %d: %v", sweeps[i].ID, err)
			} else if done {
				completed++
			}
		}
		if len(sweeps) < trackPageSize {
			break
		}
		after = sweeps[len(sweeps)-1].ID
	}
	s.releaseOrphanLocks()
	return completed, nil
}

// reconcileSweep applies exactly one transition of SOLANA_SWEEPS.md to a sweep, from its rows and the chain.
func (s *SolanaSweepService) reconcileSweep(ctx context.Context, sw *models.Sweep) (bool, error) {
	var attempts []models.SolanaSweepAttempt
	if err := s.db.Where("sweep_id = ?", sw.ID).Order("id").Find(&attempts).Error; err != nil {
		return false, err
	}
	if len(attempts) == 0 {
		return false, s.resolveNoAttempt(ctx, sw)
	}
	sigs := make([]string, len(attempts))
	for i := range attempts {
		sigs[i] = attempts[i].Signature
	}
	statuses, err := s.client.GetSignatureStatuses(ctx, sigs)
	if err != nil {
		return false, err
	}
	var landed *models.SolanaSweepAttempt
	anyConfirmed := false
	for i := range attempts {
		var st *solana.SignatureStatus
		if i < len(statuses) {
			st = statuses[i]
		}
		switch {
		case st == nil:
		case st.Failed():
			if attempts[i].Status != models.SolanaSweepAttemptFailed {
				_ = s.db.Model(&attempts[i]).Update("status", models.SolanaSweepAttemptFailed)
				attempts[i].Status = models.SolanaSweepAttemptFailed
			}
		case st.ConfirmationStatus == solana.CommitmentFinalized:
			if landed == nil {
				landed = &attempts[i]
			}
		default:
			anyConfirmed = true
		}
	}
	if landed != nil {
		return true, s.book(ctx, sw, landed, attempts)
	}
	if anyConfirmed {
		s.db.Model(&models.SweepTransaction{}).Where("sweep_id = ? AND status = ?", sw.ID, SweepTxStatusBroadcast).Update("status", SweepTxStatusConfirming)
		return false, nil
	}
	latest := &attempts[len(attempts)-1]
	if latest.Status == models.SolanaSweepAttemptFailed || latest.Status == models.SolanaSweepAttemptExpired {
		return false, s.rebuildOrFail(ctx, sw, attempts)
	}
	expired, err := s.attemptExpired(ctx, sw, latest)
	if err != nil || !expired {
		return false, err
	}
	if err := s.db.Model(latest).Update("status", models.SolanaSweepAttemptExpired).Error; err != nil {
		return false, err
	}
	latest.Status = models.SolanaSweepAttemptExpired
	return false, s.rebuildOrFail(ctx, sw, attempts)
}

// attemptExpired is the expiry evidence: the finalized block height past the attempt's validity
// (or, when the height is unknown, the attempt older than the validity window) and the signature
// absent at finalized on two distinct endpoints. One endpoint cannot evidence it: wait.
func (s *SolanaSweepService) attemptExpired(ctx context.Context, sw *models.Sweep, latest *models.SolanaSweepAttempt) (bool, error) {
	if latest.LastValidBlockHeight > 0 {
		height, err := s.client.GetBlockHeight(ctx, solana.CommitmentFinalized)
		if err != nil {
			return false, err
		}
		if height <= latest.LastValidBlockHeight {
			return false, nil
		}
	} else if s.now().Sub(latest.CreatedAt) < s.cfg.ValidityWindow {
		return false, nil
	}
	tx, reached, err := s.client.GetTransactionFromDistinctNodes(ctx, latest.Signature, solana.CommitmentFinalized, max(2, s.cfg.FetchNodes))
	if err != nil || tx != nil {
		return false, err
	}
	if reached < 2 {
		s.recordAnomaly(latest.Signature, "", decimal.Zero, fmt.Sprintf("%s: sweep %d has one rpc endpoint; expiry cannot be evidenced", anomalyEvidenceUnavailable, sw.ID))
		return false, nil
	}
	return true, nil
}

// resolveNoAttempt handles a sweep whose rows exist but whose signature was never persisted (a crash
// or a database error before the attempt row; nothing is sent before it). Past the validity window
// the chain decides: every account at or above its claim means nothing was sent.
func (s *SolanaSweepService) resolveNoAttempt(ctx context.Context, sw *models.Sweep) error {
	if s.now().Sub(sw.CreatedAt) < s.cfg.ValidityWindow {
		return nil
	}
	txs, err := s.sweepTxs.ListBySweep(sw.ID)
	if err != nil || len(txs) == 0 {
		return err
	}
	bc, err := s.currencies.GetByID(txs[0].BlockchainCurrencyID)
	if err != nil {
		return err
	}
	for i := range txs {
		balance, err := s.balance(ctx, txs[i].FromAddress)
		if err != nil {
			return err
		}
		claimed := txs[i].Amount.Shift(int32(bc.WalletPrecision)).BigInt().Uint64()
		if balance >= claimed {
			continue
		}
		acct, err := s.accounts.GetByTokenAccount(txs[i].FromAddress)
		if err != nil {
			return err
		}
		if s.explainDrain(ctx, acct, balance, sw) {
			return nil
		}
		return s.failSweep(sw, "an account drained with nothing of ours in its history")
	}
	s.recordAnomaly("none", txs[0].FromAddress, decimal.Zero, fmt.Sprintf("%s: sweep %d had no attempt after %s", anomalySweepNeverSent, sw.ID, s.cfg.ValidityWindow))
	return s.failSweep(sw, "no attempt was ever signed")
}

// rebuildOrFail re-reads every account against the sweep's claim first: an account below its claim
// while no attempt is finalized means an attempt may have landed behind lagging nodes, so the sweep
// waits (at the budget too). Otherwise a fresh attempt goes out while the budget allows, and past
// it the sweep fails and releases exactly its deposits.
func (s *SolanaSweepService) rebuildOrFail(ctx context.Context, sw *models.Sweep, attempts []models.SolanaSweepAttempt) error {
	txs, err := s.sweepTxs.ListBySweep(sw.ID)
	if err != nil || len(txs) == 0 {
		return err
	}
	bc, err := s.currencies.GetByID(txs[0].BlockchainCurrencyID)
	if err != nil {
		return err
	}
	params, err := s.paramsFor(bc)
	if err != nil {
		return err
	}
	var items []sweepItem
	for i := range txs {
		acct, err := s.accounts.GetByTokenAccount(txs[i].FromAddress)
		if err != nil {
			return err
		}
		balance, err := s.balance(ctx, txs[i].FromAddress)
		if err != nil {
			return err
		}
		claimed := txs[i].Amount.Shift(int32(params.Decimals)).BigInt().Uint64()
		amount, close, short := amountFor(balance, claimed)
		if short {
			if s.now().Sub(attempts[len(attempts)-1].CreatedAt) < s.cfg.DrainWait {
				log.Printf("[solana sweep] sweep %d: %s holds %d of its %d claim while no attempt is finalized; waiting", sw.ID, txs[i].FromAddress, balance, claimed)
				return nil
			}
			if s.explainDrain(ctx, acct, balance, sw) {
				return nil
			}
			return s.failSweep(sw, "an account drained with nothing of ours in its history")
		}
		items = append(items, sweepItem{acct: acct, amount: amount, close: close})
	}
	if len(attempts) >= s.cfg.MaxAttempts {
		return s.failSweep(sw, "no attempt landed in "+strconv.Itoa(len(attempts))+" broadcasts")
	}
	if err := s.signAndSend(ctx, sw, params, items, len(attempts)+1); err != nil {
		return fmt.Errorf("rebuild: %w", err)
	}
	log.Printf("[solana sweep] sweep %d rebuilt (attempt %d)", sw.ID, len(attempts)+1)
	return nil
}

// book completes the sweep from the one attempt that finalized: sweep transactions confirmed with
// their fee share, the sweep journal, rent, the linked deposits swept, the lock released, accounts
// closed when the chain shows them gone, and the hot ATA's balance delta reconciled.
func (s *SolanaSweepService) book(ctx context.Context, sw *models.Sweep, landed *models.SolanaSweepAttempt, attempts []models.SolanaSweepAttempt) error {
	tx, err := s.client.GetTransactionFromNodes(ctx, landed.Signature, solana.CommitmentFinalized, s.cfg.FetchNodes)
	if err != nil {
		return err
	}
	if tx == nil || tx.Meta == nil {
		return fmt.Errorf("finalized %s has no transaction record yet", landed.Signature)
	}
	txs, err := s.sweepTxs.ListBySweep(sw.ID)
	if err != nil || len(txs) == 0 {
		return err
	}
	fee := solana.LamportsToSOL(tx.Meta.Fee)
	reclaimed, funded := solana.RentMovements(tx)
	total := decimal.Zero
	for _, t := range txs {
		total = total.Add(t.Amount)
	}
	share := fee.Div(decimal.NewFromInt(int64(len(txs)))).Truncate(9)
	bcID := txs[0].BlockchainCurrencyID
	err = s.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		for i := range txs {
			g := share
			if i == 0 {
				g = fee.Sub(share.Mul(decimal.NewFromInt(int64(len(txs) - 1))))
			}
			if err := dbTx.Model(&models.SweepTransaction{}).Where("id = ?", txs[i].ID).
				Updates(map[string]any{"status": SweepTxStatusConfirmed, "gas_fee": g, "tx_hash": landed.Signature}).Error; err != nil {
				return err
			}
		}
		for i := range attempts {
			status := models.SolanaSweepAttemptExpired
			if attempts[i].ID == landed.ID {
				status = models.SolanaSweepAttemptLanded
			} else if attempts[i].Status == models.SolanaSweepAttemptFailed {
				continue
			}
			if err := dbTx.Model(&models.SolanaSweepAttempt{}).Where("id = ?", attempts[i].ID).Update("status", status).Error; err != nil {
				return err
			}
		}
		// Deposits become swept only here, through the booked sweep and its links.
		if err := dbTx.Exec("UPDATE deposits SET status = ? WHERE id IN (SELECT deposit_id FROM solana_sweep_deposits WHERE sweep_id = ? AND deleted_at IS NULL) AND status IN ?",
			models.DepositStatusSwept, sw.ID, []string{models.DepositStatusConfirmed, models.DepositStatusSwept}).Error; err != nil {
			return err
		}
		if err := dbTx.Where("sweep_id = ?", sw.ID).Delete(&models.SolanaSweepLock{}).Error; err != nil {
			return err
		}
		if err := s.sweepSvc.completeIn(ctx, dbTx, sw.ID, total, fee, bcID); err != nil {
			return err
		}
		return s.recordRent(ctx, dbTx, sw.ID, bcID, reclaimed, funded)
	})
	if err != nil {
		return err
	}
	for _, t := range txs {
		acct, err := s.accounts.GetByTokenAccount(t.FromAddress)
		if err != nil || !s.cfg.CloseAccounts {
			continue
		}
		if info, err := s.client.GetAccountInfo(ctx, t.FromAddress, solana.CommitmentConfirmed); err == nil && info == nil {
			_ = s.accounts.UpdateStatus(acct.ID, models.SolanaDepositAccountClosed)
		}
	}
	s.reconcile(tx, txs, total)
	log.Printf("[solana sweep] sweep %d booked from %s: %s moved, fee %s SOL, rent reclaimed %d funded %d lamports", sw.ID, landed.Signature, total, fee, reclaimed, funded)
	return nil
}

// reconcile compares the hot ATA's token balance delta in the landed transaction with the amounts
// booked; any difference is an anomaly for an operator, the booking stays as the chain shows it.
func (s *SolanaSweepService) reconcile(tx *solana.ParsedTransaction, txs []models.SweepTransaction, total decimal.Decimal) {
	hotATA := txs[0].ToAddress
	keys := tx.Transaction.Message.AccountKeys
	find := func(list []solana.ParsedTokenBalance) (decimal.Decimal, bool) {
		for _, b := range list {
			if b.AccountIndex < len(keys) && keys[b.AccountIndex].Pubkey == hotATA {
				v, ok := new(big.Int).SetString(b.UITokenAmount.Amount, 10)
				if !ok {
					return decimal.Zero, false
				}
				return decimal.NewFromBigInt(v, -int32(b.UITokenAmount.Decimals)), true
			}
		}
		return decimal.Zero, false
	}
	post, okPost := find(tx.Meta.PostTokenBalances)
	pre, _ := find(tx.Meta.PreTokenBalances)
	if !okPost {
		return
	}
	delta := post.Sub(pre)
	if delta.Equal(total) {
		return
	}
	diff := total.Sub(delta).Abs()
	log.Printf("[solana sweep] %s: hot wallet received %s but %s was booked; anomaly recorded", tx.Signature(), delta, total)
	s.recordAnomaly(tx.Signature(), hotATA, diff, anomalySweepMismatch+": received "+delta.String()+" booked "+total.String())
}

// recordRent books rent in SOL: closing deposit ATAs returns lamports to the fee payer (income),
// creating the hot wallet ATA locks lamports in an account we still own (asset reclassification).
func (s *SolanaSweepService) recordRent(ctx context.Context, tx *gorm.DB, sweepID, bcID uint, reclaimed, funded uint64) error {
	if reclaimed == 0 && funded == 0 {
		return nil
	}
	if s.journal == nil {
		log.Printf("[solana sweep] sweep %d rent reclaimed %d funded %d lamports; no journal wired to book it", sweepID, reclaimed, funded)
		return nil
	}
	native, err := nativeAssetFor(tx, bcID)
	if err != nil {
		return err
	}
	var lines []ledger.Line
	if reclaimed > 0 {
		amount := solana.LamportsToSOL(reclaimed)
		lines = append(lines,
			ledger.Line{Account: legacyAccount("crypto_assets", native, ledger.KindAsset), Amount: amount},
			ledger.Line{Account: legacyAccount("rent_reclaimed", native, ledger.KindIncome), Amount: amount.Neg()},
		)
	}
	if funded > 0 {
		amount := solana.LamportsToSOL(funded)
		lines = append(lines,
			ledger.Line{Account: legacyAccount("token_account_rent", native, ledger.KindAsset), Amount: amount},
			ledger.Line{Account: legacyAccount("crypto_assets", native, ledger.KindAsset), Amount: amount.Neg()},
		)
	}
	id := strconv.FormatUint(uint64(sweepID), 10)
	_, err = s.journal.PostIn(ctx, tx, ledger.Journal{
		Kind:           ledger.KindAdjustment,
		Reference:      ledger.Reference{Type: "solana_rent", ID: id},
		IdempotencyKey: "payminto:solana_rent:" + id,
		Metadata:       map[string]any{"blockchain_currency_id": bcID, "reclaimed_lamports": reclaimed, "funded_lamports": funded},
		Lines:          lines,
	})
	return err
}

// nativeAssetFor resolves the chain's native ledger asset (SOL.SOLANA) for a token row.
func nativeAssetFor(tx *gorm.DB, bcID uint) (string, error) {
	assets, err := blockchainCurrencyAssetResolver()(tx, bcID)
	if err != nil {
		return "", err
	}
	if assets.Native == "" {
		return "", fmt.Errorf("ledger: no native currency row for blockchain currency %d; seed SOL on SOLANA", bcID)
	}
	return assets.Native, nil
}

// failSweep releases exactly the deposits this sweep claimed, its lock, and marks its rows.
func (s *SolanaSweepService) failSweep(sw *models.Sweep, reason string) error {
	log.Printf("[solana sweep] sweep %d %s; releasing its deposits", sw.ID, reason)
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("UPDATE deposits SET status = ? WHERE id IN (SELECT deposit_id FROM solana_sweep_deposits WHERE sweep_id = ? AND deleted_at IS NULL) AND status = ?",
			models.DepositStatusConfirmed, sw.ID, models.DepositStatusSwept).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.SweepTransaction{}).Where("sweep_id = ?", sw.ID).Update("status", SweepTxStatusFailed).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.SolanaSweepAttempt{}).Where("sweep_id = ? AND status IN ?", sw.ID, []string{models.SolanaSweepAttemptSigned, models.SolanaSweepAttemptSent}).Update("status", models.SolanaSweepAttemptExpired).Error; err != nil {
			return err
		}
		if err := tx.Where("sweep_id = ?", sw.ID).Delete(&models.SolanaSweepLock{}).Error; err != nil {
			return err
		}
		return tx.Model(&models.Sweep{}).Where("id = ?", sw.ID).Update("status", SweepStatusFailed).Error
	})
}
