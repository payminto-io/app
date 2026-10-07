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
	// DropGrace is how long an unseen sweep signature may stay unknown before the sweep is failed and retried.
	DropGrace time.Duration
	Send      solana.SendOptions
}

func (c SolanaSweepConfig) withDefaults() SolanaSweepConfig {
	if c.BatchSize <= 0 || c.BatchSize > solana.DefaultSweepBatchSize {
		c.BatchSize = solana.DefaultSweepBatchSize
	}
	if c.ComputeUnitLimit == 0 {
		c.ComputeUnitLimit = 120_000
	}
	if c.DropGrace <= 0 {
		c.DropGrace = 5 * time.Minute
	}
	return c
}

// SolanaSweepService drains confirmed SPL deposits into the hot wallet's ATA with a sponsored fee
// payer, then completes sweeps once finalized and books gas in SOL.SOLANA.
type SolanaSweepService struct {
	db         *gorm.DB
	client     *solana.Client
	chain      *models.Blockchain
	deposits   repository.DepositRepository
	accounts   repository.SolanaDepositAccountRepository
	currencies repository.BlockchainCurrencyRepository
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

// NewSolanaSweepService wires the sweeper; journal may be nil (rent reclaim then goes unbooked and is logged).
func NewSolanaSweepService(
	db *gorm.DB,
	client *solana.Client,
	chain *models.Blockchain,
	deposits repository.DepositRepository,
	accounts repository.SolanaDepositAccountRepository,
	currencies repository.BlockchainCurrencyRepository,
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
		db: db, client: client, chain: chain, deposits: deposits, accounts: accounts, currencies: currencies,
		sweepRepo: sweepRepo, sweepTxs: sweepTxs, sweepSvc: sweepSvc, sweepTxSvc: sweepTxSvc, keys: keys,
		feePayer: feePayer, hotWallet: hotWallet, journal: journal, cfg: cfg.withDefaults(), now: time.Now,
	}
}

type sweepGroup struct {
	tokenAccount string
	deposits     []models.Deposit
}

// SweepConfirmed sweeps every confirmed Solana token deposit. Returns the number of sweep
// transactions broadcast. Per-batch errors are logged; the batch's deposits return to confirmed.
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

type sweepItem struct {
	group  *sweepGroup
	acct   *models.SolanaDepositAccount
	amount uint64
	signer solana.Ed25519Signer
}

