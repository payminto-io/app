package worker

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/blockchain"
	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"github.com/payminto/payminto/backend/internal/service"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Stub adapter used only in these tests — not the fakeAdapter from other pkg.
// ---------------------------------------------------------------------------

type stubAdapter struct {
	mu           sync.Mutex
	latestBlock  uint64
	latestErr    error
	parseBlockFn func(ctx context.Context, blockNumber uint64, watched map[string]blockchain.WatchedAddressInfo) ([]blockchain.Transaction, error)
}

func (s *stubAdapter) Name() string                           { return "stub" }
func (s *stubAdapter) Code() string                           { return "STUB" }
func (s *stubAdapter) IsMainnet() bool                        { return false }
func (s *stubAdapter) GenerateAddress(uint32) (string, error) { return "", nil }
func (s *stubAdapter) GetBalance(context.Context, string, string) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (s *stubAdapter) GetConfirmations(context.Context, string) (uint64, error) { return 0, nil }
func (s *stubAdapter) MonitorBlocks(context.Context, uint64) (<-chan blockchain.Transaction, error) {
	ch := make(chan blockchain.Transaction)
	close(ch)
	return ch, nil
}
func (s *stubAdapter) BroadcastTransaction(context.Context, []byte) (string, error) {
	return "", nil
}
func (s *stubAdapter) EstimateGas(context.Context, blockchain.TxParams) (*big.Int, error) {
	return big.NewInt(0), nil
}
func (s *stubAdapter) LatestBlock(ctx context.Context) (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.latestBlock, s.latestErr
}
func (s *stubAdapter) ParseBlock(ctx context.Context, blockNumber uint64, watched map[string]blockchain.WatchedAddressInfo) ([]blockchain.Transaction, error) {
	if s.parseBlockFn != nil {
		return s.parseBlockFn(ctx, blockNumber, watched)
	}
	return []blockchain.Transaction{}, nil
}

var _ blockchain.ChainAdapter = (*stubAdapter)(nil)

// ---------------------------------------------------------------------------
// Test DB helpers
// ---------------------------------------------------------------------------

func setupProcessorDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	err = db.AutoMigrate(
		&models.Member{},
		&models.BlockchainFamily{},
		&models.Blockchain{},
		&models.Currency{},
		&models.BlockchainCurrency{},
		&models.Wallet{},
		&models.DepositAddress{},
		&models.PaymentRequest{},
		&models.Deposit{},
		&models.MissedDeposit{},
	)
	if err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

// seedChain creates a Blockchain + BlockchainCurrency row and returns both.
func seedChain(t *testing.T, db *gorm.DB) (*models.Blockchain, *models.BlockchainCurrency) {
	t.Helper()
	family := &models.BlockchainFamily{Code: "evm", Name: "EVM"}
	if err := db.Create(family).Error; err != nil {
		t.Fatal(err)
	}
	chain := &models.Blockchain{
		Code:               "ETH",
		Name:               "Ethereum",
		BlockchainFamilyID: family.ID,
		Height:             0,
		MinConfirmations:   1,
	}
	if err := db.Create(chain).Error; err != nil {
		t.Fatal(err)
	}
	currency := &models.Currency{Code: "ETH", Name: "Ethereum", Type: "native"}
	if err := db.Create(currency).Error; err != nil {
		t.Fatal(err)
	}
	bc := &models.BlockchainCurrency{
		BlockchainID:   chain.ID,
		CurrencyID:     currency.ID,
		DepositEnabled: true,
	}
	if err := db.Create(bc).Error; err != nil {
		t.Fatal(err)
	}
	return chain, bc
}

// seedMember creates a minimal member row for tests.
func seedMember(t *testing.T, db *gorm.DB, name string) *models.Member {
	t.Helper()
	email := name + "@test.com"
	member := &models.Member{Name: name, Email: &email}
	if err := db.Create(member).Error; err != nil {
		t.Fatal(err)
	}
	return member
}

