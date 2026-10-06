package worker

import (
	"context"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newSCWTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.BlockchainFamily{},
		&models.Blockchain{},
		&models.BlockchainCurrency{},
		&models.Currency{},
		&models.AddressDeployment{},
	); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	return db
}

// seedBlockchainForSCW creates the minimal Blockchain row required by
// AddressDeployment's FK constraint.
func seedBlockchainForSCW(t *testing.T, db *gorm.DB) uint {
	t.Helper()
	bf := models.BlockchainFamily{Name: "EVM-SCW", Code: "EVM-SCW"}
	db.Create(&bf)
	bc := models.Blockchain{Code: "ETH-SCW", Name: "Ethereum", BlockchainFamilyID: bf.ID}
	if err := db.Create(&bc).Error; err != nil {
		t.Fatalf("seed blockchain: %v", err)
	}
	return bc.ID
}

// TestSCWBroadcaster_PicksUpPendingAndMarksDeployed verifies that the
// broadcaster finds a pending AddressDeployment row and transitions it to
// 'deployed' within one poll cycle.
func TestSCWBroadcaster_PicksUpPendingAndMarksDeployed(t *testing.T) {
	db := newSCWTestDB(t)
	bcID := seedBlockchainForSCW(t, db)

	deploymentRepo := repository.NewAddressDeploymentRepository(db)

	// Insert a pending deployment.
	d := &models.AddressDeployment{
		BlockchainID: bcID,
		Address:      "0xSCWAddress",
		Status:       "pending",
	}
	if err := deploymentRepo.Create(d); err != nil {
		t.Fatalf("create deployment: %v", err)
	}

	// Run the broadcaster with a very short interval.
	broadcaster := NewSCWDepositWalletBroadcaster(deploymentRepo, 20*time.Millisecond, 10)
	if broadcaster.Name() != "scw_deposit_wallet_broadcaster" {
		t.Errorf("Name(): got %q", broadcaster.Name())
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = broadcaster.Start(ctx)
	}()

	// Wait for at least one poll cycle to process the pending deployment.
	time.Sleep(100 * time.Millisecond)
	cancel()
	<-done

	// Verify the deployment was transitioned to 'deployed'.
	got, err := deploymentRepo.GetByID(d.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != "deployed" {
		t.Errorf("status: got %q want deployed", got.Status)
	}
}

// TestSCWBroadcaster_StopsOnContextCancel verifies that the worker exits
// cleanly when its context is cancelled, even with no pending deployments.
func TestSCWBroadcaster_StopsOnContextCancel(t *testing.T) {
	db := newSCWTestDB(t)
	deploymentRepo := repository.NewAddressDeploymentRepository(db)

	broadcaster := NewSCWDepositWalletBroadcaster(deploymentRepo, 50*time.Millisecond, 10)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- broadcaster.Start(ctx)
	}()

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Start() returned unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("broadcaster did not stop within 2 seconds after ctx cancellation")
	}
}

// TestSCWBroadcaster_NoPendingRows verifies that the broadcaster does not
// error when there are no pending deployments.
func TestSCWBroadcaster_NoPendingRows(t *testing.T) {
	db := newSCWTestDB(t)
	deploymentRepo := repository.NewAddressDeploymentRepository(db)

	broadcaster := NewSCWDepositWalletBroadcaster(deploymentRepo, 20*time.Millisecond, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()

	// Should exit cleanly after the timeout without panicking.
	if err := broadcaster.Start(ctx); err != nil {
		t.Errorf("Start() returned error: %v", err)
	}
}

// TestSCWBroadcaster_BatchLimit verifies that batchSize caps how many rows
// are processed per tick. We insert 5 pending deployments and use batchSize=3
// so the first tick processes exactly 3.
func TestSCWBroadcaster_BatchLimit(t *testing.T) {
	db := newSCWTestDB(t)
	bcID := seedBlockchainForSCW(t, db)
	deploymentRepo := repository.NewAddressDeploymentRepository(db)

	for i := range 5 {
		d := &models.AddressDeployment{
			BlockchainID: bcID,
			Address:      "0xAddr" + string(rune('A'+i)),
			Status:       "pending",
		}
		_ = deploymentRepo.Create(d)
	}

	// batchSize=3: first tick processes 3 rows.
	broadcaster := NewSCWDepositWalletBroadcaster(deploymentRepo, 20*time.Millisecond, 3)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = broadcaster.Start(ctx)
	}()

	// Wait for one full tick.
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	// At least 3 (and at most 5) should now be 'deployed'.
	deployed, err := deploymentRepo.ListByStatus("deployed")
	if err != nil {
		t.Fatalf("ListByStatus deployed: %v", err)
	}
	if len(deployed) < 3 {
		t.Errorf("expected at least 3 deployed, got %d", len(deployed))
	}
}
