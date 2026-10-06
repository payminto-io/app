package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/gorm"
)

// defaultPermissions is the canonical set of Payminto permissions. Each element
// is [name, displayName, description].
var defaultPermissions = [][3]string{
	{"payments.read", "Read Payments", "View payment requests and their statuses"},
	{"payments.create", "Create Payments", "Create new payment requests"},
	{"payments.delete", "Delete Payments", "Delete payment requests"},
	{"deposits.read", "Read Deposits", "View incoming deposit records"},
	{"wallets.read", "Read Wallets", "View wallet addresses and balances"},
	{"wallets.manage", "Manage Wallets", "Create and configure hot wallets"},
	{"withdrawals.read", "Read Withdrawals", "View withdrawal requests"},
	{"withdrawals.create", "Create Withdrawals", "Initiate withdrawal requests"},
	{"withdrawals.approve", "Approve Withdrawals", "Approve pending withdrawals"},
	{"sweeps.read", "Read Sweeps", "View sweep batch history"},
	{"sweeps.trigger", "Trigger Sweeps", "Manually trigger a sweep batch"},
	{"webhooks.manage", "Manage Webhooks", "Create, update, and delete webhook endpoints"},
	{"settings.read", "Read Settings", "View platform configuration"},
	{"settings.write", "Write Settings", "Modify platform configuration"},
	{"members.read", "Read Members", "View team member list"},
	{"members.invite", "Invite Members", "Invite new team members"},
	{"members.remove", "Remove Members", "Remove team members"},
	{"roles.manage", "Manage Roles", "Create, update, and assign roles"},
	{"analytics.read", "Read Analytics", "View analytics and reporting data"},
	{"system.admin", "System Admin", "Full system administration access"},
}

// PermissionService manages RBAC permissions and seeds the default set.
type PermissionService struct {
	permissionRepo repository.PermissionRepository
}

// NewPermissionService constructs a PermissionService backed by the given repository.
func NewPermissionService(permissionRepo repository.PermissionRepository) *PermissionService {
	return &PermissionService{permissionRepo: permissionRepo}
}

// SeedDefaults inserts the standard permission set if not already present.
// Idempotent — skips permissions that already exist by name.
func (s *PermissionService) SeedDefaults(ctx context.Context) error {
	for _, row := range defaultPermissions {
		name, displayName, desc := row[0], row[1], row[2]
		_, err := s.permissionRepo.GetByName(name)
		if err == nil {
			continue // already exists
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("permission seed lookup %q: %w", name, err)
		}
		p := &models.Permission{
			Name:        name,
			DisplayName: displayName,
			Description: desc,
		}
		if err := s.permissionRepo.Create(p); err != nil {
			return fmt.Errorf("permission seed create %q: %w", name, err)
		}
	}
	return nil
}

// GetByID returns a permission by primary key.
func (s *PermissionService) GetByID(_ context.Context, id uint) (*models.Permission, error) {
	return s.permissionRepo.GetByID(id)
}

// GetByName returns a permission by its resource.action name.
func (s *PermissionService) GetByName(_ context.Context, name string) (*models.Permission, error) {
	return s.permissionRepo.GetByName(name)
}

// ListAll returns every permission.
func (s *PermissionService) ListAll(_ context.Context) ([]models.Permission, error) {
	return s.permissionRepo.List()
}
