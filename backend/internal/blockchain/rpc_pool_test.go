package blockchain

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupPoolTestDB creates an isolated in-memory SQLite DB with blockchain and
// rpc_node tables, seeds one Blockchain, and returns the repo + its ID.
func setupPoolTestDB(t *testing.T) (*gorm.DB, repository.RPCNodeRepository) {
	t.Helper()
	dsn := "file:" + t.Name() + "_pool?mode=memory&cache=shared&_fk=1"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.BlockchainFamily{}, &models.Blockchain{}, &models.RPCNode{}); err != nil {
		t.Fatal(err)
	}

	// Seed FK dependency: BlockchainFamily + Blockchain.
	family := &models.BlockchainFamily{Name: "Ethereum", Code: "EVM"}
	if err := db.Create(family).Error; err != nil {
		t.Fatalf("seed family: %v", err)
	}
	bc := &models.Blockchain{Code: "ETH", Name: "Ethereum Mainnet", BlockchainFamilyID: family.ID, Status: "active"}
	if err := db.Create(bc).Error; err != nil {
		t.Fatalf("seed blockchain: %v", err)
	}

	return db, repository.NewRPCNodeRepository(db)
}

// seedBCIDPool fetches the primary key of the seeded Blockchain.
func seedBCIDPool(t *testing.T, db *gorm.DB) uint {
	t.Helper()
	var bc models.Blockchain
	if err := db.First(&bc).Error; err != nil {
		t.Fatalf("seedBCIDPool: %v", err)
	}
	return bc.ID
}

func TestRPCPool_Empty(t *testing.T) {
	_, repo := setupPoolTestDB(t)
	pool := NewRPCPool(999, repo)
	_ = pool.Refresh()
	if _, err := pool.Pick(); err == nil {
		t.Error("expected error on empty pool")
	}
}

func TestRPCPool_PickRoundRobin(t *testing.T) {
	db, repo := setupPoolTestDB(t)
	bcID := seedBCIDPool(t, db)

	_ = repo.Create(&models.RPCNode{BlockchainID: bcID, Name: "node-a", URL: "https://a.example", Priority: 10, Status: models.RPCNodeStatusHealthy})
	_ = repo.Create(&models.RPCNode{BlockchainID: bcID, Name: "node-b", URL: "https://b.example", Priority: 20, Status: models.RPCNodeStatusHealthy})

	pool := NewRPCPool(bcID, repo)
	if err := pool.Refresh(); err != nil {
		t.Fatal(err)
	}
	if pool.Size() != 2 {
		t.Errorf("expected 2 nodes, got %d", pool.Size())
	}

	first, _ := pool.Pick()
	second, _ := pool.Pick()
	third, _ := pool.Pick()

	if first.Name == second.Name {
		t.Error("expected different nodes on consecutive picks")
	}
	if third.Name != first.Name {
		t.Error("expected round-robin to loop back to first")
	}
}

func TestRPCPool_MarkFailure_EvictsAfterThreshold(t *testing.T) {
	db, repo := setupPoolTestDB(t)
	bcID := seedBCIDPool(t, db)

	_ = repo.Create(&models.RPCNode{BlockchainID: bcID, Name: "node-a", URL: "https://a.example", Status: models.RPCNodeStatusHealthy})
	_ = repo.Create(&models.RPCNode{BlockchainID: bcID, Name: "node-b", URL: "https://b.example", Status: models.RPCNodeStatusHealthy})

	pool := NewRPCPool(bcID, repo).WithFailThreshold(2)
	_ = pool.Refresh()

	nodes, _ := repo.ListByBlockchain(bcID)
	targetID := nodes[0].ID

	pool.MarkFailure(targetID, errors.New("connection refused"))
	pool.MarkFailure(targetID, errors.New("connection refused"))

	// After 2 failures, the node should be unhealthy and evicted.
	if pool.Size() != 1 {
		t.Errorf("expected pool to shrink to 1 after failure threshold, got %d", pool.Size())
	}

	updated, _ := repo.GetByID(targetID)
	if updated.Status != models.RPCNodeStatusUnhealthy {
		t.Errorf("expected status unhealthy, got %s", updated.Status)
	}
}

