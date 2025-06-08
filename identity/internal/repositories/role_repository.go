package repositories

import (
	"context"

	"github.com/geoffjay/plantd/identity/internal/models"
)

// RoleRepository defines the interface for role data access operations.
type RoleRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, role *models.Role) error
	GetByID(ctx context.Context, id uint) (*models.Role, error)
	GetByName(ctx context.Context, name string) (*models.Role, error)
	Update(ctx context.Context, role *models.Role) error
	Delete(ctx context.Context, id uint) error

	// List operations with pagination
	List(ctx context.Context, offset, limit int) ([]*models.Role, error)
	Count(ctx context.Context) (int64, error)

	// Advanced query operations
	GetByScope(ctx context.Context, scope models.RoleScope, offset, limit int) ([]*models.Role, error)
	GetGlobalRoles(ctx context.Context, offset, limit int) ([]*models.Role, error)
	GetOrganizationRoles(ctx context.Context, offset, limit int) ([]*models.Role, error)
	GetByUser(ctx context.Context, userID uint, offset, limit int) ([]*models.Role, error)
	GetByUserAndOrganization(ctx context.Context, userID, organizationID uint, offset, limit int) ([]*models.Role, error)
	GetByOrganization(ctx context.Context, organizationID uint, offset, limit int) ([]*models.Role, error)

	// Search operations
	SearchByName(ctx context.Context, namePattern string, offset, limit int) ([]*models.Role, error)
	SearchByDescription(ctx context.Context, descriptionPattern string, offset, limit int) ([]*models.Role, error)

	// Permission operations
	GetRolesWithPermission(ctx context.Context, permission string, offset, limit int) ([]*models.Role, error)

	// Relationship operations
	AssignToUser(ctx context.Context, roleID, userID uint) error
	UnassignFromUser(ctx context.Context, roleID, userID uint) error
	AssignToOrganization(ctx context.Context, roleID, organizationID uint) error
	UnassignFromOrganization(ctx context.Context, roleID, organizationID uint) error

	// Preload operations
	GetWithUsers(ctx context.Context, id uint) (*models.Role, error)
	GetWithOrganizations(ctx context.Context, id uint) (*models.Role, error)
	GetWithAll(ctx context.Context, id uint) (*models.Role, error)
}
