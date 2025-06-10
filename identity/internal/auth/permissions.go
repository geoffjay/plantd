package auth

import (
	"context"
	"fmt"
	"strings"
)

// PermissionCategory represents different categories of permissions.
type PermissionCategory string

const (
	// CategoryUser permissions for user management.
	CategoryUser PermissionCategory = "user"
	// CategoryOrganization permissions for organization management.
	CategoryOrganization PermissionCategory = "organization"
	// CategoryRole permissions for role management.
	CategoryRole PermissionCategory = "role"
	// CategoryAuth permissions for authentication operations.
	CategoryAuth PermissionCategory = "auth"
	// CategorySystem permissions for system-level operations.
	CategorySystem PermissionCategory = "system"
)

// Permission represents a specific permission in the system.
type Permission string

// User Management Permissions.
const (
	// User read permissions
	PermissionUserRead   Permission = "user:read"
	PermissionUserList   Permission = "user:list"
	PermissionUserSearch Permission = "user:search"

	// User write permissions
	PermissionUserCreate Permission = "user:create"
	PermissionUserUpdate Permission = "user:update"
	PermissionUserDelete Permission = "user:delete"

	// User activation permissions
	PermissionUserActivate   Permission = "user:activate"
	PermissionUserDeactivate Permission = "user:deactivate"

	// User profile permissions
	PermissionUserProfile       Permission = "user:profile"
	PermissionUserProfileUpdate Permission = "user:profile:update"
	PermissionUserPasswordReset Permission = "user:password:reset"
)

// Organization Management Permissions
const (
	// Organization read permissions
	PermissionOrganizationRead   Permission = "organization:read"
	PermissionOrganizationList   Permission = "organization:list"
	PermissionOrganizationSearch Permission = "organization:search"

	// Organization write permissions
	PermissionOrganizationCreate Permission = "organization:create"
	PermissionOrganizationUpdate Permission = "organization:update"
	PermissionOrganizationDelete Permission = "organization:delete"

	// Organization membership permissions
	PermissionOrganizationMemberAdd    Permission = "organization:member:add"
	PermissionOrganizationMemberRemove Permission = "organization:member:remove"
	PermissionOrganizationMemberList   Permission = "organization:member:list"

	// Organization admin permissions
	PermissionOrganizationAdmin Permission = "organization:admin"
)

// Role Management Permissions
const (
	// Role read permissions
	PermissionRoleRead   Permission = "role:read"
	PermissionRoleList   Permission = "role:list"
	PermissionRoleSearch Permission = "role:search"

	// Role write permissions
	PermissionRoleCreate Permission = "role:create"
	PermissionRoleUpdate Permission = "role:update"
	PermissionRoleDelete Permission = "role:delete"

	// Role assignment permissions
	PermissionRoleAssign   Permission = "role:assign"
	PermissionRoleUnassign Permission = "role:unassign"

	// Role permission management
	PermissionRolePermissionAdd    Permission = "role:permission:add"
	PermissionRolePermissionRemove Permission = "role:permission:remove"
)

// Authentication Permissions
const (
	// Authentication operations
	PermissionAuthLogin   Permission = "auth:login"
	PermissionAuthLogout  Permission = "auth:logout"
	PermissionAuthRefresh Permission = "auth:refresh"

	// Token management
	PermissionAuthTokenRevoke Permission = "auth:token:revoke"
	PermissionAuthTokenList   Permission = "auth:token:list"

	// Session management
	PermissionAuthSessionList   Permission = "auth:session:list"
	PermissionAuthSessionRevoke Permission = "auth:session:revoke"
)

// System Permissions
const (
	// System administration
	PermissionSystemAdmin Permission = "system:admin"
	PermissionSystemRead  Permission = "system:read"

	// Health and monitoring
	PermissionSystemHealth  Permission = "system:health"
	PermissionSystemMetrics Permission = "system:metrics"

	// Configuration management
	PermissionSystemConfig Permission = "system:config"
)

// AllPermissions returns all available permissions in the system
func AllPermissions() []Permission {
	return []Permission{
		// User permissions
		PermissionUserRead, PermissionUserList, PermissionUserSearch,
		PermissionUserCreate, PermissionUserUpdate, PermissionUserDelete,
		PermissionUserActivate, PermissionUserDeactivate,
		PermissionUserProfile, PermissionUserProfileUpdate, PermissionUserPasswordReset,

		// Organization permissions
		PermissionOrganizationRead, PermissionOrganizationList, PermissionOrganizationSearch,
		PermissionOrganizationCreate, PermissionOrganizationUpdate, PermissionOrganizationDelete,
		PermissionOrganizationMemberAdd, PermissionOrganizationMemberRemove, PermissionOrganizationMemberList,
		PermissionOrganizationAdmin,

		// Role permissions
		PermissionRoleRead, PermissionRoleList, PermissionRoleSearch,
		PermissionRoleCreate, PermissionRoleUpdate, PermissionRoleDelete,
		PermissionRoleAssign, PermissionRoleUnassign,
		PermissionRolePermissionAdd, PermissionRolePermissionRemove,

		// Auth permissions
		PermissionAuthLogin, PermissionAuthLogout, PermissionAuthRefresh,
		PermissionAuthTokenRevoke, PermissionAuthTokenList,
		PermissionAuthSessionList, PermissionAuthSessionRevoke,

		// System permissions
		PermissionSystemAdmin, PermissionSystemRead,
		PermissionSystemHealth, PermissionSystemMetrics,
		PermissionSystemConfig,
	}
}

