package repositories

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/geoffjay/plantd/identity/internal/models"
)

// organizationRepositoryGorm implements OrganizationRepository using GORM.
type organizationRepositoryGorm struct {
	db *gorm.DB
}

// NewOrganizationRepository creates a new OrganizationRepository implementation using GORM.
func NewOrganizationRepository(db *gorm.DB) OrganizationRepository {
	return &organizationRepositoryGorm{db: db}
}

// Create creates a new organization.
func (r *organizationRepositoryGorm) Create(ctx context.Context, organization *models.Organization) error {
	return r.db.WithContext(ctx).Create(organization).Error
}

// GetByID retrieves an organization by ID.
func (r *organizationRepositoryGorm) GetByID(ctx context.Context, id uint) (*models.Organization, error) {
	var organization models.Organization
	err := r.db.WithContext(ctx).First(&organization, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &organization, nil
}

// GetBySlug retrieves an organization by slug.
func (r *organizationRepositoryGorm) GetBySlug(ctx context.Context, slug string) (*models.Organization, error) {
	var organization models.Organization
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&organization).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &organization, nil
}

// Update updates an organization.
func (r *organizationRepositoryGorm) Update(ctx context.Context, organization *models.Organization) error {
	return r.db.WithContext(ctx).Save(organization).Error
}

// Delete soft deletes an organization.
func (r *organizationRepositoryGorm) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Organization{}, id).Error
}

// List retrieves organizations with pagination.
func (r *organizationRepositoryGorm) List(ctx context.Context, offset, limit int) ([]*models.Organization, error) {
	var organizations []*models.Organization
	err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&organizations).Error
	return organizations, err
}

// Count returns the total number of organizations.
func (r *organizationRepositoryGorm) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Organization{}).Count(&count).Error
	return count, err
}

// GetActiveOrganizations retrieves active organizations with pagination.
func (r *organizationRepositoryGorm) GetActiveOrganizations(ctx context.Context, offset, limit int) ([]*models.Organization, error) {
	var organizations []*models.Organization
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Offset(offset).Limit(limit).
		Find(&organizations).Error
	return organizations, err
}

// GetByUser retrieves organizations by user with pagination.
func (r *organizationRepositoryGorm) GetByUser(ctx context.Context, userID uint, offset, limit int) ([]*models.Organization, error) {
	var organizations []*models.Organization
	err := r.db.WithContext(ctx).
		Joins("JOIN user_organizations ON organizations.id = user_organizations.organization_id").
		Where("user_organizations.user_id = ?", userID).
		Offset(offset).Limit(limit).
		Find(&organizations).Error
	return organizations, err
}

// SearchByName searches organizations by name pattern with pagination.
func (r *organizationRepositoryGorm) SearchByName(ctx context.Context, namePattern string, offset, limit int) ([]*models.Organization, error) {
	var organizations []*models.Organization
	err := r.db.WithContext(ctx).
		Where("name LIKE ?", "%"+namePattern+"%").
		Offset(offset).Limit(limit).
		Find(&organizations).Error
	return organizations, err
}

// SearchBySlug searches organizations by slug pattern with pagination.
func (r *organizationRepositoryGorm) SearchBySlug(ctx context.Context, slugPattern string, offset, limit int) ([]*models.Organization, error) {
	var organizations []*models.Organization
	err := r.db.WithContext(ctx).
		Where("slug LIKE ?", "%"+slugPattern+"%").
		Offset(offset).Limit(limit).
		Find(&organizations).Error
	return organizations, err
}

// AddUser adds a user to an organization.
func (r *organizationRepositoryGorm) AddUser(ctx context.Context, organizationID, userID uint) error {
	return r.db.WithContext(ctx).Exec(
		"INSERT INTO user_organizations (organization_id, user_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
		organizationID, userID,
	).Error
}

// RemoveUser removes a user from an organization.
func (r *organizationRepositoryGorm) RemoveUser(ctx context.Context, organizationID, userID uint) error {
	return r.db.WithContext(ctx).Exec(
		"DELETE FROM user_organizations WHERE organization_id = ? AND user_id = ?",
		organizationID, userID,
	).Error
}

// AddRole adds a role to an organization.
func (r *organizationRepositoryGorm) AddRole(ctx context.Context, organizationID, roleID uint) error {
	return r.db.WithContext(ctx).Exec(
		"INSERT INTO organization_roles (organization_id, role_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
		organizationID, roleID,
	).Error
}

// RemoveRole removes a role from an organization.
func (r *organizationRepositoryGorm) RemoveRole(ctx context.Context, organizationID, roleID uint) error {
	return r.db.WithContext(ctx).Exec(
		"DELETE FROM organization_roles WHERE organization_id = ? AND role_id = ?",
		organizationID, roleID,
	).Error
}

// GetMembers retrieves organization members with pagination.
func (r *organizationRepositoryGorm) GetMembers(ctx context.Context, organizationID uint, offset, limit int) ([]*models.User, error) {
	var users []*models.User
	err := r.db.WithContext(ctx).
		Joins("JOIN user_organizations ON users.id = user_organizations.user_id").
		Where("user_organizations.organization_id = ?", organizationID).
		Offset(offset).Limit(limit).
		Find(&users).Error
	return users, err
}

// GetMembersWithRoles retrieves organization members with preloaded roles.
func (r *organizationRepositoryGorm) GetMembersWithRoles(ctx context.Context, organizationID uint, offset, limit int) ([]*models.User, error) {
	var users []*models.User
	err := r.db.WithContext(ctx).
		Preload("Roles").
		Joins("JOIN user_organizations ON users.id = user_organizations.user_id").
		Where("user_organizations.organization_id = ?", organizationID).
		Offset(offset).Limit(limit).
		Find(&users).Error
	return users, err
}

// CountMembers returns the number of members in an organization.
func (r *organizationRepositoryGorm) CountMembers(ctx context.Context, organizationID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("user_organizations").
		Where("organization_id = ?", organizationID).
		Count(&count).Error
	return count, err
}

// GetWithUsers retrieves an organization with preloaded users.
func (r *organizationRepositoryGorm) GetWithUsers(ctx context.Context, id uint) (*models.Organization, error) {
	var organization models.Organization
	err := r.db.WithContext(ctx).Preload("Users").First(&organization, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &organization, nil
}

// GetWithRoles retrieves an organization with preloaded roles.
func (r *organizationRepositoryGorm) GetWithRoles(ctx context.Context, id uint) (*models.Organization, error) {
	var organization models.Organization
	err := r.db.WithContext(ctx).Preload("Roles").First(&organization, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &organization, nil
}

// GetWithAll retrieves an organization with all preloaded relationships.
func (r *organizationRepositoryGorm) GetWithAll(ctx context.Context, id uint) (*models.Organization, error) {
	var organization models.Organization
	err := r.db.WithContext(ctx).Preload("Users").Preload("Roles").First(&organization, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &organization, nil
}
