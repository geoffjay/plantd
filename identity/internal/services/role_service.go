package services

import (
	"context"

	"github.com/geoffjay/plantd/identity/internal/models"
)

// RoleService defines the interface for role business logic operations.
type RoleService interface {
	// Role CRUD operations with business rules
	CreateRole(ctx context.Context, req *CreateRoleRequest) (*models.Role, error)
	GetRoleByID(ctx context.Context, id uint) (*models.Role, error)
	GetRoleByName(ctx context.Context, name string) (*models.Role, error)
	UpdateRole(ctx context.Context, id uint, req *UpdateRoleRequest) (*models.Role, error)
	DeleteRole(ctx context.Context, id uint) error

	// Role listing and searching
	ListRoles(ctx context.Context, req *ListRolesRequest) ([]*models.Role, error)
	SearchRoles(ctx context.Context, req *SearchRolesRequest) ([]*models.Role, error)
	CountRoles(ctx context.Context) (int64, error)

	// Permission management
	AddPermissionToRole(ctx context.Context, roleID uint, permission string) error
	RemovePermissionFromRole(ctx context.Context, roleID uint, permission string) error
	GetRolePermissions(ctx context.Context, roleID uint) ([]string, error)
	HasPermission(ctx context.Context, roleID uint, permission string) (bool, error)

	// Role assignment to users and organizations
	AssignRoleToUser(ctx context.Context, roleID, userID uint) error
	UnassignRoleFromUser(ctx context.Context, roleID, userID uint) error
	GetUsersWithRole(ctx context.Context, roleID uint, offset, limit int) ([]*models.User, error)

	AssignRoleToOrganization(ctx context.Context, roleID, orgID uint) error
	UnassignRoleFromOrganization(ctx context.Context, roleID, orgID uint) error
	GetOrganizationsWithRole(ctx context.Context, roleID uint, offset, limit int) ([]*models.Organization, error)

	// Role querying by scope and permissions
	GetRolesByScope(ctx context.Context, scope models.RoleScope, offset, limit int) ([]*models.Role, error)
	GetRolesWithPermission(ctx context.Context, permission string, offset, limit int) ([]*models.Role, error)
	GetGlobalRoles(ctx context.Context, offset, limit int) ([]*models.Role, error)
	GetOrganizationRoles(ctx context.Context, offset, limit int) ([]*models.Role, error)

	// Role validation
	ValidateRoleNameUniqueness(ctx context.Context, name string, scope models.RoleScope, excludeID *uint) error
	ValidatePermissions(ctx context.Context, permissions []string) error
}

// CreateRoleRequest represents the request to create a new role.
type CreateRoleRequest struct {
	Name        string           `json:"name" validate:"required,min=1,max=100"`
	Description string           `json:"description" validate:"max=500"`
	Permissions []string         `json:"permissions" validate:"required,min=1"`
	Scope       models.RoleScope `json:"scope" validate:"required,oneof=global organization"`
}

// UpdateRoleRequest represents the request to update a role.
type UpdateRoleRequest struct {
	Name        *string           `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description *string           `json:"description,omitempty" validate:"omitempty,max=500"`
	Permissions []string          `json:"permissions,omitempty" validate:"omitempty,min=1"`
	Scope       *models.RoleScope `json:"scope,omitempty" validate:"omitempty,oneof=global organization"`
}

// ListRolesRequest represents the request to list roles with pagination and filtering.
type ListRolesRequest struct {
	Offset    int               `json:"offset" validate:"min=0"`
	Limit     int               `json:"limit" validate:"min=1,max=100"`
	Scope     *models.RoleScope `json:"scope,omitempty" validate:"omitempty,oneof=global organization"`
	SortBy    string            `json:"sort_by" validate:"omitempty,oneof=id name scope created_at updated_at"`
	SortOrder string            `json:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// SearchRolesRequest represents the request to search roles.
type SearchRolesRequest struct {
	Query        string            `json:"query" validate:"required,min=1"`
	SearchFields []string          `json:"search_fields" validate:"required"`
	Scope        *models.RoleScope `json:"scope,omitempty" validate:"omitempty,oneof=global organization"`
	Permissions  []string          `json:"permissions,omitempty"`
	Offset       int               `json:"offset" validate:"min=0"`
	Limit        int               `json:"limit" validate:"min=1,max=100"`
}
