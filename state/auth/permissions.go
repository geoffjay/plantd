package auth

import "strings"

// State service specific permissions as defined in the execution plan.
const (
	// Scope Management Permissions (Global)
	StateScopeCreate = "state:scope:create" // Create new service scopes
	StateScopeDelete = "state:scope:delete" // Delete entire service scopes
	StateScopeList   = "state:scope:list"   // List available scopes

	// Data Access Permissions (per-scope)
	StateDataRead   = "state:data:read"   // Read key-value pairs
	StateDataWrite  = "state:data:write"  // Set/update key-value pairs
	StateDataDelete = "state:data:delete" // Delete key-value pairs

	// Administrative Permissions
	StateAdminFull  = "state:admin:full"  // Full administrative access
	StateHealthRead = "state:health:read" // Health check endpoint access

	// System Operations
	StateMetricsRead = "state:metrics:read" // Access metrics
	StateSystemAdmin = "state:system:admin" // Full system administration

	// Service-Specific Scoped Permissions (templates)
	StateScopeReadTemplate   = "state:scope:%s:read"   // Read from specific scope
	StateScopeWriteTemplate  = "state:scope:%s:write"  // Write to specific scope
	StateScopeDeleteTemplate = "state:scope:%s:delete" // Delete from specific scope
	StateScopeAdminTemplate  = "state:scope:%s:admin"  // Admin specific scope
)

// Permission represents a user permission with scope context.
type Permission struct {
	Name  string `json:"name"`
	Scope string `json:"scope,omitempty"`
}

// PermissionChecker defines the interface for checking permissions.
type PermissionChecker interface {
	HasPermission(permission, scope string) bool
	HasAnyPermission(permissions []string, scope string) bool
	HasAllPermissions(permissions []string, scope string) bool
	GetPermissions() []Permission
}

// PermissionUtils provides utilities for permission validation and processing.
type PermissionUtils struct{}

// NewPermissionUtils creates a new instance of PermissionUtils.
func NewPermissionUtils() *PermissionUtils {
	return &PermissionUtils{}
}

// ValidatePermission validates that a permission string is properly formatted.
func (pu *PermissionUtils) ValidatePermission(permission string) error {
	if permission == "" {
		return &PermissionError{Message: "permission cannot be empty"}
	}

	parts := strings.Split(permission, ":")
	if len(parts) < 2 {
		return &PermissionError{Message: "permission must have at least service:action format"}
	}

	if parts[0] != "state" {
		return &PermissionError{Message: "permission must start with 'state:'"}
	}

	return nil
}

// IsWildcardPermission checks if a permission is a wildcard permission.
func (pu *PermissionUtils) IsWildcardPermission(permission string) bool {
	return strings.HasSuffix(permission, ":*") || permission == "state:*"
}

// IsGlobalPermission checks if a permission applies globally (not scoped).
func (pu *PermissionUtils) IsGlobalPermission(permission string) bool {
	globalPermissions := []string{
		StateScopeCreate,
		StateScopeDelete,
		StateScopeList,
		StateAdminFull,
		StateHealthRead,
		StateMetricsRead,
		StateSystemAdmin,
	}

	for _, globalPerm := range globalPermissions {
		if permission == globalPerm {
			return true
		}
	}

	return false
}

// IsScopedPermission checks if a permission is scope-specific.
func (pu *PermissionUtils) IsScopedPermission(permission string) bool {
	return strings.Contains(permission, ":scope:")
}