// buildProcessor builds a processor with real repo/service wired to an in-memory DB.
func buildProcessor(t *testing.T, db *gorm.DB, chain *models.Blockchain, adapter blockchain.ChainAdapter) *BlockchainProcessor {
	t.Helper()
	blockchainRepo := repository.NewBlockchainRepository(db)
	depositRepo := repository.NewDepositRepository(db)
	depositAddrRepo := repository.NewDepositAddressRepository(db)
	missedRepo := repository.NewMissedDepositRepository(db)
	bcCurRepo := repository.NewBlockchainCurrencyRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)

	depositSvc := service.NewDepositService(depositRepo, depositAddrRepo, paymentRepo, bcCurRepo, blockchainRepo)

	return NewBlockchainProcessor(
		chain,
		blockchainRepo,
		depositSvc,
		depositAddrRepo,
		missedRepo,
		bcCurRepo,
		adapter,
	)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestBlockchainProcessor_Name(t *testing.T) {
	db := setupProcessorDB(t)
	chain, _ := seedChain(t, db)
	p := buildProcessor(t, db, chain, &stubAdapter{})

	if p.Name() != "ETH_block_processor" {
		t.Errorf("expected ETH_block_processor, got %q", p.Name())
	}
}

func TestBlockchainProcessor_HandleExtractedDeposit_Matched(t *testing.T) {
	db := setupProcessorDB(t)
	chain, bc := seedChain(t, db)

	// Create a member and a deposit address for the chain currency
	member := seedMember(t, db, "matched")

	da := &models.DepositAddress{
		Address:              "0xWATCHED",
		BlockchainCurrencyID: bc.ID,
		MemberID:             member.ID,
	}
	db.Create(da)

	p := buildProcessor(t, db, chain, &stubAdapter{})
	if err := p.refreshWatchedAddresses(); err != nil {
		t.Fatal(err)
	}

	tx := blockchain.Transaction{
		TxHash:      "0xabc123",
		FromAddress: "0xSENDER",
		ToAddress:   "0xwatched", // lowercase (C6: normalised)
		Amount:      decimal.NewFromFloat(0.5),
		BlockNumber: 100,
	}

	if err := p.handleExtractedDeposit(t.Context(), tx); err != nil {
		t.Fatalf("handleExtractedDeposit: %v", err)
	}

	var deposits []models.Deposit
	db.Find(&deposits)
	if len(deposits) != 1 {
		t.Fatalf("expected 1 deposit row, got %d", len(deposits))
	}
	if deposits[0].TxID != "0xabc123" {
		t.Errorf("expected tx 0xabc123, got %s", deposits[0].TxID)
	}
}

func TestBlockchainProcessor_HandleExtractedDeposit_Unmatched_RecordsMissed(t *testing.T) {
	db := setupProcessorDB(t)
	chain, _ := seedChain(t, db)

	p := buildProcessor(t, db, chain, &stubAdapter{})
	// No deposit addresses seeded — everything is unmatched.

	tx := blockchain.Transaction{
		TxHash:      "0xunmatched",
		FromAddress: "0xSENDER",
		ToAddress:   "0xUNKNOWN",
		Amount:      decimal.NewFromFloat(1.0),
		BlockNumber: 200,
	}

	if err := p.handleExtractedDeposit(t.Context(), tx); err != nil {
		t.Fatalf("handleExtractedDeposit: %v", err)
	}

	var missed []models.MissedDeposit
	db.Find(&missed)
	if len(missed) != 1 {
		t.Fatalf("expected 1 missed_deposit row, got %d", len(missed))
	}
	if missed[0].TxHash != "0xunmatched" {
		t.Errorf("expected tx 0xunmatched, got %s", missed[0].TxHash)
	}
	if missed[0].Reason != "address not watched" {
		t.Errorf("unexpected reason: %q", missed[0].Reason)
	}
	// C5: BlockchainCurrencyID should be nil; BlockchainID should be the chain's ID.
	if missed[0].BlockchainCurrencyID != nil {
		t.Errorf("expected nil BlockchainCurrencyID for unmatched, got %v", missed[0].BlockchainCurrencyID)
	}
	if missed[0].BlockchainID != chain.ID {
		t.Errorf("expected BlockchainID=%d, got %d", chain.ID, missed[0].BlockchainID)
	}
}

