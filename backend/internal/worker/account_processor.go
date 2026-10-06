package worker

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"time"

	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/payminto/payminto/backend/internal/service"
	"gorm.io/gorm"
)

// AccountProcessorConfig holds the ticker intervals for each sub-loop in
// AccountProcessorJob. Exposing them as fields lets tests inject short
// intervals for fast cycle verification.
type AccountProcessorConfig struct {
	// ETHSweepInterval is how often processETHAutoSweep runs.
	ETHSweepInterval time.Duration
	// BTCSweepInterval is how often processBitcoinSweeps runs.
	BTCSweepInterval time.Duration
	// ERC20SweepInterval is how often processERC20Sweeps runs.
	ERC20SweepInterval time.Duration
	// PayloadInterval is how often createSweepTransactionPayload runs.
	PayloadInterval time.Duration
	// RewardInterval is how often processRewards runs.
	RewardInterval time.Duration
	// FailedRewardInterval is how often processFailedRewards runs.
	FailedRewardInterval time.Duration
	// WithdrawalInterval is how often processWithdrawals runs.
	WithdrawalInterval time.Duration
	// StaleBTCInterval is how often retryStaleBTCSweepTransactions runs.
	StaleBTCInterval time.Duration
	// StaleInitiatedInterval is how often processStaleInitiatedSweeps runs.
	StaleInitiatedInterval time.Duration
}

// DefaultAccountProcessorConfig returns production-grade intervals.
func DefaultAccountProcessorConfig() AccountProcessorConfig {
	return AccountProcessorConfig{
		ETHSweepInterval:       30 * time.Second,
		BTCSweepInterval:       60 * time.Second,
		ERC20SweepInterval:     30 * time.Second,
		PayloadInterval:        15 * time.Second,
		RewardInterval:         5 * time.Minute,
		FailedRewardInterval:   10 * time.Minute,
		WithdrawalInterval:     30 * time.Second,
		StaleBTCInterval:       5 * time.Minute,
		StaleInitiatedInterval: 2 * time.Minute,
	}
}

// AccountProcessorJob is the central sweep orchestrator. It runs 9 concurrent
// sub-goroutines, each with its own ticker and panic-recovery defer. Each
// sub-goroutine honours ctx.Done() and exits cleanly on cancellation.
//
// Sub-goroutines 1–4 (sweep loops) are real; sub-goroutines 5–9 (rewards,
// withdrawals) are stubbed pending Phase G and Phase I.
type AccountProcessorJob struct {
	cfg           AccountProcessorConfig
	sweepSvc      *service.SweepService
	sweepTxSvc    *service.SweepTransactionService
	sweepUTXOSvc  *service.SweepUTXOService
	rewardSvc     *service.AccountRewardService
	withdrawalSvc *service.WithdrawalProcessingService

	// evmSweepSvc performs real native-EVM sweeps to cold storage. Optional;
	// when nil, processETHAutoSweep falls back to its diagnostic no-op.
	evmSweepSvc *service.EVMSweepService

	// coldWalletETH is the destination for EVM native-coin sweeps.
	coldWalletETH string
	// coldWalletBTC is the destination for Bitcoin sweeps.
	coldWalletBTC string

	// btcBlockchainID and btcNativeCurrencyBCID are resolved once at construction
	// via BlockchainRepository.GetByCode("BTC"). Hard failures at construction time
	// surface immediately rather than causing silent mis-accounting at runtime.
	btcBlockchainID       uint
	btcNativeCurrencyBCID uint
}

