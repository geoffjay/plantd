package repositories

import (
	"context"

	"github.com/geoffjay/plantd/identity/internal/models"
)

// UserRepository defines the interface for user data access operations.
type UserRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uint) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id uint) error

	// List operations with pagination
	List(ctx context.Context, offset, limit int) ([]*models.User, error)
	Count(ctx context.Context) (int64, error)

	// Advanced query operations
	GetByOrganization(ctx context.Context, organizationID uint, offset, limit int) ([]*models.User, error)
	GetByRole(ctx context.Context, roleID uint, offset, limit int) ([]*models.User, error)
	GetActiveUsers(ctx context.Context, offset, limit int) ([]*models.User, error)
	GetVerifiedUsers(ctx context.Context, offset, limit int) ([]*models.User, error)

	// Search operations
	SearchByEmail(ctx context.Context, emailPattern string, offset, limit int) ([]*models.User, error)
	SearchByUsername(ctx context.Context, usernamePattern string, offset, limit int) ([]*models.User, error)
	SearchByName(ctx context.Context, namePattern string, offset, limit int) ([]*models.User, error)

	// Relationship operations
	AddToOrganization(ctx context.Context, userID, organizationID uint) error
	RemoveFromOrganization(ctx context.Context, userID, organizationID uint) error
	AssignRole(ctx context.Context, userID, roleID uint) error
	UnassignRole(ctx context.Context, userID, roleID uint) error

	// Preload operations
	GetWithRoles(ctx context.Context, id uint) (*models.User, error)
	GetWithOrganizations(ctx context.Context, id uint) (*models.User, error)
	GetWithAll(ctx context.Context, id uint) (*models.User, error)
}
