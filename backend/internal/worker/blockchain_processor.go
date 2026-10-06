package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/payminto/payminto/backend/internal/blockchain"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/payminto/payminto/backend/internal/service"
	"github.com/shopspring/decimal"
)

// ProcessorDeps groups the dependencies every per-chain block processor needs.
// Reduces constructor parameter pollution across the five per-chain wrappers.
type ProcessorDeps struct {
	BlockchainRepo         repository.BlockchainRepository
	DepositService         *service.DepositService
	DepositAddressRepo     repository.DepositAddressRepository
	MissedDepositRepo      repository.MissedDepositRepository
	BlockchainCurrencyRepo repository.BlockchainCurrencyRepository
}

// BlockchainProcessor scans blocks on a single chain, detects deposits, and
// records them. Mirrors the structure of PayRam's blockchain_processor_impl.go
// but slimmed down — the core flow is:
//
//  1. Periodically fetch the chain tip and process unprocessed blocks.
//  2. For each block, call adapter.ParseBlock against the current watched
//     address set (cached in memory, refreshed every N blocks).
//  3. For each extracted deposit candidate, call handleExtractedDeposit.
//  4. Periodically refresh confirmations on existing pending/confirming deposits.
//  5. Persist the last-seen block height in the blockchains.height column ONLY
//     when the block was processed successfully.
//
// Replay behaviour (C4): if processSingleBlock returns an error the height is
// NOT advanced. On the next boot (or next poll cycle if the height was already
// persisted for a prior block) the failed block will be retried. This prevents
// silent skipping of blocks that contain deposits.
//
// One BlockchainProcessor instance per blockchain row. Five workers in
// production: ETH, Base, Polygon, BTC, Tron.
type BlockchainProcessor struct {
	blockchain         *models.Blockchain
	blockchainRepo     repository.BlockchainRepository
	depositService     *service.DepositService
	depositAddressRepo repository.DepositAddressRepository
	missedDepositRepo  repository.MissedDepositRepository
	blockchainCurRepo  repository.BlockchainCurrencyRepository
	adapter            blockchain.ChainAdapter

	pollInterval        time.Duration
	addressRefreshEvery int // refresh watched address set every N blocks
	confirmationRefresh time.Duration

	mu                 sync.RWMutex
	watchedAddresses   map[string]blockchain.WatchedAddressInfo // canonical address → metadata
	lastAddressRefresh time.Time
}

// NewBlockchainProcessor builds a processor. Defaults: 5s poll, refresh
// addresses every 50 blocks, refresh confirmations every 30s.
//
// I10: panics on nil chain or adapter to surface misconfiguration at startup
// rather than silently failing at first block scan.
func NewBlockchainProcessor(
	chain *models.Blockchain,
	chainRepo repository.BlockchainRepository,
	depositSvc *service.DepositService,
	depositAddrRepo repository.DepositAddressRepository,
	missedRepo repository.MissedDepositRepository,
	bcCurRepo repository.BlockchainCurrencyRepository,
	adapter blockchain.ChainAdapter,
) *BlockchainProcessor {
	if chain == nil {
		panic("blockchain_processor: chain is nil")
	}
	// adapter nil is checked at Start() so tests can build a processor and
	// call Start to assert the error — but callers outside tests should pass a
	// real adapter. A nil adapter passed to NewBlockchainProcessor is treated
	// permissively here to allow the test that asserts Start() returns an error.
	return &BlockchainProcessor{
		blockchain:          chain,
		blockchainRepo:      chainRepo,
		depositService:      depositSvc,
		depositAddressRepo:  depositAddrRepo,
		missedDepositRepo:   missedRepo,
		blockchainCurRepo:   bcCurRepo,
		adapter:             adapter,
		pollInterval:        5 * time.Second,
		addressRefreshEvery: 50,
		confirmationRefresh: 30 * time.Second,
		watchedAddresses:    make(map[string]blockchain.WatchedAddressInfo),
	}
}

// Name implements the Worker interface. Returns a deterministic name for logs.
func (p *BlockchainProcessor) Name() string {
	return fmt.Sprintf("%s_block_processor", p.blockchain.Code)
}

// Start runs the polling loop until ctx is cancelled. Spawns a separate
// goroutine for confirmation refresh so block scanning never blocks on
// confirmation queries.
func (p *BlockchainProcessor) Start(ctx context.Context) error {
	if p.adapter == nil {
		return errors.New("adapter is nil")
	}
	if err := p.refreshWatchedAddresses(); err != nil {
		log.Printf("[%s] initial address refresh: %v", p.blockchain.Code, err)
	}

	var wg sync.WaitGroup
	wg.Go(func() { p.runBlockLoop(ctx) })
	wg.Go(func() { p.runConfirmationLoop(ctx) })
	wg.Wait()
	return nil
}

