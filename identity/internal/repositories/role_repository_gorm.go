package repositories

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/geoffjay/plantd/identity/internal/models"
)

// roleRepositoryGorm implements RoleRepository using GORM.
type roleRepositoryGorm struct {
	db *gorm.DB
}

// NewRoleRepository creates a new RoleRepository implementation using GORM.
func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &roleRepositoryGorm{db: db}
}

// Create creates a new role.
func (r *roleRepositoryGorm) Create(ctx context.Context, role *models.Role) error {
	return r.db.WithContext(ctx).Create(role).Error
}

// GetByID retrieves a role by ID.
func (r *roleRepositoryGorm) GetByID(ctx context.Context, id uint) (*models.Role, error) {
	var role models.Role
	err := r.db.WithContext(ctx).First(&role, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

// GetByName retrieves a role by name.
func (r *roleRepositoryGorm) GetByName(ctx context.Context, name string) (*models.Role, error) {
	var role models.Role
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&role).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

// Update updates a role.
func (r *roleRepositoryGorm) Update(ctx context.Context, role *models.Role) error {
	return r.db.WithContext(ctx).Save(role).Error
}

// Delete soft deletes a role.
func (r *roleRepositoryGorm) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Role{}, id).Error
}

// List retrieves roles with pagination.
func (r *roleRepositoryGorm) List(ctx context.Context, offset, limit int) ([]*models.Role, error) {
	var roles []*models.Role
	err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&roles).Error
	return roles, err
}

// Count returns the total number of roles.
func (r *roleRepositoryGorm) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Role{}).Count(&count).Error
	return count, err
}

// GetByScope retrieves roles by scope with pagination.
func (r *roleRepositoryGorm) GetByScope(ctx context.Context, scope models.RoleScope, offset, limit int) ([]*models.Role, error) {
	var roles []*models.Role
	err := r.db.WithContext(ctx).
		Where("scope = ?", scope).
		Offset(offset).Limit(limit).
		Find(&roles).Error
	return roles, err
}

// GetGlobalRoles retrieves global roles with pagination.
func (r *roleRepositoryGorm) GetGlobalRoles(ctx context.Context, offset, limit int) ([]*models.Role, error) {
	return r.GetByScope(ctx, models.RoleScopeGlobal, offset, limit)
}

// GetOrganizationRoles retrieves organization-scoped roles with pagination.
func (r *roleRepositoryGorm) GetOrganizationRoles(ctx context.Context, offset, limit int) ([]*models.Role, error) {
	return r.GetByScope(ctx, models.RoleScopeOrganization, offset, limit)
}

// GetByUser retrieves roles by user with pagination.
func (r *roleRepositoryGorm) GetByUser(ctx context.Context, userID uint, offset, limit int) ([]*models.Role, error) {
	var roles []*models.Role
	query := r.db.WithContext(ctx).
		Joins("JOIN user_roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ?", userID).
		Offset(offset)

	// Apply limit only when a positive value is provided; GORM treats -1/0 differently across drivers
	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&roles).Error
	return roles, err
}

// GetByUserAndOrganization retrieves roles by user and organization with pagination.
func (r *roleRepositoryGorm) GetByUserAndOrganization(ctx context.Context, userID, organizationID uint, offset, limit int) ([]*models.Role, error) {
	var roles []*models.Role
	err := r.db.WithContext(ctx).
		Joins("JOIN user_roles ON roles.id = user_roles.role_id").
		Joins("JOIN organization_roles ON roles.id = organization_roles.role_id").
		Where("user_roles.user_id = ? AND organization_roles.organization_id = ?", userID, organizationID).
		Offset(offset).Limit(limit).
		Find(&roles).Error
	return roles, err
}

// GetByOrganization retrieves roles by organization with pagination.
func (r *roleRepositoryGorm) GetByOrganization(ctx context.Context, organizationID uint, offset, limit int) ([]*models.Role, error) {
	var roles []*models.Role
	err := r.db.WithContext(ctx).
		Joins("JOIN organization_roles ON roles.id = organization_roles.role_id").
		Where("organization_roles.organization_id = ?", organizationID).
		Offset(offset).Limit(limit).
		Find(&roles).Error
	return roles, err
}

// SearchByName searches roles by name pattern with pagination.
func (r *roleRepositoryGorm) SearchByName(ctx context.Context, namePattern string, offset, limit int) ([]*models.Role, error) {
	var roles []*models.Role
	err := r.db.WithContext(ctx).
		Where("name LIKE ?", "%"+namePattern+"%").
		Offset(offset).Limit(limit).
		Find(&roles).Error
	return roles, err
}

// SearchByDescription searches roles by description pattern with pagination.
func (r *roleRepositoryGorm) SearchByDescription(ctx context.Context, descriptionPattern string, offset, limit int) ([]*models.Role, error) {
	var roles []*models.Role
	err := r.db.WithContext(ctx).
		Where("description LIKE ?", "%"+descriptionPattern+"%").
		Offset(offset).Limit(limit).
		Find(&roles).Error
	return roles, err
}

// GetRolesWithPermission retrieves roles that have a specific permission.
func (r *roleRepositoryGorm) GetRolesWithPermission(ctx context.Context, permission string, offset, limit int) ([]*models.Role, error) {
	var roles []*models.Role
	err := r.db.WithContext(ctx).
		Where("permissions LIKE ?", "%"+permission+"%").
		Offset(offset).Limit(limit).
		Find(&roles).Error
	return roles, err
}

// AssignToUser assigns a role to a user.
func (r *roleRepositoryGorm) AssignToUser(ctx context.Context, roleID, userID uint) error {
	return r.db.WithContext(ctx).Exec(
		"INSERT INTO user_roles (role_id, user_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
		roleID, userID,
	).Error
}

// UnassignFromUser unassigns a role from a user.
func (r *roleRepositoryGorm) UnassignFromUser(ctx context.Context, roleID, userID uint) error {
	return r.db.WithContext(ctx).Exec(
		"DELETE FROM user_roles WHERE role_id = ? AND user_id = ?",
		roleID, userID,
	).Error
}

// AssignToOrganization assigns a role to an organization.
func (r *roleRepositoryGorm) AssignToOrganization(ctx context.Context, roleID, organizationID uint) error {
	return r.db.WithContext(ctx).Exec(
		"INSERT INTO organization_roles (role_id, organization_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
		roleID, organizationID,
	).Error
}

// UnassignFromOrganization unassigns a role from an organization.
func (r *roleRepositoryGorm) UnassignFromOrganization(ctx context.Context, roleID, organizationID uint) error {
	return r.db.WithContext(ctx).Exec(
		"DELETE FROM organization_roles WHERE role_id = ? AND organization_id = ?",
		roleID, organizationID,
	).Error
}

// GetWithUsers retrieves a role with preloaded users.
func (r *roleRepositoryGorm) GetWithUsers(ctx context.Context, id uint) (*models.Role, error) {
	var role models.Role
	err := r.db.WithContext(ctx).Preload("Users").First(&role, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

// GetWithOrganizations retrieves a role with preloaded organizations.
func (r *roleRepositoryGorm) GetWithOrganizations(ctx context.Context, id uint) (*models.Role, error) {
	var role models.Role
	err := r.db.WithContext(ctx).Preload("Organizations").First(&role, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}

// GetWithAll retrieves a role with all preloaded relationships.
func (r *roleRepositoryGorm) GetWithAll(ctx context.Context, id uint) (*models.Role, error) {
	var role models.Role
	err := r.db.WithContext(ctx).Preload("Users").Preload("Organizations").First(&role, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &role, nil
}