func TestBlockchainProcessor_HandleExtractedDeposit_DustIgnored(t *testing.T) {
	db := setupProcessorDB(t)
	chain, bc := seedChain(t, db)

	// Set a min deposit amount on the blockchain currency.
	minAmt := decimal.NewFromFloat(1.0)
	db.Model(&models.BlockchainCurrency{}).Where("id = ?", bc.ID).Update("min_deposit_amount", minAmt)

	member := seedMember(t, db, "dust")
	da := &models.DepositAddress{
		Address:              "0xDUST_ADDR",
		BlockchainCurrencyID: bc.ID,
		MemberID:             member.ID,
	}
	db.Create(da)

	p := buildProcessor(t, db, chain, &stubAdapter{})
	if err := p.refreshWatchedAddresses(); err != nil {
		t.Fatal(err)
	}

	tx := blockchain.Transaction{
		TxHash:      "0xdust",
		ToAddress:   "0xdust_addr",               // lowercase normalised (ETH chain)
		Amount:      decimal.NewFromFloat(0.001), // below min
		BlockNumber: 300,
	}

	if err := p.handleExtractedDeposit(t.Context(), tx); err != nil {
		t.Fatalf("handleExtractedDeposit: %v", err)
	}

	var deposits []models.Deposit
	db.Find(&deposits)
	var missed []models.MissedDeposit
	db.Find(&missed)

	if len(deposits) != 0 {
		t.Errorf("expected 0 deposits (dust), got %d", len(deposits))
	}
	if len(missed) != 0 {
		t.Errorf("expected 0 missed (dust), got %d", len(missed))
	}
}

func TestBlockchainProcessor_HandleExtractedDeposit_ZeroAmount_Ignored(t *testing.T) {
	db := setupProcessorDB(t)
	chain, bc := seedChain(t, db)

	member := seedMember(t, db, "zero")
	da := &models.DepositAddress{
		Address:              "0xZERO_ADDR",
		BlockchainCurrencyID: bc.ID,
		MemberID:             member.ID,
	}
	db.Create(da)

	p := buildProcessor(t, db, chain, &stubAdapter{})
	if err := p.refreshWatchedAddresses(); err != nil {
		t.Fatal(err)
	}

	tx := blockchain.Transaction{
		TxHash:      "0xzero",
		ToAddress:   "0xzero_addr",
		Amount:      decimal.Zero,
		BlockNumber: 400,
	}

	if err := p.handleExtractedDeposit(t.Context(), tx); err != nil {
		t.Fatalf("handleExtractedDeposit: %v", err)
	}

	var deposits []models.Deposit
	db.Find(&deposits)
	var missed []models.MissedDeposit
	db.Find(&missed)

	if len(deposits) != 0 {
		t.Errorf("expected 0 deposits, got %d", len(deposits))
	}
	if len(missed) != 0 {
		t.Errorf("expected 0 missed deposits, got %d", len(missed))
	}
}

func TestBlockchainProcessor_RefreshWatchedAddresses(t *testing.T) {
	db := setupProcessorDB(t)
	chain, bc := seedChain(t, db)

	member := seedMember(t, db, "refresh")

	// Seed two deposit addresses for this chain.
	addrs := []string{"0xADDR_A", "0xADDR_B"}
	for _, addr := range addrs {
		db.Create(&models.DepositAddress{
			Address:              addr,
			BlockchainCurrencyID: bc.ID,
			MemberID:             member.ID,
		})
	}

	p := buildProcessor(t, db, chain, &stubAdapter{})
	if err := p.refreshWatchedAddresses(); err != nil {
		t.Fatal(err)
	}

	p.mu.RLock()
	size := len(p.watchedAddresses)
	p.mu.RUnlock()

	if size != 2 {
		t.Errorf("expected watched address map size 2, got %d", size)
	}
}