func (s *SolanaSweepService) sweepBatch(ctx context.Context, bcID uint, groups []*sweepGroup) (bool, error) {
	bc, err := s.currencies.GetByID(bcID)
	if err != nil {
		return false, err
	}
	mint, err := solana.ParsePublicKey(bc.Address)
	if err != nil {
		return false, fmt.Errorf("mint %q: %w", bc.Address, err)
	}
	program, err := solana.TokenProgramFor(bc.Standard)
	if err != nil {
		return false, err
	}
	decimals := uint8(bc.WalletPrecision)

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
			s.release(g)
		}
	}

	var items []sweepItem
	for _, g := range claimed {
		acct, err := s.accounts.GetByTokenAccount(g.tokenAccount)
		if err != nil {
			release()
			return false, fmt.Errorf("solana deposit account %s: %w", g.tokenAccount, err)
		}
		bal, err := s.client.GetTokenAccountBalance(ctx, g.tokenAccount, solana.CommitmentConfirmed)
		if errors.Is(err, solana.ErrAccountNotFound) || (err == nil && bal.Amount == "0") {
			// Already drained (manually or by an earlier sweep); nothing on chain to move.
			log.Printf("[solana sweep] %s holds nothing; marking its deposits swept", g.tokenAccount)
			_ = s.accounts.UpdateStatus(acct.ID, models.SolanaDepositAccountClosed)
			continue
		}
		if err != nil {
			release()
			return false, fmt.Errorf("balance of %s: %w", g.tokenAccount, err)
		}
		amount, err := strconv.ParseUint(bal.Amount, 10, 64)
		if err != nil {
			release()
			return false, fmt.Errorf("balance of %s: %w", g.tokenAccount, err)
		}
		items = append(items, sweepItem{group: g, acct: acct, amount: amount})
	}
	if len(items) == 0 {
		return false, nil
	}

	params := solana.SweepParams{
		FeePayer: s.feePayer.PublicKey(), HotWalletOwner: s.hotWallet, Mint: mint, TokenProgram: program, Decimals: decimals,
		ComputeUnitLimit: s.cfg.ComputeUnitLimit, PriorityFeeMicroLamports: s.cfg.PriorityFeeMicroLamports, CloseAccounts: s.cfg.CloseAccounts,
	}
	signers := []solana.Signer{s.feePayer}
	for i := range items {
		it := &items[i]
		owner, err := solana.ParsePublicKey(it.acct.OwnerAddress)
		if err != nil {
			release()
			return false, err
		}
		ata, _ := solana.ParsePublicKey(it.acct.TokenAccount)
		priv, _, err := s.keys.PrivateKeyForAddress(it.acct.OwnerAddress)
		if err != nil {
			release()
			return false, fmt.Errorf("key for %s: %w", it.acct.OwnerAddress, err)
		}
		it.signer, err = solana.NewEd25519Signer(priv)
		zeroBytes(priv)
		if err != nil {
			release()
			return false, err
		}
		signers = append(signers, it.signer)
		params.Items = append(params.Items, solana.SweepItem{Owner: owner, TokenAccount: ata, Amount: it.amount})
	}
	defer func() {
		for i := range items {
			items[i].signer.Zero()
		}
	}()

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
	res, err := solana.SendAndConfirm(ctx, s.client, build, signers, s.cfg.Send)
	if err != nil && res.Signature == "" {
		release()
		return false, fmt.Errorf("broadcast: %w", err)
	}
	if err != nil && !errors.Is(err, solana.ErrNotLanded) {
		// Sent but expired or failed before landing: nothing moved, retry next round.
		release()
		return false, fmt.Errorf("sweep %s did not land: %w", res.Signature, err)
	}

	sweep, err := s.sweepSvc.CreateSweep(s.chain.ID)
	if err != nil {
		return true, fmt.Errorf("sweep %s broadcast but batch row failed: %w", res.Signature, err)
	}
	for _, it := range items {
		amount := decimal.NewFromBigInt(new(big.Int).SetUint64(it.amount), -int32(decimals))
		if _, err := s.sweepTxSvc.RecordBroadcastSweep(sweep.ID, bc.ID, it.acct.TokenAccount, hotATA.String(), res.Signature, amount, decimal.Zero); err != nil {
			log.Printf("[solana sweep] record sweep tx for %s (tx %s on chain): %v", it.acct.TokenAccount, res.Signature, err)
		}
		if s.cfg.CloseAccounts {
			_ = s.accounts.UpdateStatus(it.acct.ID, models.SolanaDepositAccountClosed)
		}
	}
	log.Printf("[solana sweep] sent %s: %d accounts of %s to %s (attempts=%d landed=%v)", res.Signature, len(items), bc.CurrencyCode, hotATA, res.Attempts, res.Landed)
	return true, nil
}

func (s *SolanaSweepService) claim(g *sweepGroup) bool {
	var won []models.Deposit
	for _, d := range g.deposits {
		n, err := s.deposits.ClaimForSweep(d.ID)
		if err != nil || n == 0 {
			for _, w := range won {
				_ = s.deposits.UpdateStatus(w.ID, models.DepositStatusConfirmed)
			}
			return false
		}
		won = append(won, d)
	}
	return true
}

func (s *SolanaSweepService) release(g *sweepGroup) {
	for _, d := range g.deposits {
		if err := s.deposits.UpdateStatus(d.ID, models.DepositStatusConfirmed); err != nil {
			log.Printf("[solana sweep] release deposit %d: %v", d.ID, err)
		}
	}
}

