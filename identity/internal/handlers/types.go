// Package handlers provides MDP protocol handlers for the identity service.
package handlers

import (
	"time"

	"github.com/geoffjay/plantd/identity/internal/models"
)

// Common request/response types for MDP protocol

// RequestHeader represents common request metadata.
type RequestHeader struct {
	RequestID string `json:"request_id"`
	UserID    *uint  `json:"user_id,omitempty"`
	OrgID     *uint  `json:"org_id,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// ResponseHeader represents common response metadata.
type ResponseHeader struct {
	RequestID string `json:"request_id"`
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Header ResponseHeader `json:"header"`
	Code   string         `json:"code"`
	Detail string         `json:"detail,omitempty"`
}

// HealthCheckRequest represents a health check request.
type HealthCheckRequest struct {
	Header RequestHeader `json:"header"`
}

// HealthCheckResponse represents a health check response.
type HealthCheckResponse struct {
	Header   ResponseHeader `json:"header"`
	Status   string         `json:"status"`
	Version  string         `json:"version"`
	Uptime   time.Duration  `json:"uptime"`
	DBStatus string         `json:"db_status"`
	Services []string       `json:"services"`
}

// Authentication-related types

// LoginRequest represents a login request.
type LoginRequest struct {
	Header     RequestHeader `json:"header"`
	Identifier string        `json:"identifier" validate:"required"` // email or username
	Password   string        `json:"password" validate:"required"`
	IPAddress  string        `json:"ip_address,omitempty"`
	UserAgent  string        `json:"user_agent,omitempty"`
}

// LoginResponse represents a login response.
type LoginResponse struct {
	Header       ResponseHeader `json:"header"`
	User         *models.User   `json:"user,omitempty"`
	AccessToken  string         `json:"access_token,omitempty"`
	RefreshToken string         `json:"refresh_token,omitempty"`
	ExpiresAt    int64          `json:"expires_at,omitempty"`
}

// RefreshTokenRequest represents a token refresh request.
type RefreshTokenRequest struct {
	Header       RequestHeader `json:"header"`
	RefreshToken string        `json:"refresh_token" validate:"required"`
	IPAddress    string        `json:"ip_address,omitempty"`
}

// RefreshTokenResponse represents a token refresh response.
type RefreshTokenResponse struct {
	Header       ResponseHeader `json:"header"`
	AccessToken  string         `json:"access_token,omitempty"`
	RefreshToken string         `json:"refresh_token,omitempty"`
	ExpiresAt    int64          `json:"expires_at,omitempty"`
}

// LogoutRequest represents a logout request.
type LogoutRequest struct {
	Header      RequestHeader `json:"header"`
	AccessToken string        `json:"access_token" validate:"required"`
}

// LogoutResponse represents a logout response.
type LogoutResponse struct {
	Header ResponseHeader `json:"header"`
}

// ValidateTokenRequest represents a token validation request.
type ValidateTokenRequest struct {
	Header RequestHeader `json:"header"`
	Token  string        `json:"token" validate:"required"`
}

// ValidateTokenResponse represents a token validation response.
type ValidateTokenResponse struct {
	Header      ResponseHeader `json:"header"`
	Valid       bool           `json:"valid"`
	UserID      *uint          `json:"user_id,omitempty"`
	Email       string         `json:"email,omitempty"`
	Roles       []string       `json:"roles,omitempty"`
	Permissions []string       `json:"permissions,omitempty"`
	ExpiresAt   *int64         `json:"expires_at,omitempty"`
}

// User management types

// CreateUserRequest represents a request to create a user.
type CreateUserRequest struct {
	Header           RequestHeader `json:"header"`
	Email            string        `json:"email" validate:"required,email"`
	Username         string        `json:"username" validate:"required,min=3,max=50"`
	Password         string        `json:"password" validate:"required,min=8"`
	FirstName        string        `json:"first_name" validate:"max=100"`
	LastName         string        `json:"last_name" validate:"max=100"`
	SendWelcomeEmail bool          `json:"send_welcome_email"`
}

// CreateUserResponse represents a response to create a user.
type CreateUserResponse struct {
	Header ResponseHeader `json:"header"`
	User   *models.User   `json:"user,omitempty"`
}

// GetUserRequest represents a request to get a user.
type GetUserRequest struct {
	Header   RequestHeader `json:"header"`
	UserID   *uint         `json:"user_id,omitempty"`
	Email    string        `json:"email,omitempty"`
	Username string        `json:"username,omitempty"`
}

// GetUserResponse represents a response to get a user.
type GetUserResponse struct {
	Header ResponseHeader `json:"header"`
	User   *models.User   `json:"user,omitempty"`
}

// UpdateUserRequest represents a request to update a user.
type UpdateUserRequest struct {
	Header    RequestHeader `json:"header"`
	UserID    uint          `json:"user_id" validate:"required"`
	Email     *string       `json:"email,omitempty" validate:"omitempty,email"`
	Username  *string       `json:"username,omitempty" validate:"omitempty,min=3,max=50"`
	FirstName *string       `json:"first_name,omitempty" validate:"omitempty,max=100"`
	LastName  *string       `json:"last_name,omitempty" validate:"omitempty,max=100"`
	IsActive  *bool         `json:"is_active,omitempty"`
}

// UpdateUserResponse represents a response to update a user.
type UpdateUserResponse struct {
	Header ResponseHeader `json:"header"`
	User   *models.User   `json:"user,omitempty"`
}

// DeleteUserRequest represents a request to delete a user.
type DeleteUserRequest struct {
	Header RequestHeader `json:"header"`
	UserID uint          `json:"user_id" validate:"required"`
}

// DeleteUserResponse represents a response to delete a user.
type DeleteUserResponse struct {
	Header ResponseHeader `json:"header"`
}

// ListUsersRequest represents a request to list users.
type ListUsersRequest struct {
	Header            RequestHeader `json:"header"`
	Offset            int           `json:"offset" validate:"min=0"`
	Limit             int           `json:"limit" validate:"min=1,max=100"`
	IncludeInactive   bool          `json:"include_inactive"`
	IncludeUnverified bool          `json:"include_unverified"`
	SortBy            string        `json:"sort_by" validate:"omitempty,oneof=id email username created_at updated_at"`
	SortOrder         string        `json:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// ListUsersResponse represents a response to list users.
type ListUsersResponse struct {
	Header ResponseHeader `json:"header"`
	Users  []*models.User `json:"users,omitempty"`
	Total  int64          `json:"total"`
	Offset int            `json:"offset"`
	Limit  int            `json:"limit"`
}

// Organization management types

// CreateOrganizationRequest represents a request to create an organization.
type CreateOrganizationRequest struct {
	Header      RequestHeader `json:"header"`
	Name        string        `json:"name" validate:"required,min=1,max=255"`
	Slug        string        `json:"slug" validate:"omitempty,min=1,max=100,alphanum"`
	Description string        `json:"description" validate:"max=1000"`
	IsActive    *bool         `json:"is_active,omitempty"`
}

// CreateOrganizationResponse represents a response to create an organization.
type CreateOrganizationResponse struct {
	Header       ResponseHeader       `json:"header"`
	Organization *models.Organization `json:"organization,omitempty"`
}

// GetOrganizationRequest represents a request to get an organization.
type GetOrganizationRequest struct {
	Header RequestHeader `json:"header"`
	OrgID  *uint         `json:"org_id,omitempty"`
	Slug   string        `json:"slug,omitempty"`
}

// GetOrganizationResponse represents a response to get an organization.
type GetOrganizationResponse struct {
	Header       ResponseHeader       `json:"header"`
	Organization *models.Organization `json:"organization,omitempty"`
}

// UpdateOrganizationRequest represents a request to update an organization.
type UpdateOrganizationRequest struct {
	Header      RequestHeader `json:"header"`
	OrgID       uint          `json:"org_id" validate:"required"`
	Name        *string       `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Slug        *string       `json:"slug,omitempty" validate:"omitempty,min=1,max=100,alphanum"`
	Description *string       `json:"description,omitempty" validate:"omitempty,max=1000"`
	IsActive    *bool         `json:"is_active,omitempty"`
}

// UpdateOrganizationResponse represents a response to update an organization.
type UpdateOrganizationResponse struct {
	Header       ResponseHeader       `json:"header"`
	Organization *models.Organization `json:"organization,omitempty"`
}

// DeleteOrganizationRequest represents a request to delete an organization.
type DeleteOrganizationRequest struct {
	Header RequestHeader `json:"header"`
	OrgID  uint          `json:"org_id" validate:"required"`
}

// DeleteOrganizationResponse represents a response to delete an organization.
type DeleteOrganizationResponse struct {
	Header ResponseHeader `json:"header"`
}

// ListOrganizationsRequest represents a request to list organizations.
type ListOrganizationsRequest struct {
	Header          RequestHeader `json:"header"`
	Offset          int           `json:"offset" validate:"min=0"`
	Limit           int           `json:"limit" validate:"min=1,max=100"`
	IncludeInactive bool          `json:"include_inactive"`
	SortBy          string        `json:"sort_by" validate:"omitempty,oneof=id name slug created_at updated_at"`
	SortOrder       string        `json:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// ListOrganizationsResponse represents a response to list organizations.
type ListOrganizationsResponse struct {
	Header        ResponseHeader         `json:"header"`
	Organizations []*models.Organization `json:"organizations,omitempty"`
	Total         int64                  `json:"total"`
	Offset        int                    `json:"offset"`
	Limit         int                    `json:"limit"`
}

// Role management types

// CreateRoleRequest represents a request to create a role.
type CreateRoleRequest struct {
	Header      RequestHeader    `json:"header"`
	Name        string           `json:"name" validate:"required,min=1,max=100"`
	Description string           `json:"description" validate:"max=500"`
	Permissions []string         `json:"permissions" validate:"required,min=1"`
	Scope       models.RoleScope `json:"scope" validate:"required,oneof=global organization"`
}

// CreateRoleResponse represents a response to create a role.
type CreateRoleResponse struct {
	Header ResponseHeader `json:"header"`
	Role   *models.Role   `json:"role,omitempty"`
}

// GetRoleRequest represents a request to get a role.
type GetRoleRequest struct {
	Header RequestHeader `json:"header"`
	RoleID *uint         `json:"role_id,omitempty"`
	Name   string        `json:"name,omitempty"`
}

// GetRoleResponse represents a response to get a role.
type GetRoleResponse struct {
	Header ResponseHeader `json:"header"`
	Role   *models.Role   `json:"role,omitempty"`
}

// UpdateRoleRequest represents a request to update a role.
type UpdateRoleRequest struct {
	Header      RequestHeader     `json:"header"`
	RoleID      uint              `json:"role_id" validate:"required"`
	Name        *string           `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description *string           `json:"description,omitempty" validate:"omitempty,max=500"`
	Permissions []string          `json:"permissions,omitempty" validate:"omitempty,min=1"`
	Scope       *models.RoleScope `json:"scope,omitempty" validate:"omitempty,oneof=global organization"`
}

// UpdateRoleResponse represents a response to update a role.
type UpdateRoleResponse struct {
	Header ResponseHeader `json:"header"`
	Role   *models.Role   `json:"role,omitempty"`
}

// DeleteRoleRequest represents a request to delete a role.
type DeleteRoleRequest struct {
	Header RequestHeader `json:"header"`
	RoleID uint          `json:"role_id" validate:"required"`
}

// DeleteRoleResponse represents a response to delete a role.
type DeleteRoleResponse struct {
	Header ResponseHeader `json:"header"`
}

// ListRolesRequest represents a request to list roles.
type ListRolesRequest struct {
	Header    RequestHeader     `json:"header"`
	Offset    int               `json:"offset" validate:"min=0"`
	Limit     int               `json:"limit" validate:"min=1,max=100"`
	Scope     *models.RoleScope `json:"scope,omitempty" validate:"omitempty,oneof=global organization"`
	SortBy    string            `json:"sort_by" validate:"omitempty,oneof=id name scope created_at updated_at"`
	SortOrder string            `json:"sort_order" validate:"omitempty,oneof=asc desc"`
}

// ListRolesResponse represents a response to list roles.
type ListRolesResponse struct {
	Header ResponseHeader `json:"header"`
	Roles  []*models.Role `json:"roles,omitempty"`
	Total  int64          `json:"total"`
	Offset int            `json:"offset"`
	Limit  int            `json:"limit"`
}

// Permission management types

// CheckPermissionRequest represents a request to check a permission.
type CheckPermissionRequest struct {
	Header     RequestHeader `json:"header"`
	UserID     uint          `json:"user_id" validate:"required"`
	Permission string        `json:"permission" validate:"required"`
	OrgID      *uint         `json:"org_id,omitempty"`
}

// CheckPermissionResponse represents a response to check a permission.
type CheckPermissionResponse struct {
	Header  ResponseHeader `json:"header"`
	Allowed bool           `json:"allowed"`
	Reason  string         `json:"reason,omitempty"`
}

// AssignRoleRequest represents a request to assign a role to a user.
type AssignRoleRequest struct {
	Header RequestHeader `json:"header"`
	UserID uint          `json:"user_id" validate:"required"`
	RoleID uint          `json:"role_id" validate:"required"`
	OrgID  *uint         `json:"org_id,omitempty"`
}

// AssignRoleResponse represents a response to assign a role.
type AssignRoleResponse struct {
	Header ResponseHeader `json:"header"`
}

// UnassignRoleRequest represents a request to unassign a role from a user.
type UnassignRoleRequest struct {
	Header RequestHeader `json:"header"`
	UserID uint          `json:"user_id" validate:"required"`
	RoleID uint          `json:"role_id" validate:"required"`
	OrgID  *uint         `json:"org_id,omitempty"`
}

// UnassignRoleResponse represents a response to unassign a role.
type UnassignRoleResponse struct {
	Header ResponseHeader `json:"header"`
}
