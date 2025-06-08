package repositories

import (
	"context"

	"github.com/geoffjay/plantd/identity/internal/models"
)

// OrganizationRepository defines the interface for organization data access operations.
type OrganizationRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, organization *models.Organization) error
	GetByID(ctx context.Context, id uint) (*models.Organization, error)
	GetBySlug(ctx context.Context, slug string) (*models.Organization, error)
	Update(ctx context.Context, organization *models.Organization) error
	Delete(ctx context.Context, id uint) error

	// List operations with pagination
	List(ctx context.Context, offset, limit int) ([]*models.Organization, error)
	Count(ctx context.Context) (int64, error)

	// Advanced query operations
	GetActiveOrganizations(ctx context.Context, offset, limit int) ([]*models.Organization, error)
	GetByUser(ctx context.Context, userID uint, offset, limit int) ([]*models.Organization, error)

	// Search operations
	SearchByName(ctx context.Context, namePattern string, offset, limit int) ([]*models.Organization, error)
	SearchBySlug(ctx context.Context, slugPattern string, offset, limit int) ([]*models.Organization, error)

	// Relationship operations
	AddUser(ctx context.Context, organizationID, userID uint) error
	RemoveUser(ctx context.Context, organizationID, userID uint) error
	AddRole(ctx context.Context, organizationID, roleID uint) error
	RemoveRole(ctx context.Context, organizationID, roleID uint) error

	// Member operations
	GetMembers(ctx context.Context, organizationID uint, offset, limit int) ([]*models.User, error)
	GetMembersWithRoles(ctx context.Context, organizationID uint, offset, limit int) ([]*models.User, error)
	CountMembers(ctx context.Context, organizationID uint) (int64, error)

	// Preload operations
	GetWithUsers(ctx context.Context, id uint) (*models.Organization, error)
	GetWithRoles(ctx context.Context, id uint) (*models.Organization, error)
	GetWithAll(ctx context.Context, id uint) (*models.Organization, error)
}