// TrackConfirmations completes pending Solana sweeps once finalized: marks their transactions
// confirmed with their share of the fee, posts the sweep journal (token move plus SOL gas) and a
// rent-reclaim journal, or fails and releases the deposits when the transaction never landed.
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
	txs, err := s.sweepTxs.ListBySweep(sw.ID)
	if err != nil || len(txs) == 0 || txs[0].TxHash == "" {
		return false, err
	}
	sig := txs[0].TxHash
	statuses, err := s.client.GetSignatureStatuses(ctx, []string{sig})
	if err != nil {
		return false, err
	}
	var st *solana.SignatureStatus
	if len(statuses) > 0 {
		st = statuses[0]
	}
	switch {
	case st == nil:
		if s.now().Sub(sw.CreatedAt) < s.cfg.DropGrace {
			return false, nil
		}
		tx, err := s.client.GetTransaction(ctx, sig, solana.CommitmentConfirmed)
		if err != nil || tx != nil {
			return false, err
		}
		return false, s.failSweep(sw, txs, "dropped before landing")
	case st.Failed():
		return false, s.failSweep(sw, txs, "failed on chain: "+string(st.Err))
	case st.ConfirmationStatus != solana.CommitmentFinalized:
		for i := range txs {
			if txs[i].Status == SweepTxStatusBroadcast {
				_ = s.sweepTxs.UpdateStatus(txs[i].ID, SweepTxStatusConfirming)
			}
		}
		return false, nil
	}
	tx, err := s.client.GetTransaction(ctx, sig, solana.CommitmentFinalized)
	if err != nil {
		return false, err
	}
	if tx == nil || tx.Meta == nil {
		return false, fmt.Errorf("finalized %s has no transaction record", sig)
	}
	fee := solana.LamportsToSOL(tx.Meta.Fee)
	rent := solana.FeePayerRentRefund(tx)
	total := decimal.Zero
	for _, t := range txs {
		total = total.Add(t.Amount)
	}
	// Gas is per transaction; rows carry an equal share with the remainder on the first.
	share := fee.Div(decimal.NewFromInt(int64(len(txs)))).Truncate(9)
	bcID := txs[0].BlockchainCurrencyID
	err = s.db.WithContext(ctx).Transaction(func(dbTx *gorm.DB) error {
		for i := range txs {
			g := share
			if i == 0 {
				g = fee.Sub(share.Mul(decimal.NewFromInt(int64(len(txs) - 1))))
			}
			if err := dbTx.Model(&models.SweepTransaction{}).Where("id = ?", txs[i].ID).
				Updates(map[string]any{"status": SweepTxStatusConfirmed, "gas_fee": g}).Error; err != nil {
				return err
			}
		}
		if err := s.sweepSvc.completeIn(ctx, dbTx, sw.ID, total, fee, bcID); err != nil {
			return err
		}
		return s.recordRentReclaim(ctx, dbTx, sw.ID, bcID, rent)
	})
	if err != nil {
		return false, err
	}
	log.Printf("[solana sweep] sweep %d finalized (%s): %s moved, fee %s SOL, rent reclaimed %d lamports", sw.ID, sig, total, fee, rent)
	return true, nil
}

// recordRentReclaim books the lamports closing deposit ATAs returned to the fee payer.
func (s *SolanaSweepService) recordRentReclaim(ctx context.Context, tx *gorm.DB, sweepID, bcID uint, lamports uint64) error {
	if lamports == 0 {
		return nil
	}
	if s.journal == nil {
		log.Printf("[solana sweep] sweep %d reclaimed %d lamports of rent; no journal wired to book it", sweepID, lamports)
		return nil
	}
	native, err := nativeAssetFor(tx, bcID)
	if err != nil {
		return err
	}
	amount := solana.LamportsToSOL(lamports)
	id := strconv.FormatUint(uint64(sweepID), 10)
	_, err = s.journal.PostIn(ctx, tx, ledger.Journal{
		Kind:           ledger.KindAdjustment,
		Reference:      ledger.Reference{Type: "solana_rent_reclaim", ID: id},
		IdempotencyKey: "payminto:solana_rent_reclaim:" + id,
		Metadata:       map[string]any{"blockchain_currency_id": bcID, "lamports": lamports},
		Lines: []ledger.Line{
			{Account: legacyAccount("crypto_assets", native, ledger.KindAsset), Amount: amount},
			{Account: legacyAccount("rent_reclaimed", native, ledger.KindIncome), Amount: amount.Neg()},
		},
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

func (s *SolanaSweepService) failSweep(sw *models.Sweep, txs []models.SweepTransaction, reason string) error {
	log.Printf("[solana sweep] sweep %d %s; releasing its deposits", sw.ID, reason)
	for _, t := range txs {
		_ = s.sweepTxs.UpdateStatus(t.ID, SweepTxStatusFailed)
		res := s.db.Model(&models.Deposit{}).
			Where("to_address = ? AND blockchain_currency_id = ? AND status = ?", t.FromAddress, t.BlockchainCurrencyID, models.DepositStatusSwept).
			Update("status", models.DepositStatusConfirmed)
		if res.Error != nil {
			return res.Error
		}
		if acct, err := s.accounts.GetByTokenAccount(t.FromAddress); err == nil {
			_ = s.accounts.UpdateStatus(acct.ID, models.SolanaDepositAccountWatching)
		}
	}
	return s.sweepSvc.MarkFailed(sw.ID)
}
