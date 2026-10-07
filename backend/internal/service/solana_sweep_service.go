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
	// FetchNodes is how many pool picks confirm a signature's absence before an attempt is called expired.
	FetchNodes int
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
	return c
}

// Anomaly reasons the sweeper writes to missed_deposits.
const (
	anomalyUnexplainedDrain = "solana_unexplained_drain"
	anomalySweepMismatch    = "solana_sweep_mismatch"
)

// SolanaSweepService drains confirmed SPL deposits into the hot wallet's ATA with a sponsored fee
// payer. SweepConfirmed only broadcasts and records; TrackConfirmations follows every attempt a
// sweep ever broadcast, books exactly one landed attempt, rebuilds on evidenced expiry and
// releases the sweep's own deposits when it fails. Deposits become swept only through a booked sweep.
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

// SweepConfirmed broadcasts one transaction per batch of confirmed token deposits and returns how
// many were sent. It does not wait for confirmation. Per-batch errors are logged; the batch's
// deposits return to confirmed.
func (s *SolanaSweepService) SweepConfirmed(ctx context.Context) (int, error) {
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

// skipAccount reports accounts that must not be swept again: drained (anomaly recorded) or part
// of a sweep still being tracked.
func (s *SolanaSweepService) skipAccount(tokenAccount string) bool {
	acct, err := s.accounts.GetByTokenAccount(tokenAccount)
	if err != nil {
		return true
	}
	return acct.Status == models.SolanaDepositAccountDrained
}

type sweepItem struct {
	group  *sweepGroup
	acct   *models.SolanaDepositAccount
	amount uint64
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

	// Claim first so no second worker can broadcast the same deposits.
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
		amount, err := s.balance(ctx, g.tokenAccount)
		if err != nil {
			release()
			return false, fmt.Errorf("balance of %s: %w", g.tokenAccount, err)
		}
		if amount == 0 {
			// Nothing to move: never mark swept here. Either a tracked sweep drained it (the tracker
			// books it) or someone else did (anomaly). Its deposits go back to confirmed either way.
			s.release(g.deposits)
			s.explainDrain(ctx, acct)
			continue
		}
		items = append(items, sweepItem{group: g, acct: acct, amount: amount})
	}
	if len(items) == 0 {
		return false, nil
	}

	sent, hotATA, err := s.broadcast(ctx, params, items)
	if err != nil {
		for _, it := range items {
			s.release(it.group.deposits)
		}
		return false, fmt.Errorf("broadcast: %w", err)
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		sweep := &models.Sweep{Status: SweepStatusPending, BlockchainID: s.chain.ID}
		if err := tx.Create(sweep).Error; err != nil {
			return err
		}
		for _, it := range items {
			amount := decimal.NewFromBigInt(new(big.Int).SetUint64(it.amount), -int32(params.Decimals))
			st := &models.SweepTransaction{TxHash: sent.Signature, Amount: amount, FromAddress: it.acct.TokenAccount, ToAddress: hotATA.String(),
				Status: SweepTxStatusBroadcast, SweepID: sweep.ID, BlockchainCurrencyID: bc.ID}
			if err := tx.Create(st).Error; err != nil {
				return err
			}
			for _, d := range it.group.deposits {
				if err := tx.Create(&models.SolanaSweepDeposit{SweepID: sweep.ID, DepositID: d.ID}).Error; err != nil {
					return err
				}
			}
		}
		return tx.Create(&models.SolanaSweepAttempt{SweepID: sweep.ID, Signature: sent.Signature, Blockhash: sent.Blockhash,
			LastValidBlockHeight: sent.LastValidBlockHeight, Status: models.SolanaSweepAttemptSent}).Error
	})
	if err != nil {
		return true, fmt.Errorf("sweep %s broadcast but rows failed (reconcile from the signature): %w", sent.Signature, err)
	}
	log.Printf("[solana sweep] sent %s: %d accounts of %s to %s", sent.Signature, len(items), bc.CurrencyCode, hotATA)
	return true, nil
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

