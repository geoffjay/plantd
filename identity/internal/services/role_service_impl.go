package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
	log "github.com/sirupsen/logrus"

	"github.com/geoffjay/plantd/identity/internal/models"
	"github.com/geoffjay/plantd/identity/internal/repositories"
)

// roleServiceImpl implements the RoleService interface.
type roleServiceImpl struct {
	roleRepo  repositories.RoleRepository
	userRepo  repositories.UserRepository
	orgRepo   repositories.OrganizationRepository
	validator *validator.Validate
}

// NewRoleService creates a new RoleService implementation.
func NewRoleService(
	roleRepo repositories.RoleRepository,
	userRepo repositories.UserRepository,
	orgRepo repositories.OrganizationRepository,
) RoleService {
	return &roleServiceImpl{
		roleRepo:  roleRepo,
		userRepo:  userRepo,
		orgRepo:   orgRepo,
		validator: validator.New(),
	}
}

// CreateRole creates a new role with validation and business rules.
func (s *roleServiceImpl) CreateRole(ctx context.Context, req *CreateRoleRequest) (*models.Role, error) {
	logger := log.WithFields(log.Fields{
		"service": "role_service",
		"method":  "CreateRole",
		"name":    req.Name,
		"scope":   req.Scope,
	})

	// Validate request
	if err := s.validator.Struct(req); err != nil {
		logger.WithError(err).Error("validation failed")
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Validate permissions
	if err := s.ValidatePermissions(ctx, req.Permissions); err != nil {
		logger.WithError(err).Error("permission validation failed")
		return nil, err
	}

	// Validate name uniqueness within scope
	if err := s.ValidateRoleNameUniqueness(ctx, req.Name, req.Scope, nil); err != nil {
		logger.WithError(err).Error("name validation failed")
		return nil, err
	}

	// Convert permissions to JSON
	permissionsJSON, err := json.Marshal(req.Permissions)
	if err != nil {
		logger.WithError(err).Error("failed to marshal permissions")
		return nil, fmt.Errorf("failed to marshal permissions: %w", err)
	}

	// Create role model
	role := &models.Role{
		Name:        req.Name,
		Description: req.Description,
		Permissions: string(permissionsJSON),
		Scope:       req.Scope,
	}

	// Create role in database
	if err := s.roleRepo.Create(ctx, role); err != nil {
		logger.WithError(err).Error("failed to create role")
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	logger.WithField("role_id", role.ID).Info("role created successfully")
	return role, nil
}

// GetRoleByID retrieves a role by ID.
func (s *roleServiceImpl) GetRoleByID(ctx context.Context, id uint) (*models.Role, error) {
	logger := log.WithFields(log.Fields{
		"service": "role_service",
		"method":  "GetRoleByID",
		"role_id": id,
	})

	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		logger.WithError(err).Error("failed to get role")
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	if role == nil {
		logger.Debug("role not found")
		return nil, errors.New("role not found")
	}

	return role, nil
}

// GetRoleByName retrieves a role by name.
func (s *roleServiceImpl) GetRoleByName(ctx context.Context, name string) (*models.Role, error) {
	logger := log.WithFields(log.Fields{
		"service": "role_service",
		"method":  "GetRoleByName",
		"name":    name,
	})

	if name == "" {
		return nil, errors.New("name cannot be empty")
	}

	role, err := s.roleRepo.GetByName(ctx, name)
	if err != nil {
		logger.WithError(err).Error("failed to get role")
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	if role == nil {
		logger.Debug("role not found")
		return nil, errors.New("role not found")
	}

	return role, nil
}

// UpdateRole updates a role with validation and business rules.
func (s *roleServiceImpl) UpdateRole(ctx context.Context, id uint, req *UpdateRoleRequest) (*models.Role, error) {
	logger := createServiceLogger("role_service", "UpdateRole", log.Fields{"role_id": id})

	// Validate request
	if err := s.validator.Struct(req); err != nil {
		return nil, logAndError(logger, "validation failed", err)
	}

	// Get existing role
	role, err := s.getExistingRole(ctx, id, logger)
	if err != nil {
		return nil, err
	}

	// Update role fields with validation
	if err := s.updateRoleFields(ctx, id, req, role, logger); err != nil {
		return nil, err
	}

	// Save updated role
	if err := s.roleRepo.Update(ctx, role); err != nil {
		return nil, logAndError(logger, "failed to update role", err)
	}

	logSuccess(logger, "role updated successfully", nil)
	return role, nil
}

// getExistingRole retrieves and validates an existing role.
func (s *roleServiceImpl) getExistingRole(ctx context.Context, id uint, logger *log.Entry) (*models.Role, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		return nil, logAndError(logger, "failed to get role", err)
	}
	if role == nil {
		return nil, logAndErrorSimple(logger, "role "+ErrNotFound)
	}
	return role, nil
}

// updateRoleFields updates role fields with validation.
func (s *roleServiceImpl) updateRoleFields(ctx context.Context, id uint, req *UpdateRoleRequest, role *models.Role, logger *log.Entry) error {
	// Check name uniqueness if name is being updated
	if req.Name != nil && *req.Name != role.Name {
		scope := role.Scope
		if req.Scope != nil {
			scope = *req.Scope
		}
		if err := s.ValidateRoleNameUniqueness(ctx, *req.Name, scope, &id); err != nil {
			return logAndError(logger, "name validation failed", err)
		}
		role.Name = *req.Name
	}

	// Update other fields
	if req.Description != nil {
		role.Description = *req.Description
	}
	if req.Scope != nil {
		role.Scope = *req.Scope
	}

	// Update permissions if provided
	if req.Permissions != nil {
		if err := s.updateRolePermissions(ctx, req.Permissions, role, logger); err != nil {
			return err
		}
	}

	return nil
}

// updateRolePermissions updates role permissions with validation.
func (s *roleServiceImpl) updateRolePermissions(ctx context.Context, permissions []string, role *models.Role, logger *log.Entry) error {
	if err := s.ValidatePermissions(ctx, permissions); err != nil {
		return logAndError(logger, "permission validation failed", err)
	}
	permissionsJSON, err := json.Marshal(permissions)
	if err != nil {
		return logAndError(logger, "failed to marshal permissions", err)
	}
	role.Permissions = string(permissionsJSON)
	return nil
}

// DeleteRole soft deletes a role.
func (s *roleServiceImpl) DeleteRole(ctx context.Context, id uint) error {
	logger := log.WithFields(log.Fields{
		"service": "role_service",
		"method":  "DeleteRole",
		"role_id": id,
	})

	// Check if role exists
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		logger.WithError(err).Error("failed to get role")
		return fmt.Errorf("failed to get role: %w", err)
	}
	if role == nil {
		logger.Error("role not found")
		return errors.New("role not found")
	}

	// TODO: Check if role is assigned to users/organizations before deletion

	// Delete role
	if err := s.roleRepo.Delete(ctx, id); err != nil {
		logger.WithError(err).Error("failed to delete role")
		return fmt.Errorf("failed to delete role: %w", err)
	}

	logger.Info("role deleted successfully")
	return nil
}

// ListRoles lists roles with pagination and filtering.
func (s *roleServiceImpl) ListRoles(ctx context.Context, req *ListRolesRequest) ([]*models.Role, error) {
	logger := log.WithFields(log.Fields{
		"service": "role_service",
		"method":  "ListRoles",
		"offset":  req.Offset,
		"limit":   req.Limit,
	})

	// Validate request
	if err := s.validator.Struct(req); err != nil {
		logger.WithError(err).Error("validation failed")
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// For now, use basic list without complex filtering
	roles, err := s.roleRepo.List(ctx, req.Offset, req.Limit)
	if err != nil {
		logger.WithError(err).Error("failed to list roles")
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}

	logger.WithField("count", len(roles)).Debug("roles listed successfully")
	return roles, nil
}

// SearchRoles searches roles by query and fields.
func (s *roleServiceImpl) SearchRoles(ctx context.Context, req *SearchRolesRequest) ([]*models.Role, error) {
	logger := log.WithFields(log.Fields{
		"service": "role_service",
		"method":  "SearchRoles",
		"query":   req.Query,
		"fields":  req.SearchFields,
	})

	// Validate request
	if err := s.validator.Struct(req); err != nil {
		logger.WithError(err).Error("validation failed")
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Simple implementation using repository search methods
	var roles []*models.Role
	var err error

	// Search by specified fields
	for _, field := range req.SearchFields {
		switch field {
		case FieldName:
			roles, err = s.roleRepo.SearchByName(ctx, req.Query, req.Offset, req.Limit)
		case FieldDescription:
			// TODO: Implement SearchByDescription in repository
			logger.Warn("description search not implemented yet")
		}
		if err != nil {
			logger.WithError(err).Error("search failed")
			return nil, fmt.Errorf("search failed: %w", err)
		}
		if len(roles) > 0 {
			break // Return first successful search
		}
	}

	logger.WithField("count", len(roles)).Debug("roles searched successfully")
	return roles, nil
}

// CountRoles returns the total number of roles.
func (s *roleServiceImpl) CountRoles(ctx context.Context) (int64, error) {
	logger := log.WithFields(log.Fields{
		"service": "role_service",
		"method":  "CountRoles",
	})

	count, err := s.roleRepo.Count(ctx)
	if err != nil {
		logger.WithError(err).Error("failed to count roles")
		return 0, fmt.Errorf("failed to count roles: %w", err)
	}

	logger.WithField("count", count).Debug("roles counted successfully")
	return count, nil
}

// GetRolePermissions returns the permissions for a role.
func (s *roleServiceImpl) GetRolePermissions(ctx context.Context, roleID uint) ([]string, error) {
	logger := log.WithFields(log.Fields{
		"service": "role_service",
		"method":  "GetRolePermissions",
		"role_id": roleID,
	})

	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		logger.WithError(err).Error("failed to get role")
		return nil, fmt.Errorf("failed to get role: %w", err)
	}
	if role == nil {
		logger.Error("role not found")
		return nil, errors.New("role not found")
	}

	permissions, err := role.GetPermissions()
	if err != nil {
		logger.WithError(err).Error("failed to parse permissions")
		return nil, fmt.Errorf("failed to parse permissions: %w", err)
	}

	logger.WithField("permissions", permissions).Debug("permissions retrieved successfully")
	return permissions, nil
}

// HasPermission checks if a role has a specific permission.
func (s *roleServiceImpl) HasPermission(ctx context.Context, roleID uint, permission string) (bool, error) {
	logger := log.WithFields(log.Fields{
		"service":    "role_service",
		"method":     "HasPermission",
		"role_id":    roleID,
		"permission": permission,
	})

	role, err := s.roleRepo.GetByID(ctx, roleID)
	if err != nil {
		logger.WithError(err).Error("failed to get role")
		return false, fmt.Errorf("failed to get role: %w", err)
	}
	if role == nil {
		logger.Error("role not found")
		return false, errors.New("role not found")
	}

	hasPermission := role.HasPermission(permission)
	logger.WithField("has_permission", hasPermission).Debug("permission check completed")
	return hasPermission, nil
}

// ValidatePermissions validates a list of permissions.
func (s *roleServiceImpl) ValidatePermissions(_ context.Context, permissions []string) error {
	logger := log.WithFields(log.Fields{
		"service":     "role_service",
		"method":      "ValidatePermissions",
		"permissions": permissions,
	})

	// Basic validation - ensure no empty permissions
	for _, perm := range permissions {
		if perm == "" {
			logger.Error("empty permission found")
			return errors.New("permissions cannot be empty")
		}
	}

	// TODO: Add more sophisticated permission validation (e.g., against allowed permissions list)

	logger.Debug("permissions validated successfully")
	return nil
}

// ValidateRoleNameUniqueness validates that a role name is unique within a scope.
func (s *roleServiceImpl) ValidateRoleNameUniqueness(ctx context.Context, name string, scope models.RoleScope, excludeID *uint) error {
	logger := log.WithFields(log.Fields{
		"service": "role_service",
		"method":  "ValidateRoleNameUniqueness",
		"name":    name,
		"scope":   scope,
	})

	// Use SearchByName to find roles with similar names
	existing, err := s.roleRepo.SearchByName(ctx, name, 0, 10)
	if err != nil {
		logger.WithError(err).Error("failed to check name uniqueness")
		return fmt.Errorf("failed to check name uniqueness: %w", err)
	}

	// Check if we found an exact match within the same scope
	for _, role := range existing {
		if role.Name == name && role.Scope == scope && (excludeID == nil || role.ID != *excludeID) {
			logger.Error("role name already exists in scope")
			return errors.New("role name already exists in scope")
		}
	}

	return nil
}

// Placeholder implementations for relationship management
// These will be fully implemented when the relationship methods are added to repositories

// AddPermissionToRole adds a permission to a role.
func (s *roleServiceImpl) AddPermissionToRole(_ context.Context, roleID uint, permission string) error { //nolint:revive
	// TODO: Implement using role.AddPermission() method
	return errors.New("not implemented yet")
}

// RemovePermissionFromRole removes a permission from a role.
func (s *roleServiceImpl) RemovePermissionFromRole(_ context.Context, roleID uint, permission string) error { //nolint:revive
	// TODO: Implement using role.RemovePermission() method
	return errors.New("not implemented yet")
}

// AssignRoleToUser assigns a role to a user.
func (s *roleServiceImpl) AssignRoleToUser(_ context.Context, roleID, userID uint) error { //nolint:revive
	// TODO: Implement when role assignment methods are added to repositories
	return errors.New("not implemented yet")
}

// UnassignRoleFromUser removes a role from a user.
func (s *roleServiceImpl) UnassignRoleFromUser(_ context.Context, roleID, userID uint) error { //nolint:revive
	// TODO: Implement when role assignment methods are added to repositories
	return errors.New("not implemented yet")
}

// GetUsersWithRole returns users that have a specific role.
func (s *roleServiceImpl) GetUsersWithRole(_ context.Context, roleID uint, offset, limit int) ([]*models.User, error) { //nolint:revive
	// TODO: Implement when role assignment methods are added to repositories
	return nil, errors.New("not implemented yet")
}

// AssignRoleToOrganization assigns a role to an organization.
func (s *roleServiceImpl) AssignRoleToOrganization(_ context.Context, _, _ uint) error { //nolint:revive
	// TODO: Implement when role assignment methods are added to repositories
	return errors.New("not implemented yet")
}

// UnassignRoleFromOrganization removes a role from an organization.
func (s *roleServiceImpl) UnassignRoleFromOrganization(_ context.Context, _, _ uint) error { //nolint:revive
	// TODO: Implement when role assignment methods are added to repositories
	return errors.New("not implemented yet")
}

// GetOrganizationsWithRole returns organizations that have a specific role.
func (s *roleServiceImpl) GetOrganizationsWithRole(_ context.Context, _ uint, _, _ int) ([]*models.Organization, error) { //nolint:revive
	// TODO: Implement when role assignment methods are added to repositories
	return nil, errors.New("not implemented yet")
}

// GetRolesByScope returns roles filtered by scope.
func (s *roleServiceImpl) GetRolesByScope(_ context.Context, _ models.RoleScope, _, _ int) ([]*models.Role, error) { //nolint:revive
	// TODO: Implement when scope filtering methods are added to repositories
	return nil, errors.New("not implemented yet")
}

// GetRolesWithPermission returns roles that have a specific permission.
func (s *roleServiceImpl) GetRolesWithPermission(_ context.Context, _ string, _, _ int) ([]*models.Role, error) { //nolint:revive
	// TODO: Implement when permission filtering methods are added to repositories
	return nil, errors.New("not implemented yet")
}

// GetGlobalRoles returns global scope roles.
func (s *roleServiceImpl) GetGlobalRoles(ctx context.Context, offset, limit int) ([]*models.Role, error) {
	return s.GetRolesByScope(ctx, models.RoleScopeGlobal, offset, limit)
}

// GetOrganizationRoles returns organization scope roles.
func (s *roleServiceImpl) GetOrganizationRoles(ctx context.Context, offset, limit int) ([]*models.Role, error) {
	return s.GetRolesByScope(ctx, models.RoleScopeOrganization, offset, limit)
}
