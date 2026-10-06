package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/payminto/payminto/backend/internal/models"
	"github.com/payminto/payminto/backend/internal/repository"
)

// permCacheEntry caches the set of permission names for a member+platform pair.
type permCacheEntry struct {
	permissions map[string]struct{}
	expiresAt   time.Time
}

const permCacheTTL = 60 * time.Second

// MemberExternalPlatformRoleService manages per-project role assignments and
// provides the HasPermission hot path used by the RBAC middleware.
type MemberExternalPlatformRoleService struct {
	repo repository.MemberExternalPlatformRoleRepository

	mu    sync.RWMutex
	cache map[string]*permCacheEntry // key: "memberID:platformID"
}

// NewMemberExternalPlatformRoleService constructs the service.
func NewMemberExternalPlatformRoleService(repo repository.MemberExternalPlatformRoleRepository) *MemberExternalPlatformRoleService {
	return &MemberExternalPlatformRoleService{
		repo:  repo,
		cache: make(map[string]*permCacheEntry),
	}
}

// Assign grants a role to a member on a platform.
func (s *MemberExternalPlatformRoleService) Assign(_ context.Context, memberID, platformID, roleID uint) error {
	rec := &models.MemberExternalPlatformRole{
		MemberID:           memberID,
		ExternalPlatformID: platformID,
		RoleID:             roleID,
	}
	if err := s.repo.Assign(rec); err != nil {
		return fmt.Errorf("assign role: %w", err)
	}
	s.invalidateCache(memberID, platformID)
	return nil
}

// Revoke removes a role from a member on a platform.
func (s *MemberExternalPlatformRoleService) Revoke(_ context.Context, memberID, platformID, roleID uint) error {
	if err := s.repo.Revoke(memberID, platformID, roleID); err != nil {
		return fmt.Errorf("revoke role: %w", err)
	}
	s.invalidateCache(memberID, platformID)
	return nil
}

// ListByMember returns all role assignments for a member across platforms.
func (s *MemberExternalPlatformRoleService) ListByMember(_ context.Context, memberID uint) ([]models.MemberExternalPlatformRole, error) {
	return s.repo.ListByMember(memberID)
}

// ListByPlatform returns all role assignments on a platform.
func (s *MemberExternalPlatformRoleService) ListByPlatform(_ context.Context, platformID uint) ([]models.MemberExternalPlatformRole, error) {
	return s.repo.ListByPlatform(platformID)
}

// HasPermission returns true when the member holds at least one role on the
// platform that includes the named permission. Results are cached for 60 s to
// avoid hammering the DB on every authenticated request.
func (s *MemberExternalPlatformRoleService) HasPermission(ctx context.Context, memberID, platformID uint, permission string) bool {
	perms, err := s.permissionsFor(ctx, memberID, platformID)
	if err != nil {
		log.Printf("[rbac] permission check for member %d platform %d: %v", memberID, platformID, err)
		return false
	}
	_, ok := perms[permission]
	return ok
}

// permissionsFor returns the full permission set for a member+platform from
// cache or the DB.
func (s *MemberExternalPlatformRoleService) permissionsFor(ctx context.Context, memberID, platformID uint) (map[string]struct{}, error) {
	cacheKey := fmt.Sprintf("%d:%d", memberID, platformID)

	// Fast path: cache hit.
	s.mu.RLock()
	entry, ok := s.cache[cacheKey]
	if ok && time.Now().Before(entry.expiresAt) {
		perms := entry.permissions
		s.mu.RUnlock()
		return perms, nil
	}
	s.mu.RUnlock()

	// Slow path: load from DB.
	roles, err := s.repo.GetRolesByMemberAndPlatform(memberID, platformID)
	if err != nil {
		return nil, fmt.Errorf("load roles: %w", err)
	}

	perms := make(map[string]struct{})
	for _, role := range roles {
		for _, p := range role.Permissions {
			perms[p.Name] = struct{}{}
		}
	}

	s.mu.Lock()
	s.cache[cacheKey] = &permCacheEntry{
		permissions: perms,
		expiresAt:   time.Now().Add(permCacheTTL),
	}
	s.mu.Unlock()

	return perms, nil
}

// invalidateCache removes the cached permission set for a member+platform pair
// so subsequent calls reload from the DB.
func (s *MemberExternalPlatformRoleService) invalidateCache(memberID, platformID uint) {
	key := fmt.Sprintf("%d:%d", memberID, platformID)
	s.mu.Lock()
	delete(s.cache, key)
	s.mu.Unlock()
}