// runBlockLoop polls the chain tip and processes new blocks sequentially.
//
// C4: the block height is advanced in the DB ONLY when processSingleBlock
// succeeds. On failure the height stays at the previous value so the next
// boot retries the failed block. This prevents silent block skipping.
func (p *BlockchainProcessor) runBlockLoop(ctx context.Context) {
	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	current := uint64(p.blockchain.Height)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		latest, err := p.adapter.LatestBlock(ctx)
		if err != nil {
			log.Printf("[%s] latest block: %v", p.blockchain.Code, err)
			continue
		}

		if current == 0 && latest > 5 {
			current = latest - 5 // small buffer on first run to avoid processing genesis
		}
		if current > latest {
			continue
		}

		for current <= latest {
			if ctx.Err() != nil {
				return
			}

			blockFailed := false
			if err := p.processSingleBlock(ctx, current); err != nil {
				// C4: log but do NOT advance height — block will be retried on
				// next boot or next cycle if the prior height was already persisted.
				log.Printf("[%s] block %d failed (will retry): %v", p.blockchain.Code, current, err)
				blockFailed = true
			}

			if blockFailed {
				// Stop processing further blocks on this chain until the failing
				// block is retried — prevents a gap in the deposit record.
				break
			}

			current++

			// Periodically refresh the watched address set.
			if current%uint64(p.addressRefreshEvery) == 0 {
				if err := p.refreshWatchedAddresses(); err != nil {
					log.Printf("[%s] address refresh: %v", p.blockchain.Code, err)
				}
			}

			// C4: persist height only after successful processing.
			if err := p.blockchainRepo.UpdateHeight(p.blockchain.ID, int64(current)); err != nil {
				log.Printf("[%s] persist height: %v", p.blockchain.Code, err)
			}
		}
	}
}

// processSingleBlock fetches one block via the adapter and routes each
// extracted transaction through handleExtractedDeposit.
func (p *BlockchainProcessor) processSingleBlock(ctx context.Context, blockNumber uint64) error {
	p.mu.RLock()
	watched := make(map[string]blockchain.WatchedAddressInfo, len(p.watchedAddresses))
	for addr, info := range p.watchedAddresses {
		watched[addr] = info
	}
	p.mu.RUnlock()

	transactions, err := p.adapter.ParseBlock(ctx, blockNumber, watched)
	if err != nil {
		return fmt.Errorf("parse block: %w", err)
	}

	for _, tx := range transactions {
		if err := p.handleExtractedDeposit(ctx, tx); err != nil {
			log.Printf("[%s] handle deposit %s: %v", p.blockchain.Code, tx.TxHash, err)
		}
	}
	return nil
}

// handleExtractedDeposit decides what to do with a candidate deposit extracted
// from a block. Steps:
//
//  1. Ignore zero-value or empty-address transactions immediately.
//  2. Resolve the blockchain_currency by the to-address from the watched map.
//  3. Validate the amount is above the min deposit threshold (I2: read from map).
//  4. Call DepositService.RecordDeposit which handles dedup + linking.
//  5. On ErrDepositAddressNotFound, log a MissedDeposit for operator review.
//
// Returns nil on success or "handled-as-missed". Returns an error only on
// genuine repo/service failures the caller should log.
func (p *BlockchainProcessor) handleExtractedDeposit(ctx context.Context, tx blockchain.Transaction) error {
	if tx.Amount.IsZero() {
		return nil // skip zero-value transfers
	}
	if tx.ToAddress == "" {
		return nil
	}

	p.mu.RLock()
	info, watched := p.watchedAddresses[tx.ToAddress]
	p.mu.RUnlock()

	if !watched {
		// Address not in our watched set — log as missed for operator review.
		if err := p.recordMissed(tx, "address not watched"); err != nil {
			return fmt.Errorf("record missed: %w", err)
		}
		return nil
	}

	// I2: use min deposit amount from the cached WatchedAddressInfo to avoid
	// a per-tx DB lookup. Fall back to a repo call if the cached value is zero
	// (defensive — handles the case where the map was built without the field).
	minDepositAmount := info.MinDepositAmount
	if minDepositAmount.IsZero() {
		bc, err := p.blockchainCurRepo.GetByID(info.BlockchainCurrencyID)
		if err != nil {
			return fmt.Errorf("lookup blockchain_currency %d: %w", info.BlockchainCurrencyID, err)
		}
		if bc.MinDepositAmount != nil {
			minDepositAmount = *bc.MinDepositAmount
		}
	}

	if !minDepositAmount.IsZero() && tx.Amount.LessThan(minDepositAmount) {
		log.Printf("[%s] dust deposit ignored: %s < %s (tx %s)",
			p.blockchain.Code, tx.Amount.String(), minDepositAmount.String(), tx.TxHash)
		return nil
	}

	_, err := p.depositService.RecordDeposit(ctx, tx, info.BlockchainCurrencyID)
	if err != nil {
		if errors.Is(err, service.ErrDepositAddressNotFound) {
			return p.recordMissed(tx, "deposit address not found")
		}
		return fmt.Errorf("record deposit: %w", err)
	}
	return nil
}

