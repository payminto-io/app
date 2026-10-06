package repository

import "gorm.io/gorm"

// BaseRepository is an optional embeddable struct providing the common
// *gorm.DB handle. Concrete repository implementations may embed it or hold
// their own db field — both patterns are acceptable.
type BaseRepository struct {
	DB *gorm.DB
}

// WithDB returns a new BaseRepository bound to a transaction handle. Used by
// services that orchestrate multi-repository work inside a single GORM
// transaction (e.g., ledger operations that touch assets + liabilities +
// deposits atomically).
func (r *BaseRepository) WithDB(tx *gorm.DB) *BaseRepository {
	return &BaseRepository{DB: tx}
}