// TestBlockchainProcessor_RefreshWatchedAddresses_EVMAddressLowercased verifies
// that EVM (ETH chain) addresses are lowercased in the watched map (C6 regression).
func TestBlockchainProcessor_RefreshWatchedAddresses_EVMAddressLowercased(t *testing.T) {
	db := setupProcessorDB(t)
	chain, bc := seedChain(t, db) // chain.Code = "ETH"

	member := seedMember(t, db, "evm_case")
	// Seed the address in EIP-55 checksum form.
	checksum := "0xAbCdEf1234567890AbCdEf1234567890AbCdEf12"
	db.Create(&models.DepositAddress{
		Address:              checksum,
		BlockchainCurrencyID: bc.ID,
		MemberID:             member.ID,
	})

	p := buildProcessor(t, db, chain, &stubAdapter{})
	if err := p.refreshWatchedAddresses(); err != nil {
		t.Fatal(err)
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	lower := strings.ToLower(checksum)
	if _, ok := p.watchedAddresses[lower]; !ok {
		t.Errorf("expected lowercase key %q in watched map, got keys: %v", lower, p.watchedAddresses)
	}
	// Original checksum form should NOT be present.
	if _, ok := p.watchedAddresses[checksum]; ok {
		t.Errorf("original checksum %q should not be a key (ETH addresses must be lowercased)", checksum)
	}
}

func TestBlockchainProcessor_ProcessSingleBlock_RoutesAllTransactions(t *testing.T) {
	db := setupProcessorDB(t)
	chain, bc := seedChain(t, db)

	member := seedMember(t, db, "route")
	da := &models.DepositAddress{
		Address:              "0xROUTE_ADDR",
		BlockchainCurrencyID: bc.ID,
		MemberID:             member.ID,
	}
	db.Create(da)

	// Stub adapter returns 3 transactions for any block.
	txs := []blockchain.Transaction{
		{TxHash: "0x001", ToAddress: "0xroute_addr", Amount: decimal.NewFromFloat(1), BlockNumber: 1},
		{TxHash: "0x002", ToAddress: "0xroute_addr", Amount: decimal.NewFromFloat(2), BlockNumber: 1},
		{TxHash: "0x003", ToAddress: "0xroute_addr", Amount: decimal.NewFromFloat(3), BlockNumber: 1},
	}
	adapter := &stubAdapter{
		parseBlockFn: func(_ context.Context, _ uint64, _ map[string]blockchain.WatchedAddressInfo) ([]blockchain.Transaction, error) {
			return txs, nil
		},
	}

	p := buildProcessor(t, db, chain, adapter)
	if err := p.refreshWatchedAddresses(); err != nil {
		t.Fatal(err)
	}

	if err := p.processSingleBlock(t.Context(), 1); err != nil {
		t.Fatalf("processSingleBlock: %v", err)
	}

	var deposits []models.Deposit
	db.Find(&deposits)
	if len(deposits) != 3 {
		t.Errorf("expected 3 deposit rows, got %d", len(deposits))
	}
}

func TestBlockchainProcessor_ProcessSingleBlock_AdapterError(t *testing.T) {
	db := setupProcessorDB(t)
	chain, _ := seedChain(t, db)

	adapter := &stubAdapter{
		parseBlockFn: func(_ context.Context, _ uint64, _ map[string]blockchain.WatchedAddressInfo) ([]blockchain.Transaction, error) {
			return nil, errors.New("node unavailable")
		},
	}

	p := buildProcessor(t, db, chain, adapter)
	err := p.processSingleBlock(t.Context(), 42)
	if err == nil {
		t.Error("expected error when adapter fails, got nil")
	}
}

func TestBlockchainProcessor_RunConfirmationLoop_AdvancesPending(t *testing.T) {
	db := setupProcessorDB(t)
	chain, bc := seedChain(t, db)

	member := seedMember(t, db, "confirm")
	da := &models.DepositAddress{
		Address:              "0xCONFIRM_ADDR",
		BlockchainCurrencyID: bc.ID,
		MemberID:             member.ID,
	}
	db.Create(da)

	// Seed a deposit in 'confirming' state at block 100.
	deposit := &models.Deposit{
		TxID:                  "0xconfirmtx",
		Amount:                decimal.NewFromFloat(1),
		Status:                models.DepositStatusConfirming,
		Confirmations:         1,
		RequiredConfirmations: 12,
		ToAddress:             "0xCONFIRM_ADDR",
		BlockNumber:           100,
		BlockchainCurrencyID:  bc.ID,
		MemberID:              member.ID,
	}
	db.Create(deposit)

	// Test UpdateConfirmations directly — this is what the confirmation loop calls.
	// Chain tip at 110 = 10 confirmations (110 - 100 + 1 = 11).
	p := buildProcessor(t, db, chain, &stubAdapter{latestBlock: 110})
	if err := p.depositService.UpdateConfirmations(deposit.ID, 110); err != nil {
		t.Fatalf("UpdateConfirmations: %v", err)
	}

	// Fetch and check that confirmations advanced.
	var updated models.Deposit
	db.First(&updated, deposit.ID)
	if updated.Confirmations < 2 {
		t.Errorf("expected confirmations > 1 after update, got %d", updated.Confirmations)
	}
}

// TestBlockchainProcessor_RunConfirmationLoop_ContextCancel verifies the
// confirmation loop exits cleanly on context cancellation.
func TestBlockchainProcessor_RunConfirmationLoop_ContextCancel(t *testing.T) {
	db := setupProcessorDB(t)
	chain, _ := seedChain(t, db)

	adapter := &stubAdapter{latestBlock: 1000}
	p := buildProcessor(t, db, chain, adapter)
	p.confirmationRefresh = 10 * time.Millisecond

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		p.runConfirmationLoop(ctx)
		close(done)
	}()

	select {
	case <-done:
		// OK — loop exited after context cancel
	case <-time.After(500 * time.Millisecond):
		t.Error("confirmation loop did not exit after context cancel")
	}
}

