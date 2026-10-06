package repository

import (
	"errors"

	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// ErrUTXOAlreadySpent is returned by MarkSpent when a concurrent caller
// already marked the UTXO spent. Callers should treat this as a benign
// "someone else got it" signal and skip the UTXO.
var ErrUTXOAlreadySpent = errors.New("utxo already spent")

// UTXORepository defines persistence operations for UTXO records used by the
// Bitcoin sweep pipeline.
type UTXORepository interface {
	Create(u *models.UTXO) error
	Update(u *models.UTXO) error
	GetByID(id uint) (*models.UTXO, error)
	GetByTxIDAndVout(txID string, vout uint) (*models.UTXO, error)
	ListUnspentByAddress(address string) ([]models.UTXO, error)
	ListUnspent(opts ...QueryOption) ([]models.UTXO, error)
	MarkSpent(id uint, spentTxID string) error
}

// UTXORepositoryImpl is the GORM-backed implementation of UTXORepository.
type UTXORepositoryImpl struct {
	db *gorm.DB
}

// NewUTXORepository constructs a new UTXORepository backed by the provided *gorm.DB.
func NewUTXORepository(db *gorm.DB) UTXORepository {
	return &UTXORepositoryImpl{db: db}
}

// Create inserts a new UTXO record.
func (r *UTXORepositoryImpl) Create(u *models.UTXO) error {
	return r.db.Create(u).Error
}

// Update saves all fields of the given UTXO.
func (r *UTXORepositoryImpl) Update(u *models.UTXO) error {
	return r.db.Save(u).Error
}

// GetByID fetches a UTXO by primary key.
func (r *UTXORepositoryImpl) GetByID(id uint) (*models.UTXO, error) {
	var u models.UTXO
	if err := r.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// GetByTxIDAndVout fetches a UTXO by its (txID, output-index) pair.
// Used for duplicate detection when processing new Bitcoin blocks.
func (r *UTXORepositoryImpl) GetByTxIDAndVout(txID string, vout uint) (*models.UTXO, error) {
	var u models.UTXO
	if err := r.db.
		Where("tx_id = ? AND vout = ?", txID, vout).
		First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// ListUnspentByAddress returns all unspent UTXOs for a given Bitcoin address.
func (r *UTXORepositoryImpl) ListUnspentByAddress(address string) ([]models.UTXO, error) {
	var utxos []models.UTXO
	if err := r.db.
		Where("address = ? AND spent = ?", address, false).
		Find(&utxos).Error; err != nil {
		return nil, err
	}
	return utxos, nil
}

// ListUnspent returns all unspent UTXOs, with optional query options for
// pagination or ordering.
func (r *UTXORepositoryImpl) ListUnspent(opts ...QueryOption) ([]models.UTXO, error) {
	q := Apply(r.db.Where("spent = ?", false), opts...)
	var utxos []models.UTXO
	if err := q.Find(&utxos).Error; err != nil {
		return nil, err
	}
	return utxos, nil
}

// MarkSpent atomically marks a UTXO as spent, recording the transaction ID
// that consumed it. The UPDATE is conditional on spent=false, so concurrent
// callers cannot double-spend the same output. If the UTXO was already spent
// (RowsAffected == 0), ErrUTXOAlreadySpent is returned — callers should treat
// this as a benign "someone else got it" signal and skip the UTXO.
func (r *UTXORepositoryImpl) MarkSpent(id uint, spentTxID string) error {
	res := r.db.Model(&models.UTXO{}).
		Where("id = ? AND spent = ?", id, false).
		Updates(map[string]any{
			"spent":       true,
			"spent_tx_id": spentTxID,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrUTXOAlreadySpent
	}
	return nil
}