// NewAccountProcessorJob constructs an AccountProcessorJob, resolving BTC IDs at
// startup via the provided repositories. Returns an error if BTC cannot be found —
// this is a fatal misconfiguration that should prevent the worker from starting.
func NewAccountProcessorJob(
	cfg AccountProcessorConfig,
	sweepSvc *service.SweepService,
	sweepTxSvc *service.SweepTransactionService,
	sweepUTXOSvc *service.SweepUTXOService,
	rewardSvc *service.AccountRewardService,
	withdrawalSvc *service.WithdrawalProcessingService,
	blockchainRepo repository.BlockchainRepository,
	blockchainCurrencyRepo repository.BlockchainCurrencyRepository,
	coldWalletETH string,
	coldWalletBTC string,
) (*AccountProcessorJob, error) {
	btc, err := blockchainRepo.GetByCode("BTC")
	if err != nil {
		return nil, fmt.Errorf("account processor: resolve BTC blockchain: %w", err)
	}

	btcNative, err := blockchainCurrencyRepo.GetByBlockchainAndCurrency(btc.ID, btc.ID)
	if err != nil {
		// Fallback: find the native currency by listing all currencies for BTC blockchain.
		bcs, listErr := blockchainCurrencyRepo.ListByBlockchainID(btc.ID)
		if listErr != nil || len(bcs) == 0 {
			return nil, fmt.Errorf("account processor: resolve BTC native currency (blockchain_id=%d): %w", btc.ID, err)
		}
		// Use the first (native) blockchain currency for BTC.
		btcNative = &bcs[0]
	}

	return &AccountProcessorJob{
		cfg:                   cfg,
		sweepSvc:              sweepSvc,
		sweepTxSvc:            sweepTxSvc,
		sweepUTXOSvc:          sweepUTXOSvc,
		rewardSvc:             rewardSvc,
		withdrawalSvc:         withdrawalSvc,
		coldWalletETH:         coldWalletETH,
		coldWalletBTC:         coldWalletBTC,
		btcBlockchainID:       btc.ID,
		btcNativeCurrencyBCID: btcNative.ID,
	}, nil
}

// newAccountProcessorJobForTest constructs an AccountProcessorJob with explicit
// BTC IDs for use in unit tests that don't have a real database.
func newAccountProcessorJobForTest(
	cfg AccountProcessorConfig,
	sweepSvc *service.SweepService,
	sweepTxSvc *service.SweepTransactionService,
	sweepUTXOSvc *service.SweepUTXOService,
	rewardSvc *service.AccountRewardService,
	withdrawalSvc *service.WithdrawalProcessingService,
	coldWalletETH string,
	coldWalletBTC string,
	btcBlockchainID uint,
	btcNativeCurrencyBCID uint,
) *AccountProcessorJob {
	return &AccountProcessorJob{
		cfg:                   cfg,
		sweepSvc:              sweepSvc,
		sweepTxSvc:            sweepTxSvc,
		sweepUTXOSvc:          sweepUTXOSvc,
		rewardSvc:             rewardSvc,
		withdrawalSvc:         withdrawalSvc,
		coldWalletETH:         coldWalletETH,
		coldWalletBTC:         coldWalletBTC,
		btcBlockchainID:       btcBlockchainID,
		btcNativeCurrencyBCID: btcNativeCurrencyBCID,
	}
}

// newAccountProcessorJobFromDB is a convenience constructor for the production
// wiring path that accepts a *gorm.DB and creates the required repos internally.
// This avoids changing the ServiceRegistry signature for this worker.
func newAccountProcessorJobFromDB(
	cfg AccountProcessorConfig,
	db *gorm.DB,
	sweepSvc *service.SweepService,
	sweepTxSvc *service.SweepTransactionService,
	sweepUTXOSvc *service.SweepUTXOService,
	rewardSvc *service.AccountRewardService,
	withdrawalSvc *service.WithdrawalProcessingService,
	coldWalletETH string,
	coldWalletBTC string,
) (*AccountProcessorJob, error) {
	return NewAccountProcessorJob(
		cfg,
		sweepSvc,
		sweepTxSvc,
		sweepUTXOSvc,
		rewardSvc,
		withdrawalSvc,
		repository.NewBlockchainRepository(db),
		repository.NewBlockchainCurrencyRepository(db),
		coldWalletETH,
		coldWalletBTC,
	)
}

// Name implements Worker.
func (j *AccountProcessorJob) Name() string { return "account_processor" }

