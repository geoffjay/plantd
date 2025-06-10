package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/go-playground/validator/v10"
	log "github.com/sirupsen/logrus"

	"github.com/geoffjay/plantd/identity/internal/models"
	"github.com/geoffjay/plantd/identity/internal/repositories"
)

// organizationServiceImpl implements the OrganizationService interface.
type organizationServiceImpl struct {
	orgRepo   repositories.OrganizationRepository
	userRepo  repositories.UserRepository
	roleRepo  repositories.RoleRepository
	validator *validator.Validate
}

// NewOrganizationService creates a new OrganizationService implementation.
func NewOrganizationService(
	orgRepo repositories.OrganizationRepository,
	userRepo repositories.UserRepository,
	roleRepo repositories.RoleRepository,
) OrganizationService {
	v := validator.New()

	// Register custom slug validator
	_ = v.RegisterValidation("slug", func(fl validator.FieldLevel) bool {
		slug := fl.Field().String()
		// Allow alphanumeric characters and hyphens, must start and end with alphanumeric
		slugRegex := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9]$|^[a-zA-Z0-9]$`)
		return slugRegex.MatchString(slug)
	})

	return &organizationServiceImpl{
		orgRepo:   orgRepo,
		userRepo:  userRepo,
		roleRepo:  roleRepo,
		validator: v,
	}
}

// CreateOrganization creates a new organization with validation and business rules.
func (s *organizationServiceImpl) CreateOrganization(ctx context.Context, req *CreateOrganizationRequest) (*models.Organization, error) {
	logger := log.WithFields(log.Fields{
		"service": "organization_service",
		"method":  "CreateOrganization",
		"name":    req.Name,
		"slug":    req.Slug,
	})

	// Validate request
	if err := s.validator.Struct(req); err != nil {
		logger.WithError(err).Error("validation failed")
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Generate slug if not provided
	slug := req.Slug
	if slug == "" {
		tempOrg := &models.Organization{Name: req.Name}
		tempOrg.GenerateSlug()
		slug = tempOrg.Slug
	}

	// Validate name uniqueness
	if err := s.ValidateNameUniqueness(ctx, req.Name, nil); err != nil {
		logger.WithError(err).Error("name validation failed")
		return nil, err
	}

	// Validate slug uniqueness
	if err := s.ValidateSlugUniqueness(ctx, slug, nil); err != nil {
		logger.WithError(err).Error("slug validation failed")
		return nil, err
	}

	// Create organization model
	org := &models.Organization{
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
		IsActive:    true,
	}

	// Override default if specified
	if req.IsActive != nil {
		org.IsActive = *req.IsActive
	}

	// Create organization in database
	if err := s.orgRepo.Create(ctx, org); err != nil {
		logger.WithError(err).Error("failed to create organization")
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}

	logger.WithField("org_id", org.ID).Info("organization created successfully")
	return org, nil
}

// GetOrganizationByID retrieves an organization by ID.
func (s *organizationServiceImpl) GetOrganizationByID(ctx context.Context, id uint) (*models.Organization, error) {
	logger := log.WithFields(log.Fields{
		"service": "organization_service",
		"method":  "GetOrganizationByID",
		"org_id":  id,
	})

	org, err := s.orgRepo.GetByID(ctx, id)
	if err != nil {
		logger.WithError(err).Error("failed to get organization")
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	if org == nil {
		logger.Debug("organization not found")
		return nil, errors.New("organization not found")
	}

	return org, nil
}

// GetOrganizationBySlug retrieves an organization by slug.
func (s *organizationServiceImpl) GetOrganizationBySlug(ctx context.Context, slug string) (*models.Organization, error) {
	logger := log.WithFields(log.Fields{
		"service": "organization_service",
		"method":  "GetOrganizationBySlug",
		"slug":    slug,
	})

	if slug == "" {
		return nil, errors.New("slug cannot be empty")
	}

	org, err := s.orgRepo.GetBySlug(ctx, slug)
	if err != nil {
		logger.WithError(err).Error("failed to get organization")
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	if org == nil {
		logger.Debug("organization not found")
		return nil, errors.New("organization not found")
	}

	return org, nil
}

// UpdateOrganization updates an organization with validation and business rules.
func (s *organizationServiceImpl) UpdateOrganization(ctx context.Context, id uint, req *UpdateOrganizationRequest) (*models.Organization, error) {
	logger := createServiceLogger("organization_service", "UpdateOrganization", log.Fields{"org_id": id})

	// Validate request
	if err := s.validator.Struct(req); err != nil {
		return nil, logAndError(logger, "validation failed", err)
	}

	// Get existing organization
	org, err := s.getExistingOrganization(ctx, id, logger)
	if err != nil {
		return nil, err
	}

	// Update organization fields with validation
	if err := s.updateOrganizationFields(ctx, id, req, org, logger); err != nil {
		return nil, err
	}

	// Save updated organization
	if err := s.orgRepo.Update(ctx, org); err != nil {
		return nil, logAndError(logger, "failed to update organization", err)
	}

	logSuccess(logger, "organization updated successfully", nil)
	return org, nil
}

// getExistingOrganization retrieves and validates an existing organization.
func (s *organizationServiceImpl) getExistingOrganization(ctx context.Context, id uint, logger *log.Entry) (*models.Organization, error) {
	org, err := s.orgRepo.GetByID(ctx, id)
	if err != nil {
		return nil, logAndError(logger, "failed to get organization", err)
	}
	if org == nil {
		return nil, logAndErrorSimple(logger, "organization "+ErrNotFound)
	}
	return org, nil
}

// updateOrganizationFields updates organization fields with validation.
func (s *organizationServiceImpl) updateOrganizationFields(
	ctx context.Context,
	id uint,
	req *UpdateOrganizationRequest,
	org *models.Organization,
	logger *log.Entry,
) error {
	// Check name uniqueness if name is being updated
	if req.Name != nil && *req.Name != org.Name {
		if err := s.ValidateNameUniqueness(ctx, *req.Name, &id); err != nil {
			return logAndError(logger, "name validation failed", err)
		}
		org.Name = *req.Name
	}

	// Check slug uniqueness if slug is being updated
	if req.Slug != nil && *req.Slug != org.Slug {
		if err := s.ValidateSlugUniqueness(ctx, *req.Slug, &id); err != nil {
			return logAndError(logger, "slug validation failed", err)
		}
		org.Slug = *req.Slug
	}

	// Update other fields
	if req.Description != nil {
		org.Description = *req.Description
	}
	if req.IsActive != nil {
		org.IsActive = *req.IsActive
	}

	return nil
}

// DeleteOrganization soft deletes an organization.
func (s *organizationServiceImpl) DeleteOrganization(ctx context.Context, id uint) error {
	logger := log.WithFields(log.Fields{
		"service": "organization_service",
		"method":  "DeleteOrganization",
		"org_id":  id,
	})

	// Check if organization exists
	org, err := s.orgRepo.GetByID(ctx, id)
	if err != nil {
		logger.WithError(err).Error("failed to get organization")
		return fmt.Errorf("failed to get organization: %w", err)
	}
	if org == nil {
		logger.Error("organization not found")
		return errors.New("organization not found")
	}

	// TODO: Check if organization has dependent entities (users, etc.)
	// For now, proceed with deletion

	// Delete organization
	if err := s.orgRepo.Delete(ctx, id); err != nil {
		logger.WithError(err).Error("failed to delete organization")
		return fmt.Errorf("failed to delete organization: %w", err)
	}

	logger.Info("organization deleted successfully")
	return nil
}

// ListOrganizations lists organizations with pagination and filtering.
func (s *organizationServiceImpl) ListOrganizations(ctx context.Context, req *ListOrganizationsRequest) ([]*models.Organization, error) {
	logger := log.WithFields(log.Fields{
		"service": "organization_service",
		"method":  "ListOrganizations",
		"offset":  req.Offset,
		"limit":   req.Limit,
	})

	// Validate request
	if err := s.validator.Struct(req); err != nil {
		logger.WithError(err).Error("validation failed")
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// For now, use basic list without complex filtering
	orgs, err := s.orgRepo.List(ctx, req.Offset, req.Limit)
	if err != nil {
		logger.WithError(err).Error("failed to list organizations")
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}

	logger.WithField("count", len(orgs)).Debug("organizations listed successfully")
	return orgs, nil
}

// SearchOrganizations searches organizations by query and fields.
func (s *organizationServiceImpl) SearchOrganizations(ctx context.Context, req *SearchOrganizationsRequest) ([]*models.Organization, error) {
	logger := log.WithFields(log.Fields{
		"service": "organization_service",
		"method":  "SearchOrganizations",
		"query":   req.Query,
		"fields":  req.SearchFields,
	})

	// Validate request
	if err := s.validator.Struct(req); err != nil {
		logger.WithError(err).Error("validation failed")
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Simple implementation using repository search methods
	var orgs []*models.Organization
	var err error

	// Search by specified fields
	for _, field := range req.SearchFields {
		switch field {
		case FieldName:
			orgs, err = s.orgRepo.SearchByName(ctx, req.Query, req.Offset, req.Limit)
		case FieldSlug:
			// TODO: Implement SearchBySlug in repository
			logger.Warn("slug search not implemented yet")
		case FieldDescription:
			// TODO: Implement SearchByDescription in repository
			logger.Warn("description search not implemented yet")
		}
		if err != nil {
			logger.WithError(err).Error("search failed")
			return nil, fmt.Errorf("search failed: %w", err)
		}
		if len(orgs) > 0 {
			break // Return first successful search
		}
	}

	logger.WithField("count", len(orgs)).Debug("organizations searched successfully")
	return orgs, nil
}

// CountOrganizations returns the total number of organizations.
func (s *organizationServiceImpl) CountOrganizations(ctx context.Context) (int64, error) {
	logger := log.WithFields(log.Fields{
		"service": "organization_service",
		"method":  "CountOrganizations",
	})

	count, err := s.orgRepo.Count(ctx)
	if err != nil {
		logger.WithError(err).Error("failed to count organizations")
		return 0, fmt.Errorf("failed to count organizations: %w", err)
	}

	logger.WithField("count", count).Debug("organizations counted successfully")
	return count, nil
}

// ActivateOrganization activates an organization.
func (s *organizationServiceImpl) ActivateOrganization(ctx context.Context, id uint) error {
	logger := log.WithFields(log.Fields{
		"service": "organization_service",
		"method":  "ActivateOrganization",
		"org_id":  id,
	})

	org, err := s.orgRepo.GetByID(ctx, id)
	if err != nil {
		logger.WithError(err).Error("failed to get organization")
		return fmt.Errorf("failed to get organization: %w", err)
	}
	if org == nil {
		logger.Error("organization not found")
		return errors.New("organization not found")
	}

	if org.IsActive {
		logger.Debug("organization already active")
		return nil
	}

	org.IsActive = true
	if err := s.orgRepo.Update(ctx, org); err != nil {
		logger.WithError(err).Error("failed to activate organization")
		return fmt.Errorf("failed to activate organization: %w", err)
	}

	logger.Info("organization activated successfully")
	return nil
}

// DeactivateOrganization deactivates an organization.
func (s *organizationServiceImpl) DeactivateOrganization(ctx context.Context, id uint) error {
	logger := log.WithFields(log.Fields{
		"service": "organization_service",
		"method":  "DeactivateOrganization",
		"org_id":  id,
	})

	org, err := s.orgRepo.GetByID(ctx, id)
	if err != nil {
		logger.WithError(err).Error("failed to get organization")
		return fmt.Errorf("failed to get organization: %w", err)
	}
	if org == nil {
		logger.Error("organization not found")
		return errors.New("organization not found")
	}

	if !org.IsActive {
		logger.Debug("organization already inactive")
		return nil
	}

	org.IsActive = false
	if err := s.orgRepo.Update(ctx, org); err != nil {
		logger.WithError(err).Error("failed to deactivate organization")
		return fmt.Errorf("failed to deactivate organization: %w", err)
	}

	logger.Info("organization deactivated successfully")
	return nil
}

// ValidateSlugUniqueness validates that a slug is unique.
func (s *organizationServiceImpl) ValidateSlugUniqueness(ctx context.Context, slug string, excludeID *uint) error {
	logger := log.WithFields(log.Fields{
		"service": "organization_service",
		"method":  "ValidateSlugUniqueness",
		"slug":    slug,
	})

	existing, err := s.orgRepo.GetBySlug(ctx, slug)
	if err != nil {
		logger.WithError(err).Error("failed to check slug uniqueness")
		return fmt.Errorf("failed to check slug uniqueness: %w", err)
	}

	if existing != nil && (excludeID == nil || existing.ID != *excludeID) {
		logger.Error("slug already exists")
		return errors.New("slug already exists")
	}

	return nil
}

// ValidateNameUniqueness validates that a name is unique.
func (s *organizationServiceImpl) ValidateNameUniqueness(ctx context.Context, name string, excludeID *uint) error {
	logger := log.WithFields(log.Fields{
		"service": "organization_service",
		"method":  "ValidateNameUniqueness",
		"name":    name,
	})

	// Use SearchByName to find exact matches
	existing, err := s.orgRepo.SearchByName(ctx, name, 0, 1)
	if err != nil {
		logger.WithError(err).Error("failed to check name uniqueness")
		return fmt.Errorf("failed to check name uniqueness: %w", err)
	}

	// Check if we found an exact match
	for _, org := range existing {
		if org.Name == name && (excludeID == nil || org.ID != *excludeID) {
			logger.Error("name already exists")
			return errors.New("name already exists")
		}
	}

	return nil
}

// Placeholder implementations for member and role management
// These will be fully implemented when the relationship methods are added to repositories

// AddUserToOrganization adds a user to an organization.
func (s *organizationServiceImpl) AddUserToOrganization(ctx context.Context, orgID, userID uint) error { //nolint:revive
	// TODO: Implement when organization membership methods are added to repositories
	return errors.New("not implemented yet")
}

// RemoveUserFromOrganization removes a user from an organization.
func (s *organizationServiceImpl) RemoveUserFromOrganization(ctx context.Context, orgID, userID uint) error { //nolint:revive
	// TODO: Implement when organization membership methods are added to repositories
	return errors.New("not implemented yet")
}

// GetOrganizationMembers returns the members of an organization.
func (s *organizationServiceImpl) GetOrganizationMembers(ctx context.Context, orgID uint, offset, limit int) ([]*models.User, error) { //nolint:revive
	// TODO: Implement when organization membership methods are added to repositories
	return nil, errors.New("not implemented yet")
}

// GetOrganizationMemberCount returns the number of members in an organization.
func (s *organizationServiceImpl) GetOrganizationMemberCount(ctx context.Context, orgID uint) (int64, error) { //nolint:revive
	// TODO: Implement when organization membership methods are added to repositories
	return 0, errors.New("not implemented yet")
}

// AssignRoleToOrganization assigns a role to an organization.
func (s *organizationServiceImpl) AssignRoleToOrganization(ctx context.Context, orgID, roleID uint) error { //nolint:revive
	// TODO: Implement when role assignment methods are added to repositories
	return errors.New("not implemented yet")
}

// RemoveRoleFromOrganization removes a role from an organization.
func (s *organizationServiceImpl) RemoveRoleFromOrganization(ctx context.Context, orgID, roleID uint) error { //nolint:revive
	// TODO: Implement when role assignment methods are added to repositories
	return errors.New("not implemented yet")
}

// GetOrganizationRoles returns the roles assigned to an organization.
func (s *organizationServiceImpl) GetOrganizationRoles(ctx context.Context, orgID uint) ([]*models.Role, error) { //nolint:revive
	// TODO: Implement when role assignment methods are added to repositories
	return nil, errors.New("not implemented yet")
}
