package auth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/geoffjay/plantd/identity/pkg/client"
	log "github.com/sirupsen/logrus"
)

// AccessPattern defines common access patterns for the state service.
type AccessPattern string

const (
	// AccessPatternServiceOwner - Service that created a scope has full access
	AccessPatternServiceOwner AccessPattern = "service_owner"
	// AccessPatternCrossService - Explicit permission required for cross-service access
	AccessPatternCrossService AccessPattern = "cross_service"
	// AccessPatternAdmin - Administrative access with full permissions
	AccessPatternAdmin AccessPattern = "admin"
	// AccessPatternScoped - Scoped permissions within specific service boundaries
	AccessPatternScoped AccessPattern = "scoped"
)

// ServiceOwnership tracks which services created which scopes.
type ServiceOwnership struct {
	ScopeToOwner map[string]string // scope -> owner service
	mutex        sync.RWMutex
}

// NewServiceOwnership creates a new service ownership tracker.
func NewServiceOwnership() *ServiceOwnership {
	return &ServiceOwnership{
		ScopeToOwner: make(map[string]string),
	}
}

// SetOwner sets the owner of a scope.
func (so *ServiceOwnership) SetOwner(scope, owner string) {
	so.mutex.Lock()
	defer so.mutex.Unlock()
	so.ScopeToOwner[scope] = owner
}

// GetOwner returns the owner of a scope.
func (so *ServiceOwnership) GetOwner(scope string) (string, bool) {
	so.mutex.RLock()
	defer so.mutex.RUnlock()
	owner, exists := so.ScopeToOwner[scope]
	return owner, exists
}

// IsOwner checks if a service is the owner of a scope.
func (so *ServiceOwnership) IsOwner(scope, service string) bool {
	owner, exists := so.GetOwner(scope)
	return exists && owner == service
}

// RemoveOwnership removes ownership information for a scope.
func (so *ServiceOwnership) RemoveOwnership(scope string) {
	so.mutex.Lock()
	defer so.mutex.Unlock()
	delete(so.ScopeToOwner, scope)
}

// AccessChecker implements role-based access control patterns.
type AccessChecker struct {
	identityClient   *client.Client
	permissionCache  map[string]*CachedPermissions
	serviceOwnership *ServiceOwnership
	permissionUtils  *PermissionUtils
	inheritance      *PermissionInheritance
	cacheTTL         time.Duration
	cacheMutex       sync.RWMutex
	logger           *log.Logger
}

// AccessCheckerConfig holds configuration for the access checker.
type AccessCheckerConfig struct {
	IdentityClient *client.Client
	CacheTTL       time.Duration
	Logger         *log.Logger
}

// NewAccessChecker creates a new RBAC access checker.
func NewAccessChecker(config *AccessCheckerConfig) *AccessChecker {
	if config.CacheTTL == 0 {
		config.CacheTTL = 5 * time.Minute
	}

	if config.Logger == nil {
		config.Logger = log.New()
	}

	return &AccessChecker{
		identityClient:   config.IdentityClient,
		permissionCache:  make(map[string]*CachedPermissions),
		serviceOwnership: NewServiceOwnership(),
		permissionUtils:  NewPermissionUtils(),
		inheritance:      NewPermissionInheritance(),
		cacheTTL:         config.CacheTTL,
		logger:           config.Logger,
	}
}

// CheckScopeAccess implements the core access checking logic with multiple patterns.
func (ac *AccessChecker) CheckScopeAccess(userCtx *UserContext, operation, scope string) error {
	ac.logger.WithFields(log.Fields{
		"user_id":   userCtx.UserID,
		"operation": operation,
		"scope":     scope,
	}).Debug("Checking scope access")

	// Pattern 1: Check for global administrative permissions
	if ac.hasAdminAccess(userCtx, operation, scope) {
		ac.logger.WithFields(log.Fields{
			"user_id":   userCtx.UserID,
			"operation": operation,
			"scope":     scope,
			"pattern":   AccessPatternAdmin,
		}).Debug("Access granted via admin pattern")
		return nil
	}

	// Pattern 2: Check for global permissions
	if ac.hasGlobalPermission(userCtx, operation) {
		ac.logger.WithFields(log.Fields{
			"user_id":   userCtx.UserID,
			"operation": operation,
			"scope":     scope,
			"pattern":   AccessPatternScoped,
		}).Debug("Access granted via global permission")
		return nil
	}

	// Pattern 3: Check for scoped permissions
	if ac.hasScopedPermission(userCtx, operation, scope) {
		ac.logger.WithFields(log.Fields{
			"user_id":   userCtx.UserID,
			"operation": operation,
			"scope":     scope,
			"pattern":   AccessPatternScoped,
		}).Debug("Access granted via scoped permission")
		return nil
	}

	// Pattern 4: Check service ownership
	if ac.hasServiceOwnershipAccess(userCtx, operation, scope) {
		ac.logger.WithFields(log.Fields{
			"user_id":   userCtx.UserID,
			"operation": operation,
			"scope":     scope,
			"pattern":   AccessPatternServiceOwner,
		}).Debug("Access granted via service ownership")
		return nil
	}

	// Pattern 5: Check cross-service access (explicit permissions)
	if ac.hasCrossServiceAccess(userCtx, operation, scope) {
		ac.logger.WithFields(log.Fields{
			"user_id":   userCtx.UserID,
			"operation": operation,
			"scope":     scope,
			"pattern":   AccessPatternCrossService,
		}).Debug("Access granted via cross-service permission")
		return nil
	}

	ac.logger.WithFields(log.Fields{
		"user_id":   userCtx.UserID,
		"operation": operation,
		"scope":     scope,
	}).Warn("Access denied - no matching pattern")

	return &AuthenticationError{
		Code:    "PERMISSION_DENIED",
		Message: fmt.Sprintf("Insufficient permissions for %s on scope %s", operation, scope),
		Detail:  "No matching access pattern found",
	}
}