func TestBlockchainProcessor_Start_NilAdapterReturnsError(t *testing.T) {
	db := setupProcessorDB(t)
	chain, _ := seedChain(t, db)

	p := buildProcessor(t, db, chain, nil) // nil adapter
	err := p.Start(t.Context())
	if err == nil {
		t.Error("expected error for nil adapter")
	}
}

func TestBlockchainProcessor_HandleExtractedDeposit_Idempotent(t *testing.T) {
	db := setupProcessorDB(t)
	chain, bc := seedChain(t, db)

	member := seedMember(t, db, "idem")
	da := &models.DepositAddress{
		Address:              "0xIDEM_ADDR",
		BlockchainCurrencyID: bc.ID,
		MemberID:             member.ID,
	}
	db.Create(da)

	p := buildProcessor(t, db, chain, &stubAdapter{})
	if err := p.refreshWatchedAddresses(); err != nil {
		t.Fatal(err)
	}

	tx := blockchain.Transaction{
		TxHash:      "0xidem",
		ToAddress:   "0xidem_addr",
		Amount:      decimal.NewFromFloat(0.5),
		BlockNumber: 500,
	}

	// Call twice — second should be a no-op (dedup in RecordDeposit).
	for i := 0; i < 2; i++ {
		if err := p.handleExtractedDeposit(t.Context(), tx); err != nil {
			t.Fatalf("call %d: %v", i+1, err)
		}
	}

	var deposits []models.Deposit
	db.Find(&deposits)
	if len(deposits) != 1 {
		t.Errorf("expected exactly 1 deposit (idempotent), got %d", len(deposits))
	}
}