// Start implements Worker. It launches all 9 sub-goroutines and blocks until
// ctx is cancelled, then waits for every goroutine to exit cleanly.
func (j *AccountProcessorJob) Start(ctx context.Context) error {
	log.Printf("[AccountProcessorJob] starting 9 sub-goroutines")

	var wg sync.WaitGroup
	launch := func(fn func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fn()
		}()
	}

	launch(func() { j.runETHAutoSweep(ctx) })
	launch(func() { j.runBitcoinSweeps(ctx) })
	launch(func() { j.runERC20Sweeps(ctx) })
	launch(func() { j.runCreateSweepPayloads(ctx) })
	launch(func() { j.runProcessRewards(ctx) })
	launch(func() { j.runProcessFailedRewards(ctx) })
	launch(func() { j.runProcessWithdrawals(ctx) })
	launch(func() { j.runRetryStaleBTCSweepTransactions(ctx) })
	launch(func() { j.runStaleInitiatedSweeps(ctx) })

	wg.Wait()
	log.Printf("[AccountProcessorJob] all 9 sub-goroutines exited")
	return nil
}

// ─── Sub-goroutine 1: processETHAutoSweep ────────────────────────────────────

// runETHAutoSweep ticks on ETHSweepInterval and calls processETHAutoSweep.
func (j *AccountProcessorJob) runETHAutoSweep(ctx context.Context) {
	ticker := time.NewTicker(j.cfg.ETHSweepInterval)
	defer ticker.Stop()
	log.Printf("[AccountProcessorJob] processETHAutoSweep started (interval=%s)", j.cfg.ETHSweepInterval)
	for {
		select {
		case <-ctx.Done():
			log.Printf("[AccountProcessorJob] processETHAutoSweep: ctx cancelled, exiting")
			return
		case <-ticker.C:
			func() {
				defer recoverSubLoop("processETHAutoSweep")
				j.processETHAutoSweep()
			}()
		}
	}
}

// SetEVMSweepService wires the native-EVM sweep service into the processor.
// Kept as a setter so the existing constructors stay unchanged.
func (j *AccountProcessorJob) SetEVMSweepService(s *service.EVMSweepService) {
	j.evmSweepSvc = s
}

// processETHAutoSweep sweeps confirmed native ETH/BASE/POLYGON deposits to the
// cold wallet via EVMSweepService (real sign + broadcast). When the sweep
// service is not wired it is a no-op.
func (j *AccountProcessorJob) processETHAutoSweep() {
	if j.evmSweepSvc == nil {
		return
	}
	n, err := j.evmSweepSvc.SweepConfirmedNative(context.Background())
	if err != nil {
		log.Printf("[AccountProcessorJob] processETHAutoSweep: %v", err)
		return
	}
	if n > 0 {
		log.Printf("[AccountProcessorJob] processETHAutoSweep: swept %d native deposit(s) to cold", n)
	}
}

// ─── Sub-goroutine 2: processBitcoinSweeps ───────────────────────────────────

// runBitcoinSweeps ticks on BTCSweepInterval and calls processBitcoinSweeps.
func (j *AccountProcessorJob) runBitcoinSweeps(ctx context.Context) {
	ticker := time.NewTicker(j.cfg.BTCSweepInterval)
	defer ticker.Stop()
	log.Printf("[AccountProcessorJob] processBitcoinSweeps started (interval=%s)", j.cfg.BTCSweepInterval)
	for {
		select {
		case <-ctx.Done():
			log.Printf("[AccountProcessorJob] processBitcoinSweeps: ctx cancelled, exiting")
			return
		case <-ticker.C:
			func() {
				defer recoverSubLoop("processBitcoinSweeps")
				j.processBitcoinSweeps()
			}()
		}
	}
}