// hasAdminAccess checks for administrative access patterns.
func (ac *AccessChecker) hasAdminAccess(userCtx *UserContext, operation, scope string) bool { //nolint:revive
	// System admin has access to everything
	if ac.hasPermissionWithInheritance(userCtx, StateSystemAdmin) {
		return true
	}

	// Full admin has access to all operations
	if ac.hasPermissionWithInheritance(userCtx, StateAdminFull) {
		return true
	}

	// Scoped admin permissions
	if scope != "" {
		scopedAdminPerm := ac.permissionUtils.CreateScopedPermission(StateAdminFull, scope)
		if ac.hasPermissionWithInheritance(userCtx, scopedAdminPerm) {
			return true
		}
	}

	return false
}

// hasGlobalPermission checks for global permissions.
func (ac *AccessChecker) hasGlobalPermission(userCtx *UserContext, operation string) bool {
	return ac.hasPermissionWithInheritance(userCtx, operation)
}

// hasScopedPermission checks for scope-specific permissions.
func (ac *AccessChecker) hasScopedPermission(userCtx *UserContext, operation, scope string) bool {
	if scope == "" {
		return false
	}

	// Create scoped permission
	scopedPermission := ac.permissionUtils.CreateScopedPermission(operation, scope)
	return ac.hasPermissionWithInheritance(userCtx, scopedPermission)
}

// hasServiceOwnershipAccess checks service ownership access pattern.
func (ac *AccessChecker) hasServiceOwnershipAccess(userCtx *UserContext, operation, scope string) bool { //nolint:revive
	if scope == "" {
		return false
	}

	// Extract service from user context (this would need to be provided)
	// For now, we'll check if the user has service owner role
	// In a real implementation, this might check the user's associated service
	userService := ac.getUserService(userCtx)
	if userService == "" {
		return false
	}

	// Check if user's service owns the scope
	return ac.serviceOwnership.IsOwner(scope, userService)
}

// hasCrossServiceAccess checks cross-service access permissions.
func (ac *AccessChecker) hasCrossServiceAccess(userCtx *UserContext, operation, scope string) bool {
	if scope == "" {
		return false
	}

	// For cross-service access, require explicit scoped permissions
	// This prevents accidental access across service boundaries
	crossServicePerm := fmt.Sprintf("state:cross-service:%s:%s", scope, operation)
	return ac.hasPermissionWithInheritance(userCtx, crossServicePerm)
}

// hasPermissionWithInheritance checks permission with inheritance rules.
func (ac *AccessChecker) hasPermissionWithInheritance(userCtx *UserContext, requiredPermission string) bool {
	for _, userPerm := range userCtx.Permissions {
		if ac.inheritance.HasImpliedPermission(userPerm.Name, requiredPermission) {
			return true
		}
	}

	return false
}

// getUserService extracts the service associated with a user.
// This is a placeholder - in reality, this might come from JWT claims or user metadata.
func (ac *AccessChecker) getUserService(userCtx *UserContext) string { //nolint:revive
	// This could be enhanced to extract service information from:
	// 1. JWT token claims
	// 2. User profile metadata
	// 3. Organization membership
	// For now, return empty string
	return ""
}

// GrantCrossServiceAccess grants explicit cross-service access.
func (ac *AccessChecker) GrantCrossServiceAccess(ctx context.Context, userID uint, sourceService, targetScope, operation string) error { //nolint:revive
	ac.logger.WithFields(log.Fields{
		"user_id":        userID,
		"source_service": sourceService,
		"target_scope":   targetScope,
		"operation":      operation,
	}).Info("Granting cross-service access")

	// This would integrate with the identity service to grant permissions
	// For now, this is a placeholder
	permission := fmt.Sprintf("state:cross-service:%s:%s", targetScope, operation)

	// In a real implementation, this would call the identity service
	// to assign the permission to the user
	ac.logger.WithFields(log.Fields{
		"user_id":    userID,
		"permission": permission,
	}).Info("Cross-service permission would be granted via identity service")

	return nil
}