// ExtractScopeFromPermission extracts the scope from a scoped permission.
func (pu *PermissionUtils) ExtractScopeFromPermission(permission string) string {
	if !pu.IsScopedPermission(permission) {
		return ""
	}

	parts := strings.Split(permission, ":")
	for i, part := range parts {
		if part == "scope" && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	return ""
}

// CreateScopedPermission creates a scoped permission for a specific scope.
func (pu *PermissionUtils) CreateScopedPermission(basePermission, scope string) string {
	if scope == "" {
		return basePermission
	}

	switch basePermission {
	case StateDataRead:
		return strings.Replace(StateScopeReadTemplate, "%s", scope, 1)
	case StateDataWrite:
		return strings.Replace(StateScopeWriteTemplate, "%s", scope, 1)
	case StateDataDelete:
		return strings.Replace(StateScopeDeleteTemplate, "%s", scope, 1)
	case StateAdminFull:
		return strings.Replace(StateScopeAdminTemplate, "%s", scope, 1)
	default:
		return basePermission
	}
}

// PermissionInheritance handles permission inheritance rules.
type PermissionInheritance struct {
	utils *PermissionUtils
}

// NewPermissionInheritance creates a new permission inheritance handler.
func NewPermissionInheritance() *PermissionInheritance {
	return &PermissionInheritance{
		utils: NewPermissionUtils(),
	}
}

// GetImpliedPermissions returns permissions that are implied by a given permission.
func (pi *PermissionInheritance) GetImpliedPermissions(permission string) []string {
	var implied []string

	switch permission {
	case StateAdminFull:
		// Admin permission implies all other permissions
		implied = append(implied,
			StateScopeCreate, StateScopeDelete, StateScopeList,
			StateDataRead, StateDataWrite, StateDataDelete,
			StateHealthRead, StateMetricsRead,
		)
	case StateSystemAdmin:
		// System admin implies everything including admin
		implied = append(implied, StateAdminFull)
		implied = append(implied, pi.GetImpliedPermissions(StateAdminFull)...)
	case StateDataWrite:
		// Write permission implies read permission
		implied = append(implied, StateDataRead)
	case StateDataDelete:
		// Delete permission implies read permission
		implied = append(implied, StateDataRead)
	}

	// Handle scoped admin permissions
	if pi.utils.IsScopedPermission(permission) && strings.HasSuffix(permission, ":admin") {
		scope := pi.utils.ExtractScopeFromPermission(permission)
		if scope != "" {
			implied = append(implied,
				pi.utils.CreateScopedPermission(StateDataRead, scope),
				pi.utils.CreateScopedPermission(StateDataWrite, scope),
				pi.utils.CreateScopedPermission(StateDataDelete, scope),
			)
		}
	}

	return implied
}

// HasImpliedPermission checks if a user permission implies the required permission.
func (pi *PermissionInheritance) HasImpliedPermission(userPermission, requiredPermission string) bool {
	if userPermission == requiredPermission {
		return true
	}

	// Check wildcard permissions
	if pi.utils.IsWildcardPermission(userPermission) {
		return strings.HasPrefix(requiredPermission, strings.TrimSuffix(userPermission, "*"))
	}

	// Check implied permissions
	implied := pi.GetImpliedPermissions(userPermission)
	for _, impliedPerm := range implied {
		if impliedPerm == requiredPermission {
			return true
		}
	}

	return false
}

// StandardRoles defines common role templates for the state service.
var StandardRoles = []Role{
	{
		Name:        "state-developer",
		Description: "Developer access to state service",
		Permissions: []string{
			StateDataRead,
			StateDataWrite,
			StateHealthRead,
		},
		Scope: "organization",
	},
	{
		Name:        "state-admin",
		Description: "Administrative access to state service",
		Permissions: []string{
			StateScopeCreate,
			StateScopeDelete,
			StateScopeList,
			StateDataRead,
			StateDataWrite,
			StateDataDelete,
			StateHealthRead,
			StateMetricsRead,
		},
		Scope: "organization",
	},
	{
		Name:        "state-system-admin",
		Description: "Full system access to state service",
		Permissions: []string{
			StateSystemAdmin, // This implies all other permissions
		},
		Scope: "global",
	},
	{
		Name:        "state-readonly",
		Description: "Read-only access to state service",
		Permissions: []string{
			StateDataRead,
			StateHealthRead,
		},
		Scope: "organization",
	},
	{
		Name:        "state-service-owner",
		Description: "Service owner with full scope control",
		Permissions: []string{
			StateDataRead,
			StateDataWrite,
			StateDataDelete,
			StateHealthRead,
		},
		Scope: "service", // Special scope for service owners
	},
}

// Role represents a role with permissions.
type Role struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
	Scope       string   `json:"scope"`
}

// HasPermission checks if the role has a specific permission.
func (r *Role) HasPermission(permission string) bool {
	inheritance := NewPermissionInheritance()

	for _, rolePerm := range r.Permissions {
		if inheritance.HasImpliedPermission(rolePerm, permission) {
			return true
		}
	}

	return false
}

// GetEffectivePermissions returns all effective permissions including implied ones.
func (r *Role) GetEffectivePermissions() []string {
	inheritance := NewPermissionInheritance()
	permissionSet := make(map[string]bool)

	// Add direct permissions
	for _, perm := range r.Permissions {
		permissionSet[perm] = true

		// Add implied permissions
		implied := inheritance.GetImpliedPermissions(perm)
		for _, impliedPerm := range implied {
			permissionSet[impliedPerm] = true
		}
	}

	// Convert set to slice
	var effective []string
	for perm := range permissionSet {
		effective = append(effective, perm)
	}

	return effective
}

// PermissionError represents permission-related errors.
type PermissionError struct {
	Message string
	Code    string
}

// Error implements the error interface.
func (pe *PermissionError) Error() string {
	return pe.Message
}
