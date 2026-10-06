package repository

import "gorm.io/gorm"

// QueryOption customizes a GORM query. Repositories accept variadic options
// so callers can mix and match pagination, preloading, ordering.
type QueryOption func(*gorm.DB) *gorm.DB

// WithLimit sets a LIMIT clause.
func WithLimit(n int) QueryOption {
	return func(db *gorm.DB) *gorm.DB {
		if n <= 0 {
			return db
		}
		return db.Limit(n)
	}
}

// WithOffset sets an OFFSET clause.
func WithOffset(n int) QueryOption {
	return func(db *gorm.DB) *gorm.DB {
		if n < 0 {
			return db
		}
		return db.Offset(n)
	}
}

// WithAscendingOrder orders by a column ascending.
func WithAscendingOrder(column string) QueryOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Order(column + " ASC")
	}
}

// WithDescendingOrder orders by a column descending.
func WithDescendingOrder(column string) QueryOption {
	return func(db *gorm.DB) *gorm.DB {
		return db.Order(column + " DESC")
	}
}

// WithPreload eagerly loads an association.
func WithPreload(association string, args ...any) QueryOption {
	return func(db *gorm.DB) *gorm.DB {
		if len(args) > 0 {
			return db.Preload(association, args...)
		}
		return db.Preload(association)
	}
}

// Apply threads every option through a fresh query scope.
func Apply(db *gorm.DB, opts ...QueryOption) *gorm.DB {
	for _, opt := range opts {
		db = opt(db)
	}
	return db
}