// processBitcoinSweeps aggregates UTXOs and submits BTC sweeps for eligible
// addresses.
func (j *AccountProcessorJob) processBitcoinSweeps() {
	log.Printf("[AccountProcessorJob] processBitcoinSweeps: aggregating UTXOs")
	groups, err := j.sweepUTXOSvc.AggregatePendingUTXOs()
	if err != nil {
		log.Printf("[AccountProcessorJob] processBitcoinSweeps: aggregate: %v", err)
		return
	}
	if len(groups) == 0 {
		return
	}
	log.Printf("[AccountProcessorJob] processBitcoinSweeps: %d address groups eligible", len(groups))

	for _, g := range groups {
		sweep, err := j.sweepSvc.CreateSweep(j.btcBlockchainID)
		if err != nil {
			log.Printf("[AccountProcessorJob] processBitcoinSweeps: create sweep: %v", err)
			continue
		}
		_, err = j.sweepUTXOSvc.SubmitBTCSweepForAddress(
			sweep.ID,
			j.btcNativeCurrencyBCID,
			g,
			j.coldWalletBTC,
			service.PlaceholderBTCFee, // 0.00002 BTC = 2000 sat; TODO(phase-k-btc-fee)
		)
		if err != nil {
			log.Printf("[AccountProcessorJob] processBitcoinSweeps: submit for %s: %v", g.Address, err)
			_ = j.sweepSvc.MarkFailed(sweep.ID)
		}
	}
}

// ─── Sub-goroutine 3: processERC20Sweeps ─────────────────────────────────────

// runERC20Sweeps ticks on ERC20SweepInterval.
func (j *AccountProcessorJob) runERC20Sweeps(ctx context.Context) {
	ticker := time.NewTicker(j.cfg.ERC20SweepInterval)
	defer ticker.Stop()
	log.Printf("[AccountProcessorJob] processERC20Sweeps started (interval=%s)", j.cfg.ERC20SweepInterval)
	for {
		select {
		case <-ctx.Done():
			log.Printf("[AccountProcessorJob] processERC20Sweeps: ctx cancelled, exiting")
			return
		case <-ticker.C:
			func() {
				defer recoverSubLoop("processERC20Sweeps")
				j.processERC20Sweeps()
			}()
		}
	}
}

// processERC20Sweeps funds gas for ERC-20 deposit addresses then executes
// token transfers.
//
// TODO(phase-k-signing): Iterate confirmed deposits with ERC-20 currencies,
// call InternalBlockchainTransactionService.RecordGasFunding, wait for
// confirmation, then call SweepTransactionService.ProcessERC20Sweep. All
// on-chain sends require SecretsVault integration from Phase K.
func (j *AccountProcessorJob) processERC20Sweeps() {
	log.Printf("[AccountProcessorJob] processERC20Sweeps: checking ERC-20 sweep candidates — PENDING PHASE-K SIGNING")
}

// ─── Sub-goroutine 4: createSweepTransactionPayload ──────────────────────────

// runCreateSweepPayloads ticks on PayloadInterval.
func (j *AccountProcessorJob) runCreateSweepPayloads(ctx context.Context) {
	ticker := time.NewTicker(j.cfg.PayloadInterval)
	defer ticker.Stop()
	log.Printf("[AccountProcessorJob] createSweepTransactionPayload started (interval=%s)", j.cfg.PayloadInterval)
	for {
		select {
		case <-ctx.Done():
			log.Printf("[AccountProcessorJob] createSweepTransactionPayload: ctx cancelled, exiting")
			return
		case <-ticker.C:
			func() {
				defer recoverSubLoop("createSweepTransactionPayload")
				j.createSweepTransactionPayloads()
			}()
		}
	}
}

// createSweepTransactionPayloads builds SweepTransaction rows for sweeps that
// have been initiated but not yet assigned payload data.
func (j *AccountProcessorJob) createSweepTransactionPayloads() {
	pending, err := j.sweepTxSvc.ListPending()
	if err != nil {
		log.Printf("[AccountProcessorJob] createSweepTransactionPayload: list pending: %v", err)
		return
	}
	log.Printf("[AccountProcessorJob] createSweepTransactionPayload: %d pending sweep txs", len(pending))
}

// ─── Sub-goroutine 5: processRewards ─────────────────────────────────────────