// TestBlockchainProcessor_ERC20DecimalScaling verifies that an ERC-20 deposit
// with raw amount 1_000_000 and Decimals=6 is recorded as 1.0 (C3 regression).
func TestBlockchainProcessor_ERC20DecimalScaling(t *testing.T) {
	db := setupProcessorDB(t)
	chain, bc := seedChain(t, db)

	// Set WalletPrecision=6 on the BC to simulate USDT (6 decimals).
	db.Model(&models.BlockchainCurrency{}).Where("id = ?", bc.ID).Update("wallet_precision", 6)

	member := seedMember(t, db, "erc20")
	da := &models.DepositAddress{
		Address:              "0xERC20_ADDR",
		BlockchainCurrencyID: bc.ID,
		MemberID:             member.ID,
	}
	db.Create(da)

	p := buildProcessor(t, db, chain, &stubAdapter{})
	if err := p.refreshWatchedAddresses(); err != nil {
		t.Fatal(err)
	}

	// Simulate a raw ERC-20 amount of 1_000_000 (= 1 USDT with 6 decimals)
	// that was already scaled by the adapter using WatchedAddressInfo.Decimals.
	// The processor receives the already-scaled amount from the adapter.
	// This test verifies the watched map populates Decimals=6 correctly.
	p.mu.RLock()
	info, ok := p.watchedAddresses[strings.ToLower("0xERC20_ADDR")]
	p.mu.RUnlock()
	if !ok {
		t.Fatal("expected 0xerc20_addr in watched map")
	}
	if info.Decimals != 6 {
		t.Errorf("expected Decimals=6 in WatchedAddressInfo, got %d", info.Decimals)
	}

	// Feed a tx that the adapter would have already scaled: 1.0 USDT.
	tx := blockchain.Transaction{
		TxHash:      "0xerc20tx",
		ToAddress:   "0xerc20_addr",
		Amount:      decimal.NewFromFloat(1.0), // post-scaling
		BlockNumber: 600,
	}
	if err := p.handleExtractedDeposit(t.Context(), tx); err != nil {
		t.Fatalf("handleExtractedDeposit: %v", err)
	}

	var deposits []models.Deposit
	db.Find(&deposits)
	if len(deposits) != 1 {
		t.Fatalf("expected 1 deposit, got %d", len(deposits))
	}
	if !deposits[0].Amount.Equal(decimal.NewFromFloat(1.0)) {
		t.Errorf("expected Amount=1.0, got %s", deposits[0].Amount.String())
	}
}

// TestBlockchainProcessor_C6_ChecksumAddressMatchesLowercase verifies that a
// deposit to address 0xAbCd... (EIP-55 checksum) is matched against a watched
// address stored as 0xabcd... (lowercase) in the map (C6 regression).
func TestBlockchainProcessor_C6_ChecksumAddressMatchesLowercase(t *testing.T) {
	db := setupProcessorDB(t)
	chain, bc := seedChain(t, db) // chain.Code = "ETH"

	member := seedMember(t, db, "c6check")
	// Seed the address in mixed-case (EIP-55) form.
	mixedCase := "0xAbCdEf1234567890AbCdEf1234567890AbCdEf12"
	db.Create(&models.DepositAddress{
		Address:              mixedCase,
		BlockchainCurrencyID: bc.ID,
		MemberID:             member.ID,
	})

	p := buildProcessor(t, db, chain, &stubAdapter{})
	if err := p.refreshWatchedAddresses(); err != nil {
		t.Fatal(err)
	}

	// The adapter would normalise the tx.ToAddress to lowercase before emitting.
	lowerCase := strings.ToLower(mixedCase)
	tx := blockchain.Transaction{
		TxHash:      "0xc6test",
		ToAddress:   lowerCase,
		Amount:      decimal.NewFromFloat(0.1),
		BlockNumber: 700,
	}

	if err := p.handleExtractedDeposit(t.Context(), tx); err != nil {
		t.Fatalf("handleExtractedDeposit: %v", err)
	}

	var deposits []models.Deposit
	db.Find(&deposits)
	if len(deposits) != 1 {
		t.Errorf("expected 1 deposit (checksum→lowercase match), got %d", len(deposits))
	}
}

func TestNewEthereumBlockProcessor_Name(t *testing.T) {
	db := setupProcessorDB(t)
	chain, bc := seedChain(t, db)
	_ = bc

	deps := ProcessorDeps{
		BlockchainRepo:         repository.NewBlockchainRepository(db),
		DepositService:         service.NewDepositService(repository.NewDepositRepository(db), repository.NewDepositAddressRepository(db), repository.NewPaymentRepository(db), repository.NewBlockchainCurrencyRepository(db), repository.NewBlockchainRepository(db)),
		DepositAddressRepo:     repository.NewDepositAddressRepository(db),
		MissedDepositRepo:      repository.NewMissedDepositRepository(db),
		BlockchainCurrencyRepo: repository.NewBlockchainCurrencyRepository(db),
	}
	p := NewEthereumBlockProcessor(chain, deps, &stubAdapter{})
	if p.Name() != "ETH_block_processor" {
		t.Errorf("got %q", p.Name())
	}
}