func TestRPCPool_HealthCheck_RestoresCooled(t *testing.T) {
	db, repo := setupPoolTestDB(t)
	bcID := seedBCIDPool(t, db)

	// Create a pre-failed node with an old last_error_at.
	past := time.Now().Add(-10 * time.Minute)
	reason := "old failure"
	node := &models.RPCNode{
		BlockchainID: bcID,
		Name:         "node-recovered",
		URL:          "https://r.example",
		Status:       models.RPCNodeStatusUnhealthy,
		FailCount:    5,
		LastErrorAt:  &past,
		LastError:    &reason,
	}
	_ = repo.Create(node)

	pool := NewRPCPool(bcID, repo).WithCooldown(5 * time.Minute)
	_ = pool.Refresh()
	if pool.Size() != 0 {
		t.Errorf("pool should start empty (only healthy nodes loaded), got %d", pool.Size())
	}

	if err := pool.HealthCheck(); err != nil {
		t.Fatal(err)
	}

	if pool.Size() != 1 {
		t.Errorf("expected node to be restored after cooldown, got size %d", pool.Size())
	}
}

func TestRPCPool_MarkSuccess_ResetsFailCount(t *testing.T) {
	db, repo := setupPoolTestDB(t)
	bcID := seedBCIDPool(t, db)

	_ = repo.Create(&models.RPCNode{BlockchainID: bcID, Name: "node", URL: "https://a.example", FailCount: 2, Status: models.RPCNodeStatusHealthy})

	nodes, _ := repo.ListByBlockchain(bcID)
	pool := NewRPCPool(bcID, repo)
	pool.MarkSuccess(nodes[0].ID)

	updated, _ := repo.GetByID(nodes[0].ID)
	if updated.FailCount != 0 {
		t.Errorf("expected fail_count=0 after success, got %d", updated.FailCount)
	}
}

// TestRPCPool_Pick_ConcurrentSafe verifies that Pick is race-free when called
// from many goroutines simultaneously. Run with -race to catch data races.
func TestRPCPool_Pick_ConcurrentSafe(t *testing.T) {
	_, repo := setupPoolTestDB(t)
	_ = repo.Create(&models.RPCNode{BlockchainID: 1, Name: "a", URL: "http://a", Status: models.RPCNodeStatusHealthy})
	_ = repo.Create(&models.RPCNode{BlockchainID: 1, Name: "b", URL: "http://b", Status: models.RPCNodeStatusHealthy})
	_ = repo.Create(&models.RPCNode{BlockchainID: 1, Name: "c", URL: "http://c", Status: models.RPCNodeStatusHealthy})

	pool := NewRPCPool(1, repo)
	if err := pool.Refresh(); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			for range 100 {
				_, err := pool.Pick()
				if err != nil {
					t.Errorf("pick: %v", err)
					return
				}
			}
		})
	}
	wg.Wait()
}

// TestRPCPool_Pick_RoundRobinFair verifies that over many picks each node
// receives an equal share, confirming strict round-robin distribution.
func TestRPCPool_Pick_RoundRobinFair(t *testing.T) {
	_, repo := setupPoolTestDB(t)
	for i := range 4 {
		_ = repo.Create(&models.RPCNode{
			BlockchainID: 1,
			Name:         fmt.Sprintf("node-%d", i),
			URL:          fmt.Sprintf("http://%d", i),
			Status:       models.RPCNodeStatusHealthy,
		})
	}
	pool := NewRPCPool(1, repo)
	_ = pool.Refresh()

	counts := map[string]int{}
	for range 1000 {
		n, _ := pool.Pick()
		counts[n.Name]++
	}
	if len(counts) != 4 {
		t.Errorf("expected 4 distinct nodes, got %d", len(counts))
	}
	for name, c := range counts {
		if c != 250 {
			t.Errorf("expected 250 picks for %s, got %d", name, c)
		}
	}
}