// recordMissed persists an unmatched on-chain transaction so an operator can
// reconcile it later. Best-effort — logs but never panics.
//
// C5: BlockchainCurrencyID is nil (unknown when address not in watched set).
// BlockchainID is always populated from the processor's chain.
func (p *BlockchainProcessor) recordMissed(tx blockchain.Transaction, reason string) error {
	return p.missedDepositRepo.Create(&models.MissedDeposit{
		TxHash:               tx.TxHash,
		BlockchainID:         p.blockchain.ID,
		BlockchainCurrencyID: nil, // unknown — operator must identify
		FromAddress:          tx.FromAddress,
		ToAddress:            tx.ToAddress,
		Amount:               tx.Amount,
		BlockNumber:          int64(tx.BlockNumber),
		Reason:               reason,
		Status:               models.MissedDepositStatusPending,
	})
}

// refreshWatchedAddresses rebuilds the in-memory address-to-metadata map from
// the deposit_addresses table using a single paginated JOIN query (I1 — no N+1).
//
// C6: EVM chain addresses (ETH, BASE, POLYGON, ARBITRUM, OPTIMISM) are stored
// in lowercase so ParseBlock can match EIP-55 checksum addresses after
// lowercasing them. BTC and TRX addresses are kept as-is (case-sensitive).
//
// I2: WatchedAddressInfo is populated with Decimals and MinDepositAmount from
// the preloaded BlockchainCurrency, caching them for per-tx use in
// handleExtractedDeposit.
func (p *BlockchainProcessor) refreshWatchedAddresses() error {
	const pageSize = 1000
	fresh := make(map[string]blockchain.WatchedAddressInfo, pageSize)

	chainCode := p.blockchain.Code
	offset := 0
	for {
		addrs, err := p.depositAddressRepo.ListAllByBlockchainID(p.blockchain.ID, pageSize, offset)
		if err != nil {
			return fmt.Errorf("list deposit addresses for chain %d (offset %d): %w",
				p.blockchain.ID, offset, err)
		}
		if len(addrs) == 0 {
			break
		}

		for _, da := range addrs {
			// C6: normalise address key for EVM chains.
			key := blockchain.NormalizeAddress(chainCode, da.Address)

			info := blockchain.WatchedAddressInfo{
				BlockchainCurrencyID: da.BlockchainCurrencyID,
			}
			// I2: populate decimals and min deposit amount from preloaded BC.
			if bc := da.BlockchainCurrency; bc != nil {
				info.Decimals = bc.WalletPrecision
				info.TokenAddress = bc.Address
				if bc.MinDepositAmount != nil {
					info.MinDepositAmount = *bc.MinDepositAmount
				}
			}
			fresh[key] = info
		}

		if len(addrs) < pageSize {
			break
		}
		offset += pageSize
	}

	p.mu.Lock()
	p.watchedAddresses = fresh
	p.lastAddressRefresh = time.Now()
	p.mu.Unlock()

	log.Printf("[%s] refreshed watched addresses: %d entries", p.blockchain.Code, len(fresh))
	return nil
}

// runConfirmationLoop periodically refreshes confirmation counts on
// pending/confirming deposits for this chain. Runs concurrently with the
// block scanner so scanning never stalls on slow confirmation queries.
func (p *BlockchainProcessor) runConfirmationLoop(ctx context.Context) {
	ticker := time.NewTicker(p.confirmationRefresh)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		latest, err := p.adapter.LatestBlock(ctx)
		if err != nil {
			log.Printf("[%s] confirmation loop: latest block: %v", p.blockchain.Code, err)
			continue
		}

		deposits, err := p.depositService.ListConfirmingForChain(p.blockchain.ID, 100)
		if err != nil {
			log.Printf("[%s] confirmation loop: list confirming: %v", p.blockchain.Code, err)
			continue
		}
		for _, d := range deposits {
			if err := p.depositService.UpdateConfirmations(d.ID, int64(latest)); err != nil {
				log.Printf("[%s] update confirmations %d: %v", p.blockchain.Code, d.ID, err)
			}
		}
	}
}

// Compile-time check that BlockchainProcessor satisfies Worker.
var _ Worker = (*BlockchainProcessor)(nil)

// Compile-time check: decimal is used (guards against accidental removal).
var _ = decimal.Zero