// RevokeCrossServiceAccess revokes cross-service access.
func (ac *AccessChecker) RevokeCrossServiceAccess(ctx context.Context, userID uint, sourceService, targetScope, operation string) error { //nolint:revive
	ac.logger.WithFields(log.Fields{
		"user_id":        userID,
		"source_service": sourceService,
		"target_scope":   targetScope,
		"operation":      operation,
	}).Info("Revoking cross-service access")

	permission := fmt.Sprintf("state:cross-service:%s:%s", targetScope, operation)

	// In a real implementation, this would call the identity service
	// to remove the permission from the user
	ac.logger.WithFields(log.Fields{
		"user_id":    userID,
		"permission": permission,
	}).Info("Cross-service permission would be revoked via identity service")

	return nil
}

// RegisterScopeOwnership registers ownership of a scope to a service.
func (ac *AccessChecker) RegisterScopeOwnership(scope, ownerService string) {
	ac.serviceOwnership.SetOwner(scope, ownerService)
	ac.logger.WithFields(log.Fields{
		"scope":         scope,
		"owner_service": ownerService,
	}).Info("Registered scope ownership")
}

// UnregisterScopeOwnership removes ownership of a scope.
func (ac *AccessChecker) UnregisterScopeOwnership(scope string) {
	ac.serviceOwnership.RemoveOwnership(scope)
	ac.logger.WithFields(log.Fields{
		"scope": scope,
	}).Info("Unregistered scope ownership")
}

// GetScopeOwner returns the owner of a scope.
func (ac *AccessChecker) GetScopeOwner(scope string) (string, bool) {
	return ac.serviceOwnership.GetOwner(scope)
}

// ValidateAccessPattern validates that an access pattern is correctly configured.
func (ac *AccessChecker) ValidateAccessPattern(pattern AccessPattern, userCtx *UserContext, operation, scope string) error { //nolint:revive
	switch pattern {
	case AccessPatternServiceOwner:
		if scope == "" {
			return fmt.Errorf("service owner pattern requires scope")
		}
		if _, exists := ac.serviceOwnership.GetOwner(scope); !exists {
			return fmt.Errorf("no owner registered for scope: %s", scope)
		}
	case AccessPatternCrossService:
		if scope == "" {
			return fmt.Errorf("cross-service pattern requires scope")
		}
	case AccessPatternAdmin:
		// Admin pattern is always valid
	case AccessPatternScoped:
		if scope == "" {
			return fmt.Errorf("scoped pattern requires scope")
		}
	default:
		return fmt.Errorf("unknown access pattern: %s", pattern)
	}

	return nil
}

// GetEffectivePermissions returns all effective permissions for a user in a scope.
func (ac *AccessChecker) GetEffectivePermissions(userCtx *UserContext, scope string) []string {
	var effective []string
	permissionSet := make(map[string]bool)

	// Add direct permissions with inheritance
	for _, userPerm := range userCtx.Permissions {
		permissionSet[userPerm.Name] = true

		// Add implied permissions
		implied := ac.inheritance.GetImpliedPermissions(userPerm.Name)
		for _, impliedPerm := range implied {
			permissionSet[impliedPerm] = true
		}
	}

	// Add service ownership permissions if applicable
	if scope != "" {
		userService := ac.getUserService(userCtx)
		if userService != "" && ac.serviceOwnership.IsOwner(scope, userService) {
			ownerPermissions := []string{
				StateDataRead,
				StateDataWrite,
				StateDataDelete,
				StateHealthRead,
			}
			for _, perm := range ownerPermissions {
				permissionSet[perm] = true
			}
		}
	}

	// Convert set to slice
	for perm := range permissionSet {
		effective = append(effective, perm)
	}

	return effective
}

// ClearCache clears the permission cache.
func (ac *AccessChecker) ClearCache() {
	ac.cacheMutex.Lock()
	defer ac.cacheMutex.Unlock()
	ac.permissionCache = make(map[string]*CachedPermissions)
	ac.logger.Info("RBAC permission cache cleared")
}

// GetCacheStats returns cache statistics.
func (ac *AccessChecker) GetCacheStats() map[string]interface{} {
	ac.cacheMutex.RLock()
	defer ac.cacheMutex.RUnlock()

	stats := map[string]interface{}{
		"total_entries": len(ac.permissionCache),
		"cache_ttl":     ac.cacheTTL.String(),
	}

	// Count expired entries
	expired := 0
	now := time.Now()
	for _, cached := range ac.permissionCache {
		if now.After(cached.ExpiresAt) {
			expired++
		}
	}
	stats["expired_entries"] = expired

	return stats
}