// PermissionsByCategory returns permissions grouped by category
func PermissionsByCategory() map[PermissionCategory][]Permission {
	return map[PermissionCategory][]Permission{
		CategoryUser: {
			PermissionUserRead, PermissionUserList, PermissionUserSearch,
			PermissionUserCreate, PermissionUserUpdate, PermissionUserDelete,
			PermissionUserActivate, PermissionUserDeactivate,
			PermissionUserProfile, PermissionUserProfileUpdate, PermissionUserPasswordReset,
		},
		CategoryOrganization: {
			PermissionOrganizationRead, PermissionOrganizationList, PermissionOrganizationSearch,
			PermissionOrganizationCreate, PermissionOrganizationUpdate, PermissionOrganizationDelete,
			PermissionOrganizationMemberAdd, PermissionOrganizationMemberRemove, PermissionOrganizationMemberList,
			PermissionOrganizationAdmin,
		},
		CategoryRole: {
			PermissionRoleRead, PermissionRoleList, PermissionRoleSearch,
			PermissionRoleCreate, PermissionRoleUpdate, PermissionRoleDelete,
			PermissionRoleAssign, PermissionRoleUnassign,
			PermissionRolePermissionAdd, PermissionRolePermissionRemove,
		},
		CategoryAuth: {
			PermissionAuthLogin, PermissionAuthLogout, PermissionAuthRefresh,
			PermissionAuthTokenRevoke, PermissionAuthTokenList,
			PermissionAuthSessionList, PermissionAuthSessionRevoke,
		},
		CategorySystem: {
			PermissionSystemAdmin, PermissionSystemRead,
			PermissionSystemHealth, PermissionSystemMetrics,
			PermissionSystemConfig,
		},
	}
}

// IsValid checks if a permission string is valid
func (p Permission) IsValid() bool {
	allPermissions := AllPermissions()
	for _, perm := range allPermissions {
		if perm == p {
			return true
		}
	}
	return false
}

// Category returns the category of the permission
func (p Permission) Category() PermissionCategory {
	permStr := string(p)
	if strings.Contains(permStr, ":") {
		parts := strings.SplitN(permStr, ":", 2)
		return PermissionCategory(parts[0])
	}
	return ""
}

// String returns the string representation of the permission
func (p Permission) String() string {
	return string(p)
}

// AuthorizationContext represents the context for authorization decisions
type AuthorizationContext struct {
	UserID         uint
	OrganizationID *uint
	Resource       string
	Action         string
	RequestContext context.Context
}

// PermissionChecker defines interface for checking permissions
type PermissionChecker interface {
	// HasPermission checks if a user has a specific permission
	HasPermission(ctx context.Context, userID uint, permission Permission, orgID *uint) (bool, error)

	// HasAnyPermission checks if a user has any of the specified permissions
	HasAnyPermission(ctx context.Context, userID uint, permissions []Permission, orgID *uint) (bool, error)

	// HasAllPermissions checks if a user has all of the specified permissions
	HasAllPermissions(ctx context.Context, userID uint, permissions []Permission, orgID *uint) (bool, error)

	// GetUserPermissions returns all permissions for a user in a given context
	GetUserPermissions(ctx context.Context, userID uint, orgID *uint) ([]Permission, error)

	// CanAccessResource checks if a user can access a specific resource
	CanAccessResource(ctx context.Context, authCtx *AuthorizationContext) (bool, error)
}

// PermissionError represents permission-related errors
type PermissionError struct {
	UserID     uint
	Permission Permission
	OrgID      *uint
	Message    string
}

func (e *PermissionError) Error() string {
	if e.OrgID != nil {
		return fmt.Sprintf("user %d lacks permission '%s' in organization %d: %s",
			e.UserID, e.Permission, *e.OrgID, e.Message)
	}
	return fmt.Sprintf("user %d lacks permission '%s': %s",
		e.UserID, e.Permission, e.Message)
}

// UnauthorizedError represents an unauthorized access error
type UnauthorizedError struct {
	UserID   uint
	Resource string
	Action   string
	Message  string
}

func (e *UnauthorizedError) Error() string {
	return fmt.Sprintf("user %d unauthorized to %s resource '%s': %s",
		e.UserID, e.Action, e.Resource, e.Message)
}

// RequiredPermissions defines permissions required for common operations
var RequiredPermissions = map[string][]Permission{
	// User operations
	"user.create":     {PermissionUserCreate},
	"user.read":       {PermissionUserRead},
	"user.update":     {PermissionUserUpdate},
	"user.delete":     {PermissionUserDelete},
	"user.list":       {PermissionUserList},
	"user.activate":   {PermissionUserActivate},
	"user.deactivate": {PermissionUserDeactivate},

	// Organization operations
	"organization.create":        {PermissionOrganizationCreate},
	"organization.read":          {PermissionOrganizationRead},
	"organization.update":        {PermissionOrganizationUpdate},
	"organization.delete":        {PermissionOrganizationDelete},
	"organization.list":          {PermissionOrganizationList},
	"organization.member.add":    {PermissionOrganizationMemberAdd},
	"organization.member.remove": {PermissionOrganizationMemberRemove},

	// Role operations
	"role.create": {PermissionRoleCreate},
	"role.read":   {PermissionRoleRead},
	"role.update": {PermissionRoleUpdate},
	"role.delete": {PermissionRoleDelete},
	"role.assign": {PermissionRoleAssign},

	// Admin operations
	"admin.all": {PermissionSystemAdmin},
}
