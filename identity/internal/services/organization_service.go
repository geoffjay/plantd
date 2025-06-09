package services

import (
	"context"

	"github.com/geoffjay/plantd/identity/internal/models"
)

// OrganizationService defines the interface for organization business logic operations.
type OrganizationService interface {
	// Organization CRUD operations with business rules
	CreateOrganization(ctx context.Context, req *CreateOrganizationRequest) (*models.Organization, error)
	GetOrganizationByID(ctx context.Context, id uint) (*models.Organization, error)
	GetOrganizationBySlug(ctx context.Context, slug string) (*models.Organization, error)
	UpdateOrganization(ctx context.Context, id uint, req *UpdateOrganizationRequest) (*models.Organization, error)
	DeleteOrganization(ctx context.Context, id uint) error

	// Organization listing and searching
	ListOrganizations(ctx context.Context, req *ListOrganizationsRequest) ([]*models.Organization, error)
	SearchOrganizations(ctx context.Context, req *SearchOrganizationsRequest) ([]*models.Organization, error)
	CountOrganizations(ctx context.Context) (int64, error)

	// Organization status management
	ActivateOrganization(ctx context.Context, id uint) error
	DeactivateOrganization(ctx context.Context, id uint) error

	// Member management
	AddUserToOrganization(ctx context.Context, orgID, userID uint) error
	RemoveUserFromOrganization(ctx context.Context, orgID, userID uint) error
	GetOrganizationMembers(ctx context.Context, orgID uint, offset, limit int) ([]*models.User, error)
	GetOrganizationMemberCount(ctx context.Context, orgID uint) (int64, error)

	// Role management within organization
	AssignRoleToOrganization(ctx context.Context, orgID, roleID uint) error
	RemoveRoleFromOrganization(ctx context.Context, orgID, roleID uint) error
	GetOrganizationRoles(ctx context.Context, orgID uint) ([]*models.Role, error)

	// Organization validation
	ValidateSlugUniqueness(ctx context.Context, slug string, excludeID *uint) error
	ValidateNameUniqueness(ctx context.Context, name string, excludeID *uint) error
}

// CreateOrganizationRequest represents the request to create a new organization.
type CreateOrganizationRequest struct {
	Name        string `json:"name" validate:"required,min=1,max=255"`
	Slug        string `json:"slug" validate:"omitempty,min=1,max=100,alphanum"`
	Description string `json:"description" validate:"max=1000"`
	IsActive    *bool  `json:"is_active,omitempty"`
}

// UpdateOrganizationRequest represents the request to update an organization.
type UpdateOrganizationRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Slug        *string `json:"slug,omitempty" validate:"omitempty,min=1,max=100,alphanum"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=1000"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

// ListOrganizationsRequest represents the request to list organizations with pagination and filtering.
type ListOrganizationsRequest struct {
	Offset          int    `json:"offset" validate:"min=0"`
	Limit           int    `json:"limit" validate:"min=1,max=100"`
	IncludeInactive bool   `json:"include_inactive"`
	SortBy          string `json:"sort_by" validate:"omitempty,oneof=id name slug created_at updated_at"`
	SortOrder       string `json:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// SearchOrganizationsRequest represents the request to search organizations.
type SearchOrganizationsRequest struct {
	Query           string   `json:"query" validate:"required,min=1"`
	SearchFields    []string `json:"search_fields" validate:"required"`
	Offset          int      `json:"offset" validate:"min=0"`
	Limit           int      `json:"limit" validate:"min=1,max=100"`
	IncludeInactive bool     `json:"include_inactive"`
}
