package repository

import (
	"github.com/payminto/payminto/backend/internal/models"
	"gorm.io/gorm"
)

// RoleRepository defines all data access operations for the Role model.
type RoleRepository interface {
	Create(r *models.Role) error
	GetByID(id uint) (*models.Role, error)
	GetByName(name string) (*models.Role, error)
	GetByNames(names []string) ([]models.Role, error)
	GetByIDs(ids []uint) ([]models.Role, error)
	List() ([]models.Role, error)
	UpdatePermissions(roleID uint, permissionIDs []uint) error
}

// RoleRepositoryImpl is the GORM-backed implementation of RoleRepository.
type RoleRepositoryImpl struct {
	db *gorm.DB
}

// NewRoleRepository constructs a RoleRepositoryImpl bound to the given DB handle.
func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &RoleRepositoryImpl{db: db}
}

func (r *RoleRepositoryImpl) Create(role *models.Role) error {
	return r.db.Create(role).Error
}

// GetByID returns a Role with its Permissions association preloaded.
func (r *RoleRepositoryImpl) GetByID(id uint) (*models.Role, error) {
	var role models.Role
	err := r.db.Preload("Permissions").First(&role, id).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// GetByName returns a Role (with Permissions preloaded) matching the given name.
func (r *RoleRepositoryImpl) GetByName(name string) (*models.Role, error) {
	var role models.Role
	err := r.db.Preload("Permissions").Where("name = ?", name).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// GetByNames returns all roles whose name is in the provided slice.
func (r *RoleRepositoryImpl) GetByNames(names []string) ([]models.Role, error) {
	var roles []models.Role
	err := r.db.Where("name IN ?", names).Find(&roles).Error
	return roles, err
}

// GetByIDs returns all roles whose ID is in the provided slice.
func (r *RoleRepositoryImpl) GetByIDs(ids []uint) ([]models.Role, error) {
	var roles []models.Role
	err := r.db.Where("id IN ?", ids).Find(&roles).Error
	return roles, err
}

// List returns all roles with their Permissions preloaded.
func (r *RoleRepositoryImpl) List() ([]models.Role, error) {
	var roles []models.Role
	err := r.db.Preload("Permissions").Find(&roles).Error
	return roles, err
}

// UpdatePermissions replaces the entire set of permissions for a role inside a
// single transaction: it clears the existing role_permissions join rows and then
// inserts the new ones.
func (r *RoleRepositoryImpl) UpdatePermissions(roleID uint, permissionIDs []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var role models.Role
		if err := tx.First(&role, roleID).Error; err != nil {
			return err
		}

		// Build the permission slice from the provided IDs.
		permissions := make([]models.Permission, len(permissionIDs))
		for i, pid := range permissionIDs {
			permissions[i] = models.Permission{PaymintoModel: models.PaymintoModel{ID: pid}}
		}

		// Replace replaces the current associations with the new set.
		return tx.Model(&role).Association("Permissions").Replace(permissions)
	})
}
