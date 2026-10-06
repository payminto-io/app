package repository

import (
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ConfigurationRepository is the storage interface for runtime key-value
// configuration entries.
type ConfigurationRepository interface {
	Get(key string) (*models.Configuration, error)
	Set(key, value string) error
	SetWithDescription(key, value, description, category string) error
	Delete(key string) error
	List(opts ...QueryOption) ([]models.Configuration, error)
}

type ConfigurationRepositoryImpl struct {
	db *gorm.DB
}

func NewConfigurationRepository(db *gorm.DB) ConfigurationRepository {
	return &ConfigurationRepositoryImpl{db: db}
}

func (r *ConfigurationRepositoryImpl) Get(key string) (*models.Configuration, error) {
	var c models.Configuration
	if err := r.db.Where("key = ?", key).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *ConfigurationRepositoryImpl) Set(key, value string) error {
	c := models.Configuration{Key: key, Value: value}
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&c).Error
}

func (r *ConfigurationRepositoryImpl) SetWithDescription(key, value, description, category string) error {
	desc := description
	c := models.Configuration{
		Key:         key,
		Value:       value,
		Description: &desc,
		Category:    category,
	}
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "description", "category", "updated_at"}),
	}).Create(&c).Error
}

func (r *ConfigurationRepositoryImpl) Delete(key string) error {
	return r.db.Where("key = ?", key).Delete(&models.Configuration{}).Error
}

func (r *ConfigurationRepositoryImpl) List(opts ...QueryOption) ([]models.Configuration, error) {
	var out []models.Configuration
	q := Apply(r.db, opts...)
	if err := q.Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}
