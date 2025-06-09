package auth

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/geoffjay/plantd/identity/internal/models"
	"github.com/geoffjay/plantd/identity/internal/repositories"
)

// RBACService implements role-based access control
type RBACService struct {
	userRepo         repositories.UserRepository
	roleRepo         repositories.RoleRepository
	organizationRepo repositories.OrganizationRepository
	logger           *slog.Logger
	permissionCache  map[string][]Permission // Simple in-memory cache
	cacheExpiry      time.Duration
}

// NewRBACService creates a new RBAC service
func NewRBACService(
	userRepo repositories.UserRepository,
	roleRepo repositories.RoleRepository,
	organizationRepo repositories.OrganizationRepository,
	logger *slog.Logger,
) *RBACService {
	return &RBACService{
		userRepo:         userRepo,
		roleRepo:         roleRepo,
		organizationRepo: organizationRepo,
		logger:           logger,
		permissionCache:  make(map[string][]Permission),
		cacheExpiry:      5 * time.Minute, // Cache permissions for 5 minutes
	}
}

// HasPermission checks if a user has a specific permission
func (r *RBACService) HasPermission(ctx context.Context, userID uint, permission Permission, orgID *uint) (bool, error) {
	r.logger.Debug("Checking permission",
		"user_id", userID,
		"permission", permission,
		"org_id", orgID)

	// Get user permissions
	userPermissions, err := r.GetUserPermissions(ctx, userID, orgID)
	if err != nil {
		r.logger.Error("Failed to get user permissions",
			"error", err,
			"user_id", userID,
			"org_id", orgID)
		return false, fmt.Errorf("failed to get user permissions: %w", err)
	}

	// Check if user has the specific permission
	for _, perm := range userPermissions {
		if perm == permission {
			r.logger.Debug("Permission granted",
				"user_id", userID,
				"permission", permission,
				"org_id", orgID)
			return true, nil
		}
	}

	// Check for admin permissions
	for _, perm := range userPermissions {
		if perm == PermissionSystemAdmin {
			r.logger.Debug("Permission granted via system admin",
				"user_id", userID,
				"permission", permission)
			return true, nil
		}
		// Organization admin has all organization permissions
		if orgID != nil && perm == PermissionOrganizationAdmin && permission.Category() == CategoryOrganization {
			r.logger.Debug("Permission granted via organization admin",
				"user_id", userID,
				"permission", permission,
				"org_id", orgID)
			return true, nil
		}
	}

	r.logger.Debug("Permission denied",
		"user_id", userID,
		"permission", permission,
		"org_id", orgID)
	return false, nil
}

// HasAnyPermission checks if a user has any of the specified permissions
func (r *RBACService) HasAnyPermission(ctx context.Context, userID uint, permissions []Permission, orgID *uint) (bool, error) {
	for _, permission := range permissions {
		hasPermission, err := r.HasPermission(ctx, userID, permission, orgID)
		if err != nil {
			return false, err
		}
		if hasPermission {
			return true, nil
		}
	}
	return false, nil
}

// HasAllPermissions checks if a user has all of the specified permissions
func (r *RBACService) HasAllPermissions(ctx context.Context, userID uint, permissions []Permission, orgID *uint) (bool, error) {
	for _, permission := range permissions {
		hasPermission, err := r.HasPermission(ctx, userID, permission, orgID)
		if err != nil {
			return false, err
		}
		if !hasPermission {
			return false, nil
		}
	}
	return true, nil
}

