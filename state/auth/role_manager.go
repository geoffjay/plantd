package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/geoffjay/plantd/identity/pkg/client"
	log "github.com/sirupsen/logrus"
)

// RoleManager handles role creation, assignment, and management for state service.
type RoleManager struct {
	identityClient *client.Client
	logger         *log.Logger
}

// RoleManagerConfig holds configuration for the role manager.
type RoleManagerConfig struct {
	IdentityClient *client.Client
	Logger         *log.Logger
}

// NewRoleManager creates a new role manager instance.
func NewRoleManager(config *RoleManagerConfig) *RoleManager {
	if config.Logger == nil {
		config.Logger = log.New()
	}

	return &RoleManager{
		identityClient: config.IdentityClient,
		logger:         config.Logger,
	}
}

// RoleDefinition represents a role definition for creation.
type RoleDefinition struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Permissions  []string `json:"permissions"`
	Scope        string   `json:"scope"`
	Organization string   `json:"organization,omitempty"`
}

// SetupStandardRoles creates all standard roles for the state service.
func (rm *RoleManager) SetupStandardRoles(ctx context.Context) error {
	rm.logger.Info("Setting up standard roles for state service")

	for _, role := range StandardRoles {
		if err := rm.CreateRole(ctx, &RoleDefinition{
			Name:        role.Name,
			Description: role.Description,
			Permissions: role.Permissions,
			Scope:       role.Scope,
		}); err != nil {
			rm.logger.WithFields(log.Fields{
				"role_name": role.Name,
				"error":     err,
			}).Error("Failed to create standard role")
			return fmt.Errorf("failed to create role %s: %w", role.Name, err)
		}
	}

	rm.logger.Info("Successfully set up all standard roles")
	return nil
}

// CreateRole creates a new role in the identity service.
func (rm *RoleManager) CreateRole(ctx context.Context, roleDef *RoleDefinition) error {
	rm.logger.WithFields(log.Fields{
		"role_name":   roleDef.Name,
		"scope":       roleDef.Scope,
		"permissions": len(roleDef.Permissions),
	}).Info("Creating role")

	// Validate role definition
	if err := rm.ValidateRoleDefinition(roleDef); err != nil {
		return fmt.Errorf("invalid role definition: %w", err)
	}

	// Prepare role data for identity service
	roleData := map[string]interface{}{
		"name":        roleDef.Name,
		"description": roleDef.Description,
		"permissions": roleDef.Permissions,
		"scope":       roleDef.Scope,
	}

	if roleDef.Organization != "" {
		roleData["organization"] = roleDef.Organization
	}

	// Create role via identity service
	// This is a placeholder - the actual implementation would use the identity client
	rm.logger.WithFields(log.Fields{
		"role_data": roleData,
	}).Info("Role would be created via identity service")

	return nil
}

// AssignRoleToUser assigns a role to a user.
func (rm *RoleManager) AssignRoleToUser(ctx context.Context, userEmail, roleName string, orgID *uint) error {
	rm.logger.WithFields(log.Fields{
		"user_email": userEmail,
		"role_name":  roleName,
		"org_id":     orgID,
	}).Info("Assigning role to user")

	// This would integrate with the identity service
	// For now, this is a placeholder
	assignmentData := map[string]interface{}{
		"user_email": userEmail,
		"role_name":  roleName,
	}

	if orgID != nil {
		assignmentData["organization_id"] = *orgID
	}

	rm.logger.WithFields(log.Fields{
		"assignment_data": assignmentData,
	}).Info("Role assignment would be processed via identity service")

	return nil
}

// RemoveRoleFromUser removes a role from a user.
func (rm *RoleManager) RemoveRoleFromUser(ctx context.Context, userEmail, roleName string, orgID *uint) error {
	rm.logger.WithFields(log.Fields{
		"user_email": userEmail,
		"role_name":  roleName,
		"org_id":     orgID,
	}).Info("Removing role from user")

	// This would integrate with the identity service
	// For now, this is a placeholder
	removalData := map[string]interface{}{
		"user_email": userEmail,
		"role_name":  roleName,
	}

	if orgID != nil {
		removalData["organization_id"] = *orgID
	}

	rm.logger.WithFields(log.Fields{
		"removal_data": removalData,
	}).Info("Role removal would be processed via identity service")

	return nil
}