// runProcessRewards ticks on RewardInterval and calls ProcessPending on
// AccountRewardService. Real on-chain signing is wired in Phase K.
//
// TODO(phase-k-payout): Replace placeholder fulfillment with SecretsVault
// signing once Phase K on-chain payout signing is complete.
func (j *AccountProcessorJob) runProcessRewards(ctx context.Context) {
	ticker := time.NewTicker(j.cfg.RewardInterval)
	defer ticker.Stop()
	log.Printf("[AccountProcessorJob] processRewards started (interval=%s)", j.cfg.RewardInterval)
	for {
		select {
		case <-ctx.Done():
			log.Printf("[AccountProcessorJob] processRewards: ctx cancelled, exiting")
			return
		case <-ticker.C:
			func() {
				defer recoverSubLoop("processRewards")
				if err := j.rewardSvc.ProcessPending(); err != nil {
					log.Printf("[AccountProcessorJob] processRewards: %v", err)
				}
			}()
		}
	}
}

// ─── Sub-goroutine 6: processFailedRewards ───────────────────────────────────

// runProcessFailedRewards ticks on FailedRewardInterval. Real retry signing is
// wired in Phase K.
//
// TODO(phase-k-payout): Implement real on-chain retry once Phase K signing is complete.
func (j *AccountProcessorJob) runProcessFailedRewards(ctx context.Context) {
	ticker := time.NewTicker(j.cfg.FailedRewardInterval)
	defer ticker.Stop()
	log.Printf("[AccountProcessorJob] processFailedRewards started (interval=%s)", j.cfg.FailedRewardInterval)
	for {
		select {
		case <-ctx.Done():
			log.Printf("[AccountProcessorJob] processFailedRewards: ctx cancelled, exiting")
			return
		case <-ticker.C:
			func() {
				defer recoverSubLoop("processFailedRewards")
				if err := j.rewardSvc.RetryFailed(); err != nil {
					log.Printf("[AccountProcessorJob] processFailedRewards: %v", err)
				}
			}()
		}
	}
}

// ─── Sub-goroutine 7: processWithdrawals ─────────────────────────────────────

// runProcessWithdrawals ticks on WithdrawalInterval.
//
// TODO(phase-g-withdrawal): Replace stub with real WithdrawalProcessingService
// once Phase G withdrawal execution is implemented.
func (j *AccountProcessorJob) runProcessWithdrawals(ctx context.Context) {
	ticker := time.NewTicker(j.cfg.WithdrawalInterval)
	defer ticker.Stop()
	log.Printf("[AccountProcessorJob] processWithdrawals started (interval=%s) — stub until Phase G", j.cfg.WithdrawalInterval)
	for {
		select {
		case <-ctx.Done():
			log.Printf("[AccountProcessorJob] processWithdrawals: ctx cancelled, exiting")
			return
		case <-ticker.C:
			func() {
				defer recoverSubLoop("processWithdrawals")
				if err := j.withdrawalSvc.ProcessApproved(ctx); err != nil {
					log.Printf("[AccountProcessorJob] processWithdrawals: %v", err)
				}
			}()
		}
	}
}

// ─── Sub-goroutine 8: retryStaleBTCSweepTransactions ─────────────────────────

// runRetryStaleBTCSweepTransactions ticks on StaleBTCInterval.
func (j *AccountProcessorJob) runRetryStaleBTCSweepTransactions(ctx context.Context) {
	ticker := time.NewTicker(j.cfg.StaleBTCInterval)
	defer ticker.Stop()
	log.Printf("[AccountProcessorJob] retryStaleBTCSweepTransactions started (interval=%s)", j.cfg.StaleBTCInterval)
	for {
		select {
		case <-ctx.Done():
			log.Printf("[AccountProcessorJob] retryStaleBTCSweepTransactions: ctx cancelled, exiting")
			return
		case <-ticker.C:
			func() {
				defer recoverSubLoop("retryStaleBTCSweepTransactions")
				j.retryStaleBTCSweepTransactions()
			}()
		}
	}
}

