package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
	"gorm.io/gorm"
)

// defaultRoles defines the standard Payminto roles seeded at startup. Each
// element is [name, displayName, description, []permissionNames].
type defaultRole struct {
	name        string
	displayName string
	description string
	permissions []string
}

var standardRoles = []defaultRole{
	{
		name:        "owner",
		displayName: "Owner",
		description: "Full unrestricted access to all resources",
		permissions: []string{
			"payments.read", "payments.create", "payments.delete",
			"deposits.read",
			"wallets.read", "wallets.manage",
			"withdrawals.read", "withdrawals.create", "withdrawals.approve",
			"sweeps.read", "sweeps.trigger",
			"webhooks.manage",
			"settings.read", "settings.write",
			"members.read", "members.invite", "members.remove",
			"roles.manage",
			"analytics.read",
			"system.admin",
		},
	},
	{
		name:        "admin",
		displayName: "Admin",
		description: "Administrative access excluding system-level controls",
		permissions: []string{
			"payments.read", "payments.create", "payments.delete",
			"deposits.read",
			"wallets.read", "wallets.manage",
			"withdrawals.read", "withdrawals.create", "withdrawals.approve",
			"sweeps.read", "sweeps.trigger",
			"webhooks.manage",
			"settings.read", "settings.write",
			"members.read", "members.invite", "members.remove",
			"roles.manage",
			"analytics.read",
		},
	},
	{
		name:        "member",
		displayName: "Member",
		description: "Standard team member with create and view access",
		permissions: []string{
			"payments.read", "payments.create",
			"deposits.read",
			"wallets.read",
			"withdrawals.read", "withdrawals.create",
			"sweeps.read",
			"settings.read",
			"members.read",
			"analytics.read",
		},
	},
	{
		name:        "viewer",
		displayName: "Viewer",
		description: "Read-only access across all resources",
		permissions: []string{
			"payments.read",
			"deposits.read",
			"wallets.read",
			"withdrawals.read",
			"sweeps.read",
			"settings.read",
			"members.read",
			"analytics.read",
		},
	},
	{
		name:        "api-user",
		displayName: "API User",
		description: "Programmatic access for external integrations",
		permissions: []string{
			"payments.read", "payments.create",
			"deposits.read",
			"wallets.read",
			"withdrawals.read", "withdrawals.create",
		},
	},
}

// RoleService manages RBAC roles and their permission assignments.
type RoleService struct {
	roleRepo       repository.RoleRepository
	permissionRepo repository.PermissionRepository
}

// NewRoleService constructs a RoleService.
func NewRoleService(roleRepo repository.RoleRepository, permissionRepo repository.PermissionRepository) *RoleService {
	return &RoleService{roleRepo: roleRepo, permissionRepo: permissionRepo}
}

// SeedDefaults idempotently inserts the five standard roles with their
// default permission sets. Skips roles that already exist.
func (s *RoleService) SeedDefaults(ctx context.Context) error {
	for _, def := range standardRoles {
		_, err := s.roleRepo.GetByName(def.name)
		if err == nil {
			continue // already exists
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("role seed lookup %q: %w", def.name, err)
		}

		// Resolve permission IDs.
		perms, err := s.permissionRepo.GetByNames(def.permissions)
		if err != nil {
			return fmt.Errorf("role seed fetch permissions for %q: %w", def.name, err)
		}
		desc := def.description
		role := &models.Role{
			Name:        def.name,
			DisplayName: def.displayName,
			Description: &desc,
			Permissions: perms,
		}
		if err := s.roleRepo.Create(role); err != nil {
			return fmt.Errorf("role seed create %q: %w", def.name, err)
		}
	}
	return nil
}

// CreateRoleInput carries caller-supplied fields for a new role.
type CreateRoleInput struct {
	Name          string
	DisplayName   string
	Description   string
	PermissionIDs []uint
}

// CreateRole creates a new role and assigns the given permissions.
func (s *RoleService) CreateRole(_ context.Context, input CreateRoleInput) (*models.Role, error) {
	perms := make([]models.Permission, len(input.PermissionIDs))
	for i, pid := range input.PermissionIDs {
		perms[i] = models.Permission{PaymintoModel: models.PaymintoModel{ID: pid}}
	}
	desc := input.Description
	role := &models.Role{
		Name:        input.Name,
		DisplayName: input.DisplayName,
		Description: &desc,
		Permissions: perms,
	}
	if err := s.roleRepo.Create(role); err != nil {
		return nil, fmt.Errorf("create role: %w", err)
	}
	return role, nil
}

// GetByID returns a role with its permissions preloaded.
func (s *RoleService) GetByID(_ context.Context, id uint) (*models.Role, error) {
	return s.roleRepo.GetByID(id)
}

// GetByName returns a role by its unique name.
func (s *RoleService) GetByName(_ context.Context, name string) (*models.Role, error) {
	return s.roleRepo.GetByName(name)
}

// ListAll returns all roles with permissions preloaded.
func (s *RoleService) ListAll(_ context.Context) ([]models.Role, error) {
	return s.roleRepo.List()
}

// UpdatePermissions replaces the full permission set of a role.
func (s *RoleService) UpdatePermissions(_ context.Context, roleID uint, permissionIDs []uint) error {
	return s.roleRepo.UpdatePermissions(roleID, permissionIDs)
}