// ValidateRoleDefinition validates a role definition.
func (rm *RoleManager) ValidateRoleDefinition(roleDef *RoleDefinition) error {
	if roleDef.Name == "" {
		return fmt.Errorf("role name cannot be empty")
	}

	if !strings.HasPrefix(roleDef.Name, "state-") {
		return fmt.Errorf("state service roles must start with 'state-'")
	}

	if roleDef.Description == "" {
		return fmt.Errorf("role description cannot be empty")
	}

	if len(roleDef.Permissions) == 0 {
		return fmt.Errorf("role must have at least one permission")
	}

	// Validate scope
	validScopes := []string{"global", "organization", "service"}
	scopeValid := false
	for _, validScope := range validScopes {
		if roleDef.Scope == validScope {
			scopeValid = true
			break
		}
	}
	if !scopeValid {
		return fmt.Errorf("invalid scope: %s. Must be one of: %s", roleDef.Scope, strings.Join(validScopes, ", "))
	}

	// Validate permissions
	permUtils := NewPermissionUtils()
	for _, perm := range roleDef.Permissions {
		if err := permUtils.ValidatePermission(perm); err != nil {
			return fmt.Errorf("invalid permission %s: %w", perm, err)
		}
	}

	return nil
}

// GetRolePermissions returns the permissions for a role.
func (rm *RoleManager) GetRolePermissions(ctx context.Context, roleName string) ([]string, error) {
	rm.logger.WithFields(log.Fields{
		"role_name": roleName,
	}).Debug("Getting role permissions")

	// Find role in standard roles
	for _, role := range StandardRoles {
		if role.Name == roleName {
			return role.Permissions, nil
		}
	}

	// If not found in standard roles, query identity service
	// This is a placeholder for the actual implementation
	rm.logger.WithFields(log.Fields{
		"role_name": roleName,
	}).Info("Role permissions would be queried via identity service")

	return nil, fmt.Errorf("role not found: %s", roleName)
}

// ListRoles returns a list of all state service roles.
func (rm *RoleManager) ListRoles(ctx context.Context) ([]Role, error) {
	rm.logger.Debug("Listing state service roles")

	// Start with standard roles
	roles := make([]Role, len(StandardRoles))
	copy(roles, StandardRoles)

	// In a real implementation, this would also query the identity service
	// for any custom roles that have been created
	rm.logger.Info("Additional roles would be queried via identity service")

	return roles, nil
}

// GetUserRoles returns the roles assigned to a user.
func (rm *RoleManager) GetUserRoles(ctx context.Context, userEmail string, orgID *uint) ([]Role, error) {
	rm.logger.WithFields(log.Fields{
		"user_email": userEmail,
		"org_id":     orgID,
	}).Debug("Getting user roles")

	// This would integrate with the identity service
	// For now, return empty slice
	rm.logger.WithFields(log.Fields{
		"user_email": userEmail,
		"org_id":     orgID,
	}).Info("User roles would be queried via identity service")

	return []Role{}, nil
}

// MigrateExistingUsers applies default roles to existing users.
func (rm *RoleManager) MigrateExistingUsers(ctx context.Context) error {
	rm.logger.Info("Starting migration of existing users to role-based system")

	// This would involve:
	// 1. Query all existing users from identity service
	// 2. Analyze their current permissions
	// 3. Assign appropriate default roles
	// 4. Document the migration process

	rm.logger.Info("User migration would be processed via identity service integration")
	return nil
}

// CreateCustomRole creates a custom role for specific needs.
func (rm *RoleManager) CreateCustomRole(ctx context.Context, roleDef *RoleDefinition) error {
	rm.logger.WithFields(log.Fields{
		"role_name": roleDef.Name,
	}).Info("Creating custom role")

	// Validate that it's not conflicting with standard roles
	for _, standardRole := range StandardRoles {
		if standardRole.Name == roleDef.Name {
			return fmt.Errorf("role name conflicts with standard role: %s", roleDef.Name)
		}
	}

	return rm.CreateRole(ctx, roleDef)
}