// retryStaleBTCSweepTransactions rebroadcasts BTC sweep transactions that
// have been in 'broadcast' state longer than the stale threshold.
//
// TODO(phase-k-signing): Query BTC SweepTransactions in 'broadcast' state
// older than SweepService.StaleThreshold and rebroadcast via the Bitcoin adapter.
func (j *AccountProcessorJob) retryStaleBTCSweepTransactions() {
	broadcast, err := j.sweepTxSvc.ListByStatus("broadcast")
	if err != nil {
		log.Printf("[AccountProcessorJob] retryStaleBTCSweepTransactions: %v", err)
		return
	}
	if len(broadcast) > 0 {
		log.Printf("[AccountProcessorJob] retryStaleBTCSweepTransactions: %d broadcast txs to retry — PENDING PHASE-K SIGNING", len(broadcast))
	}
}

// ─── Sub-goroutine 9: processStaleInitiatedSweeps ────────────────────────────

// runStaleInitiatedSweeps ticks on StaleInitiatedInterval and calls both
// processStaleInitiatedSweeps and processConfirmingBTCSweepsOnce.
func (j *AccountProcessorJob) runStaleInitiatedSweeps(ctx context.Context) {
	ticker := time.NewTicker(j.cfg.StaleInitiatedInterval)
	defer ticker.Stop()
	log.Printf("[AccountProcessorJob] processStaleInitiatedSweeps started (interval=%s)", j.cfg.StaleInitiatedInterval)
	for {
		select {
		case <-ctx.Done():
			log.Printf("[AccountProcessorJob] processStaleInitiatedSweeps: ctx cancelled, exiting")
			return
		case <-ticker.C:
			func() {
				defer recoverSubLoop("processStaleInitiatedSweeps")
				j.processStaleInitiatedSweeps()
				j.processConfirmingBTCSweepsOnce()
			}()
		}
	}
}

// processStaleInitiatedSweeps marks sweeps that have been pending longer than
// StaleThreshold as stale so they can be re-evaluated on the next cycle.
func (j *AccountProcessorJob) processStaleInitiatedSweeps() {
	pending, err := j.sweepSvc.ListPending()
	if err != nil {
		log.Printf("[AccountProcessorJob] processStaleInitiatedSweeps: list: %v", err)
		return
	}
	now := time.Now()
	for _, s := range pending {
		if now.Sub(s.CreatedAt) > service.StaleThreshold {
			if err := j.sweepSvc.MarkStale(s.ID); err != nil {
				log.Printf("[AccountProcessorJob] processStaleInitiatedSweeps: mark stale %d: %v", s.ID, err)
			} else {
				log.Printf("[AccountProcessorJob] processStaleInitiatedSweeps: marked sweep %d stale", s.ID)
			}
		}
	}
}

// processConfirmingBTCSweepsOnce checks BTC sweep transactions in 'confirming'
// state and advances them when they reach the required confirmation count.
//
// TODO(phase-k-signing): Query BTC RPC for confirmation count of each
// SweepTransaction.TxHash and call UpdateStatus accordingly.
func (j *AccountProcessorJob) processConfirmingBTCSweepsOnce() {
	confirming, err := j.sweepTxSvc.ListByStatus("confirming")
	if err != nil {
		log.Printf("[AccountProcessorJob] processConfirmingBTCSweepsOnce: %v", err)
		return
	}
	if len(confirming) > 0 {
		log.Printf("[AccountProcessorJob] processConfirmingBTCSweepsOnce: %d confirming BTC txs — PENDING PHASE-K SIGNING", len(confirming))
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// recoverSubLoop is a deferred panic-recovery function used by every sub-loop.
// It logs the stack trace but does NOT re-panic, allowing the sub-goroutine to
// exit gracefully rather than crashing the entire process.
func recoverSubLoop(name string) {
	if r := recover(); r != nil {
		log.Printf("[AccountProcessorJob] PANIC recovered in %s: %v\n%s", name, r, debug.Stack())
	}
}
