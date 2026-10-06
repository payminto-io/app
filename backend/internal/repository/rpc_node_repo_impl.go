package repository

import (
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// RPCNodeRepository defines the persistence contract for RPC node records.
type RPCNodeRepository interface {
	Create(n *models.RPCNode) error
	Update(n *models.RPCNode) error
	Delete(id uint) error
	GetByID(id uint) (*models.RPCNode, error)
	ListByBlockchain(blockchainID uint, opts ...QueryOption) ([]models.RPCNode, error)
	ListHealthyByBlockchain(blockchainID uint) ([]models.RPCNode, error)
	MarkUnhealthy(id uint, reason string) error
	MarkHealthy(id uint) error
	IncrementFailCount(id uint) error
	ResetFailCount(id uint) error
	TouchLastHealthCheck(id uint) error
	BulkCreate(nodes []models.RPCNode) error
}

// RPCNodeRepositoryImpl is the GORM-backed implementation of RPCNodeRepository.
type RPCNodeRepositoryImpl struct {
	db *gorm.DB
}

// NewRPCNodeRepository constructs an RPCNodeRepository backed by db.
func NewRPCNodeRepository(db *gorm.DB) RPCNodeRepository {
	return &RPCNodeRepositoryImpl{db: db}
}

// Create inserts a new RPCNode row.
func (r *RPCNodeRepositoryImpl) Create(n *models.RPCNode) error {
	return r.db.Create(n).Error
}

// Update saves all fields on the RPC node (full save).
func (r *RPCNodeRepositoryImpl) Update(n *models.RPCNode) error {
	return r.db.Save(n).Error
}

// Delete soft-deletes an RPCNode by primary key.
func (r *RPCNodeRepositoryImpl) Delete(id uint) error {
	return r.db.Delete(&models.RPCNode{}, id).Error
}

// GetByID fetches an RPCNode by primary key.
func (r *RPCNodeRepositoryImpl) GetByID(id uint) (*models.RPCNode, error) {
	var n models.RPCNode
	if err := r.db.First(&n, id).Error; err != nil {
		return nil, err
	}
	return &n, nil
}

// ListByBlockchain returns all RPC nodes for a blockchain ordered by priority, applying opts.
func (r *RPCNodeRepositoryImpl) ListByBlockchain(blockchainID uint, opts ...QueryOption) ([]models.RPCNode, error) {
	var out []models.RPCNode
	q := Apply(r.db.Where("blockchain_id = ?", blockchainID).Order("priority ASC"), opts...)
	if err := q.Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

// ListHealthyByBlockchain returns all healthy RPC nodes for a blockchain ordered by priority.
func (r *RPCNodeRepositoryImpl) ListHealthyByBlockchain(blockchainID uint) ([]models.RPCNode, error) {
	var out []models.RPCNode
	err := r.db.Where("blockchain_id = ? AND status = ?", blockchainID, models.RPCNodeStatusHealthy).
		Order("priority ASC").
		Find(&out).Error
	return out, err
}

// MarkUnhealthy sets status=unhealthy, records the error reason, and updates timestamps.
func (r *RPCNodeRepositoryImpl) MarkUnhealthy(id uint, reason string) error {
	now := time.Now()
	return r.db.Model(&models.RPCNode{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        models.RPCNodeStatusUnhealthy,
			"last_error":    reason,
			"last_error_at": now,
			"updated_at":    now,
		}).Error
}

// MarkHealthy sets status=healthy, resets fail_count, and records last_health_check.
func (r *RPCNodeRepositoryImpl) MarkHealthy(id uint) error {
	now := time.Now()
	return r.db.Model(&models.RPCNode{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":            models.RPCNodeStatusHealthy,
			"fail_count":        0,
			"last_health_check": now,
			"updated_at":        now,
		}).Error
}

// IncrementFailCount atomically increments the fail_count for the given RPC node.
func (r *RPCNodeRepositoryImpl) IncrementFailCount(id uint) error {
	return r.db.Model(&models.RPCNode{}).
		Where("id = ?", id).
		UpdateColumn("fail_count", gorm.Expr("fail_count + 1")).Error
}

// ResetFailCount sets fail_count to 0 for the given RPC node.
func (r *RPCNodeRepositoryImpl) ResetFailCount(id uint) error {
	return r.db.Model(&models.RPCNode{}).
		Where("id = ?", id).
		Update("fail_count", 0).Error
}

// TouchLastHealthCheck updates last_health_check to now for the given RPC node.
func (r *RPCNodeRepositoryImpl) TouchLastHealthCheck(id uint) error {
	now := time.Now()
	return r.db.Model(&models.RPCNode{}).
		Where("id = ?", id).
		Update("last_health_check", now).Error
}

// BulkCreate inserts multiple RPCNode rows in batches of 100.
func (r *RPCNodeRepositoryImpl) BulkCreate(nodes []models.RPCNode) error {
	if len(nodes) == 0 {
		return nil
	}
	return r.db.CreateInBatches(nodes, 100).Error
}
