package repositories

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/geoffjay/plantd/identity/internal/models"
)

// userRepositoryGorm implements UserRepository using GORM.
type userRepositoryGorm struct {
	db *gorm.DB
}

// NewUserRepository creates a new UserRepository implementation using GORM.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepositoryGorm{db: db}
}

// Create creates a new user.
func (r *userRepositoryGorm) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// GetByID retrieves a user by ID.
func (r *userRepositoryGorm) GetByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetByEmail retrieves a user by email.
func (r *userRepositoryGorm) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetByUsername retrieves a user by username.
func (r *userRepositoryGorm) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// Update updates a user.
func (r *userRepositoryGorm) Update(ctx context.Context, user *models.User) error {
	// Check if user exists first
	var existingUser models.User
	err := r.db.WithContext(ctx).First(&existingUser, user.ID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}

	// Update the user
	result := r.db.WithContext(ctx).Save(user)
	if result.Error != nil {
		return result.Error
	}

	// Check if any rows were affected
	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}

	return nil
}

// Delete soft deletes a user.
func (r *userRepositoryGorm) Delete(ctx context.Context, id uint) error {
	// Check if user exists first
	var user models.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}

	// Delete the user
	result := r.db.WithContext(ctx).Delete(&models.User{}, id)
	if result.Error != nil {
		return result.Error
	}

	// Check if any rows were affected
	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}

	return nil
}

// List retrieves users with pagination.
func (r *userRepositoryGorm) List(ctx context.Context, offset, limit int) ([]*models.User, error) {
	var users []*models.User
	err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&users).Error
	return users, err
}

// Count returns the total number of users.
func (r *userRepositoryGorm) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.User{}).Count(&count).Error
	return count, err
}

// GetByOrganization retrieves users by organization with pagination.
func (r *userRepositoryGorm) GetByOrganization(ctx context.Context, organizationID uint, offset, limit int) ([]*models.User, error) {
	var users []*models.User
	err := r.db.WithContext(ctx).
		Joins("JOIN user_organizations ON users.id = user_organizations.user_id").
		Where("user_organizations.organization_id = ?", organizationID).
		Offset(offset).Limit(limit).
		Find(&users).Error
	return users, err
}

// GetByRole retrieves users by role with pagination.
func (r *userRepositoryGorm) GetByRole(ctx context.Context, roleID uint, offset, limit int) ([]*models.User, error) {
	var users []*models.User
	err := r.db.WithContext(ctx).
		Joins("JOIN user_roles ON users.id = user_roles.user_id").
		Where("user_roles.role_id = ?", roleID).
		Offset(offset).Limit(limit).
		Find(&users).Error
	return users, err
}

// GetActiveUsers retrieves active users with pagination.
func (r *userRepositoryGorm) GetActiveUsers(ctx context.Context, offset, limit int) ([]*models.User, error) {
	var users []*models.User
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Offset(offset).Limit(limit).
		Find(&users).Error
	return users, err
}

// GetVerifiedUsers retrieves verified users with pagination.
func (r *userRepositoryGorm) GetVerifiedUsers(ctx context.Context, offset, limit int) ([]*models.User, error) {
	var users []*models.User
	err := r.db.WithContext(ctx).
		Where("email_verified = ?", true).
		Offset(offset).Limit(limit).
		Find(&users).Error
	return users, err
}

// SearchByEmail searches users by email pattern with pagination.
func (r *userRepositoryGorm) SearchByEmail(ctx context.Context, emailPattern string, offset, limit int) ([]*models.User, error) {
	var users []*models.User
	err := r.db.WithContext(ctx).
		Where("email LIKE ?", "%"+emailPattern+"%").
		Offset(offset).Limit(limit).
		Find(&users).Error
	return users, err
}

// SearchByUsername searches users by username pattern with pagination.
func (r *userRepositoryGorm) SearchByUsername(ctx context.Context, usernamePattern string, offset, limit int) ([]*models.User, error) {
	var users []*models.User
	err := r.db.WithContext(ctx).
		Where("username LIKE ?", "%"+usernamePattern+"%").
		Offset(offset).Limit(limit).
		Find(&users).Error
	return users, err
}

// SearchByName searches users by name pattern with pagination.
func (r *userRepositoryGorm) SearchByName(ctx context.Context, namePattern string, offset, limit int) ([]*models.User, error) {
	var users []*models.User
	err := r.db.WithContext(ctx).
		Where("first_name LIKE ? OR last_name LIKE ?", "%"+namePattern+"%", "%"+namePattern+"%").
		Offset(offset).Limit(limit).
		Find(&users).Error
	return users, err
}

// AddToOrganization adds a user to an organization.
func (r *userRepositoryGorm) AddToOrganization(ctx context.Context, userID, organizationID uint) error {
	return r.db.WithContext(ctx).Exec(
		"INSERT INTO user_organizations (user_id, organization_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
		userID, organizationID,
	).Error
}

// RemoveFromOrganization removes a user from an organization.
func (r *userRepositoryGorm) RemoveFromOrganization(ctx context.Context, userID, organizationID uint) error {
	return r.db.WithContext(ctx).Exec(
		"DELETE FROM user_organizations WHERE user_id = ? AND organization_id = ?",
		userID, organizationID,
	).Error
}

// AssignRole assigns a role to a user.
func (r *userRepositoryGorm) AssignRole(ctx context.Context, userID, roleID uint) error {
	return r.db.WithContext(ctx).Exec(
		"INSERT INTO user_roles (user_id, role_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
		userID, roleID,
	).Error
}

// UnassignRole unassigns a role from a user.
func (r *userRepositoryGorm) UnassignRole(ctx context.Context, userID, roleID uint) error {
	return r.db.WithContext(ctx).Exec(
		"DELETE FROM user_roles WHERE user_id = ? AND role_id = ?",
		userID, roleID,
	).Error
}

// GetWithRoles retrieves a user with preloaded roles.
func (r *userRepositoryGorm) GetWithRoles(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Preload("Roles").First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetWithOrganizations retrieves a user with preloaded organizations.
func (r *userRepositoryGorm) GetWithOrganizations(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Preload("Organizations").First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetWithAll retrieves a user with all preloaded relationships.
func (r *userRepositoryGorm) GetWithAll(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Preload("Roles").Preload("Organizations").First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}