// GetUserPermissions returns all permissions for a user in a given context
func (r *RBACService) GetUserPermissions(ctx context.Context, userID uint, orgID *uint) ([]Permission, error) {
	cacheKey := r.buildCacheKey(userID, orgID)

	// Check cache first (simple implementation)
	if cachedPermissions, exists := r.permissionCache[cacheKey]; exists {
		return cachedPermissions, nil
	}

	// Get user with roles
	user, err := r.userRepo.GetWithRoles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	var allPermissions []Permission
	permissionSet := make(map[Permission]bool)

	// Process roles
	for _, role := range user.Roles {
		// Skip roles that don't match the context
		if !r.roleMatchesContext(role, orgID) {
			continue
		}

		rolePermissions, err := role.GetPermissions()
		if err != nil {
			r.logger.Error("Failed to get role permissions",
				"error", err,
				"role_id", role.ID,
				"role_name", role.Name)
			continue
		}

		// Add permissions to set (avoiding duplicates)
		for _, permStr := range rolePermissions {
			permission := Permission(permStr)
			if !permissionSet[permission] {
				permissionSet[permission] = true
				allPermissions = append(allPermissions, permission)
			}
		}
	}

	// If in organization context, also check organization-specific roles
	if orgID != nil {
		orgRolePointers, err := r.roleRepo.GetByUserAndOrganization(ctx, userID, *orgID, 0, 1000)
		if err != nil {
			r.logger.Error("Failed to get organization roles",
				"error", err,
				"user_id", userID,
				"org_id", orgID)
		} else {
			// Convert from []*models.Role to []models.Role
			for _, rolePtr := range orgRolePointers {
				if rolePtr != nil {
					role := *rolePtr
					rolePermissions, err := role.GetPermissions()
					if err != nil {
						r.logger.Error("Failed to get role permissions",
							"error", err,
							"role_id", role.ID,
							"role_name", role.Name)
						continue
					}

					for _, permStr := range rolePermissions {
						permission := Permission(permStr)
						if !permissionSet[permission] {
							permissionSet[permission] = true
							allPermissions = append(allPermissions, permission)
						}
					}
				}
			}
		}
	}

	// Cache the permissions
	r.permissionCache[cacheKey] = allPermissions

	r.logger.Debug("Retrieved user permissions",
		"user_id", userID,
		"org_id", orgID,
		"permission_count", len(allPermissions))

	return allPermissions, nil
}

// CanAccessResource checks if a user can access a specific resource
func (r *RBACService) CanAccessResource(ctx context.Context, authCtx *AuthorizationContext) (bool, error) {
	// Get required permissions for the resource/action combination
	operationKey := fmt.Sprintf("%s.%s", authCtx.Resource, authCtx.Action)
	requiredPermissions, exists := RequiredPermissions[operationKey]

	if !exists {
		// If no specific permissions defined, allow access for authenticated users
		r.logger.Debug("No specific permissions required for operation",
			"operation", operationKey,
			"user_id", authCtx.UserID)
		return true, nil
	}

	// Check if user has any of the required permissions
	hasPermission, err := r.HasAnyPermission(ctx, authCtx.UserID, requiredPermissions, authCtx.OrganizationID)
	if err != nil {
		return false, err
	}

	if !hasPermission {
		r.logger.Warn("Access denied",
			"user_id", authCtx.UserID,
			"resource", authCtx.Resource,
			"action", authCtx.Action,
			"org_id", authCtx.OrganizationID,
			"required_permissions", requiredPermissions)
		return false, &UnauthorizedError{
			UserID:   authCtx.UserID,
			Resource: authCtx.Resource,
			Action:   authCtx.Action,
			Message:  "insufficient permissions",
		}
	}

	return true, nil
}

// AssignRoleToUser assigns a role to a user
func (r *RBACService) AssignRoleToUser(ctx context.Context, userID, roleID uint, orgID *uint) error {
	r.logger.Info("Assigning role to user",
		"user_id", userID,
		"role_id", roleID,
		"org_id", orgID)

	// Get the role to check its scope
	role, err := r.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		return fmt.Errorf("failed to get role: %w", err)
	}

	// Validate scope compatibility
	if role.IsGlobal() && orgID != nil {
		return fmt.Errorf("cannot assign global role within organization context")
	}
	if role.IsOrganizationScoped() && orgID == nil {
		return fmt.Errorf("cannot assign organization-scoped role without organization context")
	}

	// For organization-scoped roles, verify user is member of organization
	if orgID != nil {
		isMember, err := r.isUserOrganizationMember(ctx, userID, *orgID)
		if err != nil {
			return fmt.Errorf("failed to check organization membership: %w", err)
		}
		if !isMember {
			return fmt.Errorf("user is not a member of organization %d", *orgID)
		}
	}

	// Assign the role (note: current repository interface doesn't support org-scoped role assignment)
	// This would need to be enhanced in the repository layer to support organization context
	err = r.roleRepo.AssignToUser(ctx, roleID, userID)
	if err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	// Clear permission cache for this user
	r.clearUserPermissionCache(userID, orgID)

	r.logger.Info("Role assigned successfully",
		"user_id", userID,
		"role_id", roleID,
		"org_id", orgID)

	return nil
}

