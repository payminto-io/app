package service

import (
	"context"
	"log"
	"math/big"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
)

// weiToEthScale is the decimal scale passed to decimal.NewFromBigInt to convert
// a wei integer to whole coins: value × 10^scale, so -18 divides by 10^18.
// SweepTransaction.Amount is stored in human coin units (ETH), not wei.
const weiToEthScale = -18

// evmChains is the set of chain codes EVMSweepService handles.
var evmChains = map[string]bool{"ETH": true, "BASE": true, "POLYGON": true}

// evmMinSweepWei is the default dust floor for native sweeps (0.001 ETH). A
// confirmed deposit whose net sweepable amount (balance − gas) is below this is
// left for a later round rather than swept at a loss.
var evmMinSweepWei = big.NewInt(1_000_000_000_000_000)

// nativeSweeper is the subset of EVMBroadcaster that EVMSweepService needs;
// keeping it an interface makes the service unit-testable with a fake.
type nativeSweeper interface {
	SweepNative(ctx context.Context, chainCode, fromAddress, to string, minSweepWei *big.Int) (txHash string, amount, gasFee *big.Int, err error)
}

// EVMSweepService consolidates confirmed native-coin (ETH/BASE/POLYGON) deposits
// from their per-payment deposit addresses into the cold wallet. It is the
// orchestration layer above EVMBroadcaster: it decides WHAT to sweep (confirmed,
// native, EVM, not-yet-swept deposits) and records the result, while the
// broadcaster handles the signing/balance/broadcast mechanics.
//
// Idempotent: each deposit is marked `swept` once its sweep is broadcast, so
// re-runs never double-sweep. ERC-20 (two-phase, gas-funded) sweeps and a
// confirmation-tracking loop are the next increments (see docs/EVM_SETTLEMENT.md).
type EVMSweepService struct {
	deposits    repository.DepositRepository
	currencies  repository.BlockchainCurrencyRepository
	sweepSvc    *SweepService
	sweepTxSvc  *SweepTransactionService
	broadcaster nativeSweeper
	coldWallet  string
	minSweepWei *big.Int
}

// NewEVMSweepService constructs an EVMSweepService. coldWallet is the EVM
// cold-storage destination; minSweepWei is the dust floor (deposits whose net
// sweepable amount is below this are skipped this round).
func NewEVMSweepService(
	deposits repository.DepositRepository,
	currencies repository.BlockchainCurrencyRepository,
	sweepSvc *SweepService,
	sweepTxSvc *SweepTransactionService,
	broadcaster nativeSweeper,
	coldWallet string,
	minSweepWei *big.Int,
) *EVMSweepService {
	return &EVMSweepService{
		deposits:    deposits,
		currencies:  currencies,
		sweepSvc:    sweepSvc,
		sweepTxSvc:  sweepTxSvc,
		broadcaster: broadcaster,
		coldWallet:  coldWallet,
		minSweepWei: minSweepWei,
	}
}

// SweepConfirmedNative sweeps every confirmed native-EVM deposit to the cold
// wallet. Returns the number of deposits successfully swept this round. Errors
// on individual deposits are logged and skipped (retried next tick); the method
// only returns a fatal error if it cannot list deposits at all.
func (s *EVMSweepService) SweepConfirmedNative(ctx context.Context) (int, error) {
	if s.coldWallet == "" {
		return 0, nil // not configured — nothing to do
	}
	deposits, err := s.deposits.ListByStatus(models.DepositStatusConfirmed)
	if err != nil {
		return 0, err
	}

	swept := 0
	for i := range deposits {
		if ctx.Err() != nil {
			return swept, ctx.Err()
		}
		if s.sweepOne(ctx, &deposits[i]) {
			swept++
		}
	}
	return swept, nil
}

// sweepOne handles a single deposit; returns true if a sweep tx was broadcast.
func (s *EVMSweepService) sweepOne(ctx context.Context, d *models.Deposit) bool {
	bc, err := s.currencies.GetByID(d.BlockchainCurrencyID)
	if err != nil {
		log.Printf("[EVMSweepService] deposit %d: load currency: %v", d.ID, err)
		return false
	}
	// Only native EVM coins here. ERC-20 (has a contract Address) is two-phase
	// and handled separately; non-EVM chains are out of scope for this service.
	if !evmChains[bc.BlockchainCode] || bc.Address != "" {
		return false
	}

	// CLAIM-FIRST: atomically transition confirmed → swept BEFORE broadcasting.
	// This guarantees at-most-once broadcast — if a later DB write fails, the
	// deposit is already swept so no second broadcast can occur. On broadcast
	// failure we revert the claim so the next round retries.
	claimed, err := s.deposits.ClaimForSweep(d.ID)
	if err != nil {
		log.Printf("[EVMSweepService] deposit %d claim: %v", d.ID, err)
		return false
	}
	if claimed == 0 {
		return false // already claimed by another worker/round
	}

	txHash, amountWei, gasFee, err := s.broadcaster.SweepNative(ctx, bc.BlockchainCode, d.ToAddress, s.coldWallet, s.minSweepWei)
	if err != nil {
		// Broadcast failed — release the claim so we retry next round.
		if rerr := s.deposits.UpdateStatus(d.ID, models.DepositStatusConfirmed); rerr != nil {
			log.Printf("[EVMSweepService] deposit %d revert claim after broadcast failure: %v", d.ID, rerr)
		}
		log.Printf("[EVMSweepService] deposit %d sweep %s→cold: %v", d.ID, d.ToAddress, err)
		return false
	}
	if txHash == "" {
		// Below dust/min threshold — nothing on-chain to move. Leave it swept
		// (terminal) so we don't reconsider it every tick.
		return false
	}

	// Record the broadcast tx. The deposit is already swept (claimed), so even
	// if this write fails the tx is never re-broadcast; the confirmation loop
	// reconciles the on-chain tx against the address.
	sweep, err := s.sweepSvc.CreateSweep(bc.BlockchainID)
	if err != nil {
		log.Printf("[EVMSweepService] deposit %d create sweep batch (tx %s already on-chain): %v", d.ID, txHash, err)
		return true
	}
	amount := decimal.NewFromBigInt(amountWei, weiToEthScale)
	fee := decimal.NewFromBigInt(gasFee, weiToEthScale)
	if _, err := s.sweepTxSvc.RecordBroadcastSweep(sweep.ID, d.BlockchainCurrencyID, d.ToAddress, s.coldWallet, txHash, amount, fee); err != nil {
		log.Printf("[EVMSweepService] deposit %d record sweep tx (tx %s already on-chain): %v", d.ID, txHash, err)
	}
	log.Printf("[EVMSweepService] swept deposit %d: %s %s→cold tx=%s amount=%s",
		d.ID, bc.BlockchainCode, d.ToAddress, txHash, amount)
	return true
}
