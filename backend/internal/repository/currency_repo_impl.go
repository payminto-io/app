package repository

import (
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// CurrencyRepository defines the persistence contract for currency records.
type CurrencyRepository interface {
	Create(c *models.Currency) error
	Update(c *models.Currency) error
	GetByID(id uint) (*models.Currency, error)
	GetByCode(code string) (*models.Currency, error)
	List(opts ...QueryOption) ([]models.Currency, error)
	ListVisible() ([]models.Currency, error)
	UpdatePrice(code string, price decimal.Decimal) error
}

// CurrencyRepositoryImpl is the GORM-backed implementation of CurrencyRepository.
type CurrencyRepositoryImpl struct {
	db *gorm.DB
}

// NewCurrencyRepository constructs a CurrencyRepository backed by db.
func NewCurrencyRepository(db *gorm.DB) CurrencyRepository {
	return &CurrencyRepositoryImpl{db: db}
}

// Create inserts a new Currency row.
// Uses Select("*") to ensure zero-value bool fields (e.g. Visible=false) are written explicitly.
func (r *CurrencyRepositoryImpl) Create(c *models.Currency) error {
	return r.db.Select("*").Create(c).Error
}

// Update saves all fields on the currency (full save).
func (r *CurrencyRepositoryImpl) Update(c *models.Currency) error {
	return r.db.Save(c).Error
}

// GetByID fetches a currency by primary key.
func (r *CurrencyRepositoryImpl) GetByID(id uint) (*models.Currency, error) {
	var c models.Currency
	err := r.db.First(&c, id).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// GetByCode fetches a currency by its unique code.
func (r *CurrencyRepositoryImpl) GetByCode(code string) (*models.Currency, error) {
	var c models.Currency
	err := r.db.Where("code = ?", code).First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// List returns currencies with optional query options.
func (r *CurrencyRepositoryImpl) List(opts ...QueryOption) ([]models.Currency, error) {
	var currencies []models.Currency
	q := Apply(r.db, opts...)
	err := q.Find(&currencies).Error
	return currencies, err
}

// ListVisible returns all currencies where visible = true.
func (r *CurrencyRepositoryImpl) ListVisible() ([]models.Currency, error) {
	var currencies []models.Currency
	err := r.db.Where("visible = ?", true).Find(&currencies).Error
	return currencies, err
}

// UpdatePrice updates the price and updated_at for the currency with the given code.
func (r *CurrencyRepositoryImpl) UpdatePrice(code string, price decimal.Decimal) error {
	return r.db.Model(&models.Currency{}).
		Where("code = ?", code).
		Updates(map[string]any{
			"price":      price,
			"updated_at": time.Now(),
		}).Error
}
