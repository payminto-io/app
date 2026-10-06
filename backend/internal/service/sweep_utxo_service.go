package service

import (
	"errors"
	"fmt"
	"log"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// MinBTCSweepThreshold is the minimum aggregate UTXO value (in BTC) required
// to trigger a sweep batch. Below this threshold, gas costs would outweigh the
// swept amount.
var MinBTCSweepThreshold = decimal.NewFromFloat(0.001)

// PlaceholderBTCFee is a conservative 2000-satoshi fee expressed in BTC
// (2000 sat = 0.00002 BTC), used until Phase K wires a real fee estimator
// via the Bitcoin RPC adapter's estimatesmartfee call.
//
// TODO(phase-k-btc-fee): replace with adapter.EstimateGas() output.
var PlaceholderBTCFee = decimal.NewFromFloat(0.00002)

// SweepUTXOService aggregates unspent UTXOs for Bitcoin sweep batches. It
// selects eligible UTXOs, groups them into a sweep payload, and delegates
// transaction creation to SweepTransactionService.
//
// db is held directly so SubmitBTCSweepForAddress can wrap the MarkSpent loop
// and SweepTransaction create in a single database transaction.
type SweepUTXOService struct {
	db             *gorm.DB
	utxoRepo       repository.UTXORepository
	sweepTxService *SweepTransactionService
	ledgerService  *LedgerService
}

// NewSweepUTXOService constructs a SweepUTXOService with the required dependencies.
func NewSweepUTXOService(
	db *gorm.DB,
	utxoRepo repository.UTXORepository,
	sweepTxService *SweepTransactionService,
	ledgerService *LedgerService,
) *SweepUTXOService {
	return &SweepUTXOService{
		db:             db,
		utxoRepo:       utxoRepo,
		sweepTxService: sweepTxService,
		ledgerService:  ledgerService,
	}
}

// AggregatePendingUTXOs collects all unspent UTXOs, groups them by address,
// and returns groups where the total value exceeds MinBTCSweepThreshold. Each
// entry in the returned slice represents one address ready for sweeping.
func (s *SweepUTXOService) AggregatePendingUTXOs() ([]AddressUTXOGroup, error) {
	all, err := s.utxoRepo.ListUnspent()
	if err != nil {
		return nil, fmt.Errorf("aggregate utxos: list unspent: %w", err)
	}

	byAddress := make(map[string]*AddressUTXOGroup)
	for _, u := range all {
		g, ok := byAddress[u.Address]
		if !ok {
			g = &AddressUTXOGroup{Address: u.Address}
			byAddress[u.Address] = g
		}
		g.UTXOs = append(g.UTXOs, u)
		g.Total = g.Total.Add(u.Amount)
	}

	var eligible []AddressUTXOGroup
	for _, g := range byAddress {
		if g.Total.GreaterThanOrEqual(MinBTCSweepThreshold) {
			eligible = append(eligible, *g)
		}
	}
	return eligible, nil
}

// SubmitBTCSweepForAddress submits a Bitcoin sweep for a single address group.
// It marks all selected UTXOs as spent and creates a pending SweepTransaction
// row inside a single database transaction, so either both succeed or neither
// does. estimatedFee is in BTC (e.g. PlaceholderBTCFee = 0.00002 BTC).
//
// If a UTXO was already claimed by a concurrent goroutine (ErrUTXOAlreadySpent),
// it is skipped rather than failing the whole batch — this is the expected
// "someone else got it" signal.
//
// TODO(phase-k-signing): Replace the log stub with a real Bitcoin raw
// transaction build + broadcast call after SecretsVault integration in Phase K.
func (s *SweepUTXOService) SubmitBTCSweepForAddress(
	sweepID uint,
	blockchainCurrencyID uint,
	group AddressUTXOGroup,
	toAddress string,
	estimatedFee decimal.Decimal,
) (*models.SweepTransaction, error) {
	netAmount := group.Total.Sub(estimatedFee)
	if netAmount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("btc sweep: net amount %s <= 0 after fee %s for address %s",
			netAmount, estimatedFee, group.Address)
	}

	var st *models.SweepTransaction

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		// Create the pending sweep transaction row inside the db transaction.
		var err error
		st, err = s.sweepTxService.CreateSweepTransactionPayload(
			sweepID,
			blockchainCurrencyID,
			group.Address,
			toAddress,
			netAmount,
		)
		if err != nil {
			return fmt.Errorf("btc sweep: create payload: %w", err)
		}

		// Atomically mark each UTXO as spent so concurrent goroutines cannot
		// sweep the same outputs. ErrUTXOAlreadySpent is benign — skip it.
		// Any other error aborts the transaction.
		spentTxRef := fmt.Sprintf("pending_%d", st.ID)
		for _, u := range group.UTXOs {
			if markErr := s.utxoRepo.MarkSpent(u.ID, spentTxRef); markErr != nil {
				if errors.Is(markErr, repository.ErrUTXOAlreadySpent) {
					log.Printf("[SweepUTXOService] utxo %d already spent by concurrent caller — skipping", u.ID)
					continue
				}
				return fmt.Errorf("btc sweep: mark utxo %d spent: %w", u.ID, markErr)
			}
		}
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	// TODO(phase-k-signing): Build a Bitcoin raw transaction from group.UTXOs,
	// sign it with the private key from SecretsVaultService, broadcast via the
	// Bitcoin adapter's SendRawTransaction RPC method, then call
	// sweepTxService.UpdateTxHash(st.ID, txHash) and transition the status to
	// SweepTxStatusBroadcast.
	log.Printf("[SweepUTXOService] BTC sweep: sweep_tx_id=%d from=%s net=%s fee=%s utxos=%d — PENDING PHASE-K SIGNING",
		st.ID, group.Address, netAmount, estimatedFee, len(group.UTXOs))

	return st, nil
}

// AddressUTXOGroup groups all unspent UTXOs for a single Bitcoin address,
// together with their aggregate value.
type AddressUTXOGroup struct {
	Address string
	UTXOs   []models.UTXO
	Total   decimal.Decimal
}