// RemoveRoleFromUser removes a role from a user
func (r *RBACService) RemoveRoleFromUser(ctx context.Context, userID, roleID uint, orgID *uint) error {
	r.logger.Info("Removing role from user",
		"user_id", userID,
		"role_id", roleID,
		"org_id", orgID)

	err := r.roleRepo.UnassignFromUser(ctx, roleID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove role: %w", err)
	}

	// Clear permission cache for this user
	r.clearUserPermissionCache(userID, orgID)

	r.logger.Info("Role removed successfully",
		"user_id", userID,
		"role_id", roleID,
		"org_id", orgID)

	return nil
}

// GetUserRoles returns all roles for a user
func (r *RBACService) GetUserRoles(ctx context.Context, userID uint, orgID *uint) ([]models.Role, error) {
	if orgID != nil {
		rolePointers, err := r.roleRepo.GetByUserAndOrganization(ctx, userID, *orgID, 0, 1000)
		if err != nil {
			return nil, err
		}

		// Convert from []*models.Role to []models.Role
		var roles []models.Role
		for _, rolePtr := range rolePointers {
			if rolePtr != nil {
				roles = append(roles, *rolePtr)
			}
		}
		return roles, nil
	}

	user, err := r.userRepo.GetWithRoles(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user with roles: %w", err)
	}

	// Filter roles by scope
	var roles []models.Role
	for _, role := range user.Roles {
		if role.IsGlobal() {
			roles = append(roles, role)
		}
	}

	return roles, nil
}

// ValidatePermission validates that a permission string is valid
func (r *RBACService) ValidatePermission(permission Permission) error {
	if !permission.IsValid() {
		return fmt.Errorf("invalid permission: %s", permission)
	}
	return nil
}

// Helper methods

func (r *RBACService) buildCacheKey(userID uint, orgID *uint) string {
	if orgID != nil {
		return fmt.Sprintf("user:%d:org:%d", userID, *orgID)
	}
	return fmt.Sprintf("user:%d:global", userID)
}

func (r *RBACService) roleMatchesContext(role models.Role, orgID *uint) bool {
	if role.IsGlobal() {
		return true // Global roles always match
	}

	if role.IsOrganizationScoped() {
		return orgID != nil // Organization roles only match when in org context
	}

	return true
}

func (r *RBACService) isUserOrganizationMember(ctx context.Context, userID, orgID uint) (bool, error) {
	// This could be optimized with a direct query, but for now use the existing method
	_, err := r.organizationRepo.GetByID(ctx, orgID)
	if err != nil {
		return false, err
	}

	// Check if user is member
	user, err := r.userRepo.GetWithOrganizations(ctx, userID)
	if err != nil {
		return false, err
	}

	for _, org := range user.Organizations {
		if org.ID == orgID {
			return true, nil
		}
	}

	return false, nil
}

func (r *RBACService) clearUserPermissionCache(userID uint, orgID *uint) {
	cacheKey := r.buildCacheKey(userID, orgID)
	delete(r.permissionCache, cacheKey)

	// Also clear global cache if we're clearing organization-specific cache
	if orgID != nil {
		globalKey := r.buildCacheKey(userID, nil)
		delete(r.permissionCache, globalKey)
	}
}

// ClearPermissionCache clears all cached permissions
func (r *RBACService) ClearPermissionCache() {
	r.permissionCache = make(map[string][]Permission)
	r.logger.Info("Permission cache cleared")
}
