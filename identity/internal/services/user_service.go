package services

import (
	"context"

	"github.com/geoffjay/plantd/identity/internal/models"
)

// UserService defines the interface for user business logic operations.
type UserService interface {
	// User CRUD operations with business rules
	CreateUser(ctx context.Context, req *CreateUserRequest) (*models.User, error)
	GetUserByID(ctx context.Context, id uint) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	UpdateUser(ctx context.Context, id uint, req *UpdateUserRequest) (*models.User, error)
	DeleteUser(ctx context.Context, id uint) error

	// User listing and searching
	ListUsers(ctx context.Context, req *ListUsersRequest) ([]*models.User, error)
	SearchUsers(ctx context.Context, req *SearchUsersRequest) ([]*models.User, error)
	CountUsers(ctx context.Context) (int64, error)

	// User status management
	ActivateUser(ctx context.Context, id uint) error
	DeactivateUser(ctx context.Context, id uint) error
	VerifyUserEmail(ctx context.Context, id uint) error

	// User authentication helpers
	GetActiveUsers(ctx context.Context, offset, limit int) ([]*models.User, error)
	GetVerifiedUsers(ctx context.Context, offset, limit int) ([]*models.User, error)

	// Role and organization management
	AssignUserToRole(ctx context.Context, userID, roleID uint) error
	RemoveUserFromRole(ctx context.Context, userID, roleID uint) error
	GetUserRoles(ctx context.Context, userID uint) ([]*models.Role, error)

	AssignUserToOrganization(ctx context.Context, userID, orgID uint) error
	RemoveUserFromOrganization(ctx context.Context, userID, orgID uint) error
	GetUserOrganizations(ctx context.Context, userID uint) ([]*models.Organization, error)
}

// CreateUserRequest represents the request to create a new user.
type CreateUserRequest struct {
	Email            string `json:"email" validate:"required,email"`
	Username         string `json:"username" validate:"required,min=3,max=50"`
	Password         string `json:"password" validate:"required,min=8"`
	FirstName        string `json:"first_name" validate:"max=100"`
	LastName         string `json:"last_name" validate:"max=100"`
	SendWelcomeEmail bool   `json:"send_welcome_email"`
}

// UpdateUserRequest represents the request to update a user.
type UpdateUserRequest struct {
	Email     *string `json:"email,omitempty" validate:"omitempty,email"`
	Username  *string `json:"username,omitempty" validate:"omitempty,min=3,max=50"`
	FirstName *string `json:"first_name,omitempty" validate:"omitempty,max=100"`
	LastName  *string `json:"last_name,omitempty" validate:"omitempty,max=100"`
	IsActive  *bool   `json:"is_active,omitempty"`
}

// ListUsersRequest represents the request to list users with pagination and filtering.
type ListUsersRequest struct {
	Offset            int    `json:"offset" validate:"min=0"`
	Limit             int    `json:"limit" validate:"min=1,max=100"`
	IncludeInactive   bool   `json:"include_inactive"`
	IncludeUnverified bool   `json:"include_unverified"`
	SortBy            string `json:"sort_by" validate:"omitempty,oneof=id email username created_at updated_at"`
	SortOrder         string `json:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// SearchUsersRequest represents the request to search users.
type SearchUsersRequest struct {
	Query           string   `json:"query" validate:"required,min=1"`
	SearchFields    []string `json:"search_fields" validate:"required"`
	Offset          int      `json:"offset" validate:"min=0"`
	Limit           int      `json:"limit" validate:"min=1,max=100"`
	IncludeInactive bool     `json:"include_inactive"`
}
