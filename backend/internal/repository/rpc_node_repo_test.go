package repository

import (
	"testing"

	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupRPCNodeDB creates an in-memory SQLite DB with blockchain and rpc_node
// tables auto-migrated. It seeds one BlockchainFamily + Blockchain so that
// FK constraints on rpc_nodes(blockchain_id) pass.
func setupRPCNodeDB(t *testing.T) (*gorm.DB, RPCNodeRepository) {
	t.Helper()
	dsn := "file:" + t.Name() + "_rpc?mode=memory&cache=shared&_fk=1"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	if err := db.AutoMigrate(
		&models.BlockchainFamily{},
		&models.Blockchain{},
		&models.RPCNode{},
	); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	// Seed a blockchain row so RPCNode FK constraint (blockchain_id) passes.
	family := &models.BlockchainFamily{Name: "Ethereum", Code: "EVM"}
	if err := db.Create(family).Error; err != nil {
		t.Fatalf("seed BlockchainFamily: %v", err)
	}
	bc := &models.Blockchain{Code: "ETH", Name: "Ethereum Mainnet", BlockchainFamilyID: family.ID, Status: "active"}
	if err := db.Create(bc).Error; err != nil {
		t.Fatalf("seed Blockchain: %v", err)
	}

	return db, NewRPCNodeRepository(db)
}

// seedBCID returns the primary key of the seeded Blockchain.
func seedBCID(t *testing.T, db *gorm.DB) uint {
	t.Helper()
	var bc models.Blockchain
	if err := db.First(&bc).Error; err != nil {
		t.Fatalf("seedBCID: %v", err)
	}
	return bc.ID
}

func TestRPCNodeRepo_CreateAndGet(t *testing.T) {
	db, repo := setupRPCNodeDB(t)
	bcID := seedBCID(t, db)

	node := &models.RPCNode{
		BlockchainID: bcID,
		Name:         "PublicNode-ETH",
		URL:          "https://eth.publicnode.com",
		Priority:     10,
		Status:       models.RPCNodeStatusHealthy,
	}
	if err := repo.Create(node); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if node.ID == 0 {
		t.Fatal("expected non-zero ID after Create")
	}

	got, err := repo.GetByID(node.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "PublicNode-ETH" {
		t.Errorf("name mismatch: got %q want %q", got.Name, "PublicNode-ETH")
	}
	if got.URL != "https://eth.publicnode.com" {
		t.Errorf("URL mismatch: got %q want %q", got.URL, "https://eth.publicnode.com")
	}

	// Non-existent ID should return error.
	_, err = repo.GetByID(9999)
	if err == nil {
		t.Error("expected error for non-existent ID, got nil")
	}
}

func TestRPCNodeRepo_ListByBlockchain_OrdersPriority(t *testing.T) {
	db, repo := setupRPCNodeDB(t)
	bcID := seedBCID(t, db)

	// Create 3 nodes with different priorities — inserted out of order.
	for _, p := range []int{30, 10, 20} {
		node := &models.RPCNode{
			BlockchainID: bcID,
			Name:         "node",
			URL:          "https://example.com",
			Priority:     p,
			Status:       models.RPCNodeStatusHealthy,
		}
		if err := repo.Create(node); err != nil {
			t.Fatalf("Create priority=%d: %v", p, err)
		}
	}

	nodes, err := repo.ListByBlockchain(bcID)
	if err != nil {
		t.Fatalf("ListByBlockchain: %v", err)
	}
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}
	if nodes[0].Priority > nodes[1].Priority || nodes[1].Priority > nodes[2].Priority {
		t.Errorf("nodes not sorted by priority ASC: got %d, %d, %d",
			nodes[0].Priority, nodes[1].Priority, nodes[2].Priority)
	}
}

func TestRPCNodeRepo_ListHealthyByBlockchain(t *testing.T) {
	db, repo := setupRPCNodeDB(t)
	bcID := seedBCID(t, db)

	healthy1 := &models.RPCNode{BlockchainID: bcID, Name: "h1", URL: "https://h1.example", Status: models.RPCNodeStatusHealthy}
	healthy2 := &models.RPCNode{BlockchainID: bcID, Name: "h2", URL: "https://h2.example", Status: models.RPCNodeStatusHealthy}
	unhealthy := &models.RPCNode{BlockchainID: bcID, Name: "u1", URL: "https://u1.example", Status: models.RPCNodeStatusUnhealthy}

	for _, n := range []*models.RPCNode{healthy1, healthy2, unhealthy} {
		if err := repo.Create(n); err != nil {
			t.Fatalf("Create %s: %v", n.Name, err)
		}
	}

	got, err := repo.ListHealthyByBlockchain(bcID)
	if err != nil {
		t.Fatalf("ListHealthyByBlockchain: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 healthy nodes, got %d", len(got))
	}
	for _, n := range got {
		if n.Status != models.RPCNodeStatusHealthy {
			t.Errorf("unexpected status %q in healthy list", n.Status)
		}
	}
}

func TestRPCNodeRepo_MarkUnhealthy(t *testing.T) {
	db, repo := setupRPCNodeDB(t)
	bcID := seedBCID(t, db)

	node := &models.RPCNode{BlockchainID: bcID, Name: "node", URL: "https://x.example", Status: models.RPCNodeStatusHealthy}
	if err := repo.Create(node); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.MarkUnhealthy(node.ID, "connection timeout"); err != nil {
		t.Fatalf("MarkUnhealthy: %v", err)
	}

	updated, err := repo.GetByID(node.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if updated.Status != models.RPCNodeStatusUnhealthy {
		t.Errorf("expected status unhealthy, got %s", updated.Status)
	}
	if updated.LastError == nil || *updated.LastError != "connection timeout" {
		t.Errorf("expected last_error to be set, got %v", updated.LastError)
	}
	if updated.LastErrorAt == nil {
		t.Error("expected last_error_at to be set")
	}
}

func TestRPCNodeRepo_MarkHealthy_ResetsFailCount(t *testing.T) {
	db, repo := setupRPCNodeDB(t)
	bcID := seedBCID(t, db)

	node := &models.RPCNode{
		BlockchainID: bcID,
		Name:         "node",
		URL:          "https://x.example",
		Status:       models.RPCNodeStatusUnhealthy,
		FailCount:    5,
	}
	if err := repo.Create(node); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := repo.MarkHealthy(node.ID); err != nil {
		t.Fatalf("MarkHealthy: %v", err)
	}

	updated, err := repo.GetByID(node.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if updated.Status != models.RPCNodeStatusHealthy {
		t.Errorf("expected status healthy, got %s", updated.Status)
	}
	if updated.FailCount != 0 {
		t.Errorf("expected fail_count=0 after MarkHealthy, got %d", updated.FailCount)
	}
	if updated.LastHealthCheck == nil {
		t.Error("expected last_health_check to be set after MarkHealthy")
	}
}

func TestRPCNodeRepo_IncrementFailCount(t *testing.T) {
	db, repo := setupRPCNodeDB(t)
	bcID := seedBCID(t, db)

	node := &models.RPCNode{BlockchainID: bcID, Name: "node", URL: "https://x.example", FailCount: 0, Status: models.RPCNodeStatusHealthy}
	if err := repo.Create(node); err != nil {
		t.Fatalf("Create: %v", err)
	}

	for i := 1; i <= 3; i++ {
		if err := repo.IncrementFailCount(node.ID); err != nil {
			t.Fatalf("IncrementFailCount #%d: %v", i, err)
		}
	}

	updated, err := repo.GetByID(node.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if updated.FailCount != 3 {
		t.Errorf("expected fail_count=3, got %d", updated.FailCount)
	}
}

func TestRPCNodeRepo_BulkCreate(t *testing.T) {
	db, repo := setupRPCNodeDB(t)
	bcID := seedBCID(t, db)

	nodes := []models.RPCNode{
		{BlockchainID: bcID, Name: "bulk-a", URL: "https://a.example", Status: models.RPCNodeStatusHealthy},
		{BlockchainID: bcID, Name: "bulk-b", URL: "https://b.example", Status: models.RPCNodeStatusHealthy},
		{BlockchainID: bcID, Name: "bulk-c", URL: "https://c.example", Status: models.RPCNodeStatusHealthy},
	}
	if err := repo.BulkCreate(nodes); err != nil {
		t.Fatalf("BulkCreate: %v", err)
	}

	all, err := repo.ListByBlockchain(bcID)
	if err != nil {
		t.Fatalf("ListByBlockchain after BulkCreate: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 nodes after BulkCreate, got %d", len(all))
	}

	// BulkCreate with empty slice should be a no-op.
	if err := repo.BulkCreate(nil); err != nil {
		t.Fatalf("BulkCreate(nil): %v", err)
	}
}