// balance reads a token account's base units; a missing account is 0.
func (s *SolanaSweepService) balance(ctx context.Context, tokenAccount string) (uint64, error) {
	bal, err := s.client.GetTokenAccountBalance(ctx, tokenAccount, solana.CommitmentConfirmed)
	if errors.Is(err, solana.ErrAccountNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(bal.Amount, 10, 64)
}

// broadcast resolves the owner keys, signs with the fee payer and sends once.
func (s *SolanaSweepService) broadcast(ctx context.Context, params solana.SweepParams, items []sweepItem) (solana.Sent, solana.PublicKey, error) {
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
			return solana.Sent{}, solana.PublicKey{}, err
		}
		ata, _ := solana.ParsePublicKey(it.acct.TokenAccount)
		priv, _, err := s.keys.PrivateKeyForAddress(it.acct.OwnerAddress)
		if err != nil {
			return solana.Sent{}, solana.PublicKey{}, fmt.Errorf("key for %s: %w", it.acct.OwnerAddress, err)
		}
		signer, err := solana.NewEd25519Signer(priv)
		zeroBytes(priv)
		if err != nil {
			return solana.Sent{}, solana.PublicKey{}, err
		}
		owners = append(owners, signer)
		signers = append(signers, signer)
		params.Items = append(params.Items, solana.SweepItem{Owner: owner, TokenAccount: ata, Amount: it.amount})
	}
	var hotATA solana.PublicKey
	build := func(blockhash string) (solana.Message, error) {
		ixs, ata, err := solana.SweepInstructions(params)
		if err != nil {
			return solana.Message{}, err
		}
		hotATA = ata
		bh, err := solana.ParsePublicKey(blockhash)
		if err != nil {
			return solana.Message{}, err
		}
		return solana.CompileMessage(params.FeePayer, bh, ixs)
	}
	sent, err := solana.SendOnce(ctx, s.client, build, signers)
	return sent, hotATA, err
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

// explainDrain looks for one of our sweep signatures in the account's recent history; when none
// explains the empty account, an anomaly is recorded and the account is set aside for an operator.
func (s *SolanaSweepService) explainDrain(ctx context.Context, acct *models.SolanaDepositAccount) {
	sigs, err := s.client.GetSignaturesForAddress(ctx, acct.TokenAccount, "", 20, solana.CommitmentConfirmed)
	if err != nil {
		log.Printf("[solana sweep] %s holds nothing; history lookup failed: %v", acct.TokenAccount, err)
		return
	}
	for _, info := range sigs {
		var n int64
		s.db.Model(&models.SolanaSweepAttempt{}).Where("signature = ?", info.Signature).Count(&n)
		if n > 0 {
			log.Printf("[solana sweep] %s holds nothing; drained by tracked sweep %s", acct.TokenAccount, info.Signature)
			return
		}
	}
	last := ""
	if len(sigs) > 0 {
		last = sigs[0].Signature
	}
	log.Printf("[solana sweep] %s holds nothing and no sweep of ours explains it (last signature %s); anomaly recorded", acct.TokenAccount, last)
	_ = s.accounts.UpdateStatus(acct.ID, models.SolanaDepositAccountDrained)
	s.recordAnomaly(last, acct.TokenAccount, decimal.Zero, anomalyUnexplainedDrain+": token account empty without a booked sweep")
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

// TrackConfirmations follows every attempt of each pending Solana sweep.
func (s *SolanaSweepService) TrackConfirmations(ctx context.Context) (int, error) {
	sweeps, err := s.sweepRepo.ListByBlockchain(s.chain.ID)
	if err != nil {
		return 0, err
	}
	completed := 0
	for i := range sweeps {
		sw := &sweeps[i]
		if sw.Status != SweepStatusPending {
			continue
		}
		if ctx.Err() != nil {
			return completed, ctx.Err()
		}
		done, err := s.trackSweep(ctx, sw)
		if err != nil {
			log.Printf("[solana sweep] track sweep %d: %v", sw.ID, err)
			continue
		}
		if done {
			completed++
		}
	}
	return completed, nil
}

func (s *SolanaSweepService) trackSweep(ctx context.Context, sw *models.Sweep) (bool, error) {
	var attempts []models.SolanaSweepAttempt
	if err := s.db.Where("sweep_id = ?", sw.ID).Order("id").Find(&attempts).Error; err != nil {
		return false, err
	}
	if len(attempts) == 0 {
		return false, nil
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
		s.markConfirming(sw.ID)
		return false, nil
	}
	// Every attempt unknown: only the finalized block height past the latest attempt's validity,
	// with the signature absent at finalized on more than one node, counts as expiry.
	latest := &attempts[len(attempts)-1]
	if latest.Status == models.SolanaSweepAttemptFailed {
		return false, s.rebuildOrFail(ctx, sw, attempts)
	}
	height, err := s.client.GetBlockHeight(ctx, solana.CommitmentFinalized)
	if err != nil {
		return false, err
	}
	if height <= latest.LastValidBlockHeight {
		return false, nil
	}
	tx, err := s.client.GetTransactionFromNodes(ctx, latest.Signature, solana.CommitmentFinalized, max(2, s.cfg.FetchNodes))
	if err != nil || tx != nil {
		return false, err
	}
	if err := s.db.Model(latest).Update("status", models.SolanaSweepAttemptExpired).Error; err != nil {
		return false, err
	}
	latest.Status = models.SolanaSweepAttemptExpired
	return false, s.rebuildOrFail(ctx, sw, attempts)
}

func (s *SolanaSweepService) markConfirming(sweepID uint) {
	s.db.Model(&models.SweepTransaction{}).Where("sweep_id = ? AND status = ?", sweepID, SweepTxStatusBroadcast).Update("status", SweepTxStatusConfirming)
}

// rebuildOrFail sends a fresh attempt while every account still holds its balance and the budget
// allows; otherwise the sweep fails and releases exactly its deposits.
func (s *SolanaSweepService) rebuildOrFail(ctx context.Context, sw *models.Sweep, attempts []models.SolanaSweepAttempt) error {
	if len(attempts) >= s.cfg.MaxAttempts {
		return s.failSweep(sw, "no attempt landed in "+strconv.Itoa(len(attempts))+" broadcasts")
	}
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
		amount, err := s.balance(ctx, txs[i].FromAddress)
		if err != nil {
			return err
		}
		if amount == 0 {
			// An earlier attempt may have landed on a node we have not asked yet; wait, never resend.
			log.Printf("[solana sweep] sweep %d: %s is empty while no attempt is finalized; waiting", sw.ID, txs[i].FromAddress)
			return nil
		}
		items = append(items, sweepItem{acct: acct, amount: amount})
	}
	sent, _, err := s.broadcast(ctx, params, items)
	if err != nil {
		return fmt.Errorf("rebuild: %w", err)
	}
	if err := s.db.Create(&models.SolanaSweepAttempt{SweepID: sw.ID, Signature: sent.Signature, Blockhash: sent.Blockhash,
		LastValidBlockHeight: sent.LastValidBlockHeight, Status: models.SolanaSweepAttemptSent}).Error; err != nil {
		return fmt.Errorf("record attempt %s: %w", sent.Signature, err)
	}
	s.db.Model(&models.SweepTransaction{}).Where("sweep_id = ?", sw.ID).Update("tx_hash", sent.Signature)
	log.Printf("[solana sweep] sweep %d rebuilt as %s (attempt %d)", sw.ID, sent.Signature, len(attempts)+1)
	return nil
}