// GetEffectiveRolePermissions returns all effective permissions for a role including inheritance.
func (rm *RoleManager) GetEffectiveRolePermissions(ctx context.Context, roleName string) ([]string, error) {
	rm.logger.WithFields(log.Fields{
		"role_name": roleName,
	}).Debug("Getting effective role permissions")

	// Find the role
	var targetRole *Role
	for _, role := range StandardRoles {
		if role.Name == roleName {
			targetRole = &role
			break
		}
	}

	if targetRole == nil {
		return nil, fmt.Errorf("role not found: %s", roleName)
	}

	// Get effective permissions including inheritance
	return targetRole.GetEffectivePermissions(), nil
}

// ValidateRoleAssignment validates that a role can be assigned to a user.
func (rm *RoleManager) ValidateRoleAssignment(ctx context.Context, userEmail, roleName string, orgID *uint) error {
	rm.logger.WithFields(log.Fields{
		"user_email": userEmail,
		"role_name":  roleName,
		"org_id":     orgID,
	}).Debug("Validating role assignment")

	// Find the role
	var targetRole *Role
	for _, role := range StandardRoles {
		if role.Name == roleName {
			targetRole = &role
			break
		}
	}

	if targetRole == nil {
		return fmt.Errorf("role not found: %s", roleName)
	}

	// Validate scope requirements
	switch targetRole.Scope {
	case "global":
		if orgID != nil {
			return fmt.Errorf("global role %s cannot be assigned within organization context", roleName)
		}
	case "organization":
		if orgID == nil {
			return fmt.Errorf("organization role %s requires organization context", roleName)
		}
	case "service":
		// Service roles can be assigned in any context
	}

	return nil
}

// RoleAssignmentRequest represents a role assignment request.
type RoleAssignmentRequest struct {
	UserEmail       string `json:"user_email"`
	RoleName        string `json:"role_name"`
	OrganizationID  *uint  `json:"organization_id,omitempty"`
	AssignedByEmail string `json:"assigned_by_email"`
	Reason          string `json:"reason,omitempty"`
	ExpiresAt       *int64 `json:"expires_at,omitempty"`
}

// ProcessRoleAssignmentRequest processes a role assignment request with validation.
func (rm *RoleManager) ProcessRoleAssignmentRequest(ctx context.Context, request *RoleAssignmentRequest) error {
	rm.logger.WithFields(log.Fields{
		"user_email":      request.UserEmail,
		"role_name":       request.RoleName,
		"organization_id": request.OrganizationID,
		"assigned_by":     request.AssignedByEmail,
		"reason":          request.Reason,
	}).Info("Processing role assignment request")

	// Validate the assignment
	if err := rm.ValidateRoleAssignment(ctx, request.UserEmail, request.RoleName, request.OrganizationID); err != nil {
		return fmt.Errorf("role assignment validation failed: %w", err)
	}

	// Check if the assigner has permission to assign this role
	// This would integrate with the identity service to verify permissions
	rm.logger.WithFields(log.Fields{
		"assigned_by": request.AssignedByEmail,
		"role_name":   request.RoleName,
	}).Info("Permission check for role assignment would be performed via identity service")

	// Process the assignment
	if err := rm.AssignRoleToUser(ctx, request.UserEmail, request.RoleName, request.OrganizationID); err != nil {
		return fmt.Errorf("role assignment failed: %w", err)
	}

	// Log the assignment for audit purposes
	rm.logger.WithFields(log.Fields{
		"user_email":      request.UserEmail,
		"role_name":       request.RoleName,
		"organization_id": request.OrganizationID,
		"assigned_by":     request.AssignedByEmail,
		"reason":          request.Reason,
	}).Info("Role assignment completed successfully")

	return nil
}

// GenerateRoleReport generates a comprehensive report of role assignments.
func (rm *RoleManager) GenerateRoleReport(ctx context.Context, orgID *uint) (map[string]interface{}, error) {
	rm.logger.WithFields(log.Fields{
		"org_id": orgID,
	}).Info("Generating role report")

	report := map[string]interface{}{
		"organization_id":  orgID,
		"standard_roles":   len(StandardRoles),
		"role_definitions": StandardRoles,
	}

	// In a real implementation, this would query the identity service for:
	// - Total number of users with state service roles
	// - Role distribution statistics
	// - Recent role assignments
	// - Permission usage patterns

	rm.logger.Info("Role report would be enhanced with identity service data")
	return report, nil
}
