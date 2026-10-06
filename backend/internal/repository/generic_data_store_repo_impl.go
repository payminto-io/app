package repository

import (
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GenericDataStoreRepository defines storage operations for the namespaced
// key-value GenericDataStore table.
type GenericDataStoreRepository interface {
	Get(namespace, key string) (*models.GenericDataStore, error)
	Set(namespace, key, value string, ttlSeconds *int64) error
	// Delete removes the entry for namespace+key. Returns the number of rows
	// deleted (0 means the key did not exist) and any database error.
	Delete(namespace, key string) (rowsAffected int64, err error)
	ListByNamespace(namespace string) ([]models.GenericDataStore, error)
	DeleteExpired() error
}

// GenericDataStoreRepositoryImpl is the GORM-backed implementation.
type GenericDataStoreRepositoryImpl struct {
	db *gorm.DB
}

// NewGenericDataStoreRepository constructs a GenericDataStoreRepositoryImpl.
func NewGenericDataStoreRepository(db *gorm.DB) GenericDataStoreRepository {
	return &GenericDataStoreRepositoryImpl{db: db}
}

// Get retrieves an entry by namespace+key. The expiry check is pushed into SQL
// so the result is always consistent regardless of concurrent writers.
// Returns gorm.ErrRecordNotFound when the key does not exist or has expired.
func (r *GenericDataStoreRepositoryImpl) Get(namespace, key string) (*models.GenericDataStore, error) {
	var entry models.GenericDataStore
	err := r.db.
		Where("namespace = ? AND key = ? AND (ttl IS NULL OR ttl = 0 OR ttl > ?)", namespace, key, time.Now().Unix()).
		First(&entry).Error
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

// Set upserts a namespace+key entry. ttlSeconds is the number of seconds from
// now until expiry; nil means no expiry. The TTL column stores the Unix expiry
// timestamp (not a duration) so comparisons are simple integer checks.
func (r *GenericDataStoreRepositoryImpl) Set(namespace, key, value string, ttlSeconds *int64) error {
	var expiry *int64
	if ttlSeconds != nil {
		t := time.Now().Unix() + *ttlSeconds
		expiry = &t
	}
	entry := models.GenericDataStore{
		Namespace: namespace,
		Key:       key,
		Value:     value,
		TTL:       expiry,
	}
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "namespace"}, {Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "ttl", "updated_at"}),
	}).Create(&entry).Error
}

// Delete removes the entry for the given namespace+key. Returns the number of
// rows deleted so callers can detect a concurrent delete (rowsAffected == 0).
func (r *GenericDataStoreRepositoryImpl) Delete(namespace, key string) (int64, error) {
	res := r.db.
		Where("namespace = ? AND key = ?", namespace, key).
		Delete(&models.GenericDataStore{})
	return res.RowsAffected, res.Error
}

// ListByNamespace returns all non-expired entries in a namespace.
func (r *GenericDataStoreRepositoryImpl) ListByNamespace(namespace string) ([]models.GenericDataStore, error) {
	var entries []models.GenericDataStore
	now := time.Now().Unix()
	err := r.db.
		Where("namespace = ? AND (ttl IS NULL OR ttl = 0 OR ttl > ?)", namespace, now).
		Find(&entries).Error
	return entries, err
}

// DeleteExpired removes all entries whose TTL has passed. Called periodically
// by a background janitor to keep the table tidy.
func (r *GenericDataStoreRepositoryImpl) DeleteExpired() error {
	now := time.Now().Unix()
	return r.db.
		Where("ttl IS NOT NULL AND ttl > 0 AND ttl <= ?", now).
		Delete(&models.GenericDataStore{}).Error
}