// book completes the sweep from the one attempt that finalized: sweep transactions confirmed with
// their fee share, the sweep journal, rent, the deposits swept, accounts closed, and a reconciliation
// of the hot ATA's balance delta against the recorded amounts.
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
		if err := s.sweepSvc.completeIn(ctx, dbTx, sw.ID, total, fee, bcID); err != nil {
			return err
		}
		return s.recordRent(ctx, dbTx, sw.ID, bcID, reclaimed, funded)
	})
	if err != nil {
		return err
	}
	for _, t := range txs {
		if acct, err := s.accounts.GetByTokenAccount(t.FromAddress); err == nil && s.cfg.CloseAccounts {
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

// failSweep releases exactly the deposits this sweep claimed and returns its accounts to watching.
func (s *SolanaSweepService) failSweep(sw *models.Sweep, reason string) error {
	log.Printf("[solana sweep] sweep %d %s; releasing its deposits", sw.ID, reason)
	var links []models.SolanaSweepDeposit
	if err := s.db.Where("sweep_id = ?", sw.ID).Find(&links).Error; err != nil {
		return err
	}
	for _, l := range links {
		if err := s.db.Model(&models.Deposit{}).Where("id = ? AND status = ?", l.DepositID, models.DepositStatusSwept).
			Update("status", models.DepositStatusConfirmed).Error; err != nil {
			return err
		}
	}
	txs, _ := s.sweepTxs.ListBySweep(sw.ID)
	for _, t := range txs {
		_ = s.sweepTxs.UpdateStatus(t.ID, SweepTxStatusFailed)
		if acct, err := s.accounts.GetByTokenAccount(t.FromAddress); err == nil && acct.Status == models.SolanaDepositAccountClosed {
			_ = s.accounts.UpdateStatus(acct.ID, models.SolanaDepositAccountWatching)
		}
	}
	s.db.Model(&models.SolanaSweepAttempt{}).Where("sweep_id = ? AND status = ?", sw.ID, models.SolanaSweepAttemptSent).Update("status", models.SolanaSweepAttemptExpired)
	return s.sweepSvc.MarkFailed(sw.ID)
}
