package auth

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/geoffjay/plantd/identity/internal/models"
	"github.com/geoffjay/plantd/identity/internal/repositories"
)

// OrganizationMembershipService handles organization membership and permissions.
type OrganizationMembershipService struct {
	userRepo         repositories.UserRepository
	organizationRepo repositories.OrganizationRepository
	roleRepo         repositories.RoleRepository
	rbacService      *RBACService
	logger           *slog.Logger
	auditLogger      *slog.Logger
}

// MembershipStatus represents the status of organization membership.
type MembershipStatus string

const (
	// MembershipStatusActive represents an active membership.
	MembershipStatusActive MembershipStatus = "active"
	// MembershipStatusPending represents a pending membership.
	MembershipStatusPending MembershipStatus = "pending"
	// MembershipStatusSuspended represents a suspended membership.
	MembershipStatusSuspended MembershipStatus = "suspended"
	// MembershipStatusRevoked represents a revoked membership.
	MembershipStatusRevoked MembershipStatus = "revoked"
)

// OrganizationMember represents a user's membership in an organization.
type OrganizationMember struct {
	UserID         uint             `json:"user_id"`
	OrganizationID uint             `json:"organization_id"`
	Status         MembershipStatus `json:"status"`
	JoinedAt       time.Time        `json:"joined_at"`
	LastActiveAt   *time.Time       `json:"last_active_at"`
	Roles          []models.Role    `json:"roles"`
	Permissions    []Permission     `json:"permissions"`

	// User details
	User *models.User `json:"user,omitempty"`

	// Organization details
	Organization *models.Organization `json:"organization,omitempty"`
}

// NewOrganizationMembershipService creates a new organization membership service.
func NewOrganizationMembershipService(
	userRepo repositories.UserRepository,
	organizationRepo repositories.OrganizationRepository,
	roleRepo repositories.RoleRepository,
	rbacService *RBACService,
	logger *slog.Logger,
	auditLogger *slog.Logger,
) *OrganizationMembershipService {
	return &OrganizationMembershipService{
		userRepo:         userRepo,
		organizationRepo: organizationRepo,
		roleRepo:         roleRepo,
		rbacService:      rbacService,
		logger:           logger,
		auditLogger:      auditLogger,
	}
}

// AddUserToOrganization adds a user to an organization.
func (oms *OrganizationMembershipService) AddUserToOrganization(ctx context.Context, userID, organizationID uint, adminUserID uint) error {
	oms.logger.Info("Adding user to organization",
		"user_id", userID,
		"organization_id", organizationID,
		"admin_user_id", adminUserID)

	// Verify admin has permission to add members
	hasPermission, err := oms.rbacService.HasPermission(ctx, adminUserID, PermissionOrganizationMemberAdd, &organizationID)
	if err != nil {
		return fmt.Errorf("failed to check admin permissions: %w", err)
	}
	if !hasPermission {
		return &PermissionError{
			UserID:     adminUserID,
			Permission: PermissionOrganizationMemberAdd,
			OrgID:      &organizationID,
			Message:    "insufficient permissions to add organization members",
		}
	}

	// Verify user exists
	user, err := oms.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Verify organization exists
	organization, err := oms.organizationRepo.GetByID(ctx, organizationID)
	if err != nil {
		return fmt.Errorf("failed to get organization: %w", err)
	}

	// Check if user is already a member
	isMember, err := oms.IsUserMember(ctx, userID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to check membership: %w", err)
	}
	if isMember {
		return fmt.Errorf("user %d is already a member of organization %d", userID, organizationID)
	}

	// Add user to organization
	err = oms.userRepo.AddToOrganization(ctx, userID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to add user to organization: %w", err)
	}

	// Log the membership change
	oms.auditLogger.Info("User added to organization",
		"user_id", userID,
		"user_email", user.Email,
		"organization_id", organizationID,
		"organization_name", organization.Name,
		"admin_user_id", adminUserID)

	return nil
}

// RemoveUserFromOrganization removes a user from an organization.
func (oms *OrganizationMembershipService) RemoveUserFromOrganization(ctx context.Context, userID, organizationID uint, adminUserID uint) error {
	oms.logger.Info("Removing user from organization",
		"user_id", userID,
		"organization_id", organizationID,
		"admin_user_id", adminUserID)

	// Verify admin has permission to remove members
	hasPermission, err := oms.rbacService.HasPermission(ctx, adminUserID, PermissionOrganizationMemberRemove, &organizationID)
	if err != nil {
		return fmt.Errorf("failed to check admin permissions: %w", err)
	}
	if !hasPermission {
		return &PermissionError{
			UserID:     adminUserID,
			Permission: PermissionOrganizationMemberRemove,
			OrgID:      &organizationID,
			Message:    "insufficient permissions to remove organization members",
		}
	}

	// Verify user is a member
	isMember, err := oms.IsUserMember(ctx, userID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to check membership: %w", err)
	}
	if !isMember {
		return fmt.Errorf("user %d is not a member of organization %d", userID, organizationID)
	}

	// Get user and organization for audit log
	user, _ := oms.userRepo.GetByID(ctx, userID)
	organization, _ := oms.organizationRepo.GetByID(ctx, organizationID)

	// Remove all organization-specific roles first
	orgRoles, err := oms.roleRepo.GetByUserAndOrganization(ctx, userID, organizationID, 0, 1000)
	if err != nil {
		oms.logger.Warn(
			"Failed to get user organization roles",
			"error", err,
			"user_id", userID,
			"organization_id", organizationID,
		)
	} else {
		for _, role := range orgRoles {
			if role != nil {
				err = oms.rbacService.RemoveRoleFromUser(ctx, userID, role.ID, &organizationID)
				if err != nil {
					oms.logger.Warn(
						"Failed to remove organization role",
						"error", err,
						"user_id", userID,
						"role_id", role.ID,
						"organization_id", organizationID,
					)
				}
			}
		}
	}

	// Remove user from organization
	err = oms.userRepo.RemoveFromOrganization(ctx, userID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to remove user from organization: %w", err)
	}

	// Log the membership change
	oms.auditLogger.Info("User removed from organization",
		"user_id", userID,
		"user_email", func() string {
			if user != nil {
				return user.Email
			}
			return ""
		}(),
		"organization_id", organizationID,
		"organization_name", func() string {
			if organization != nil {
				return organization.Name
			}
			return ""
		}(),
		"admin_user_id", adminUserID)

	return nil
}

// IsUserMember checks if a user is a member of an organization.
func (oms *OrganizationMembershipService) IsUserMember(ctx context.Context, userID, organizationID uint) (bool, error) {
	user, err := oms.userRepo.GetWithOrganizations(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get user with organizations: %w", err)
	}

	for _, org := range user.Organizations {
		if org.ID == organizationID {
			return true, nil
		}
	}

	return false, nil
}

// GetOrganizationMembers returns all members of an organization.
func (oms *OrganizationMembershipService) GetOrganizationMembers(
	ctx context.Context,
	organizationID uint,
	requesterUserID uint,
	offset, limit int,
) ([]OrganizationMember, error) {
	// Verify requester has permission to list members
	hasPermission, err := oms.rbacService.HasPermission(ctx, requesterUserID, PermissionOrganizationMemberList, &organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to check permissions: %w", err)
	}
	if !hasPermission {
		return nil, &PermissionError{
			UserID:     requesterUserID,
			Permission: PermissionOrganizationMemberList,
			OrgID:      &organizationID,
			Message:    "insufficient permissions to list organization members",
		}
	}

	// Get organization members
	users, err := oms.userRepo.GetByOrganization(ctx, organizationID, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization members: %w", err)
	}

	var members []OrganizationMember
	for _, user := range users {
		if user == nil {
			continue
		}

		// Get user's roles in this organization
		roles, err := oms.rbacService.GetUserRoles(ctx, user.ID, &organizationID)
		if err != nil {
			oms.logger.Warn(
				"Failed to get user roles",
				"error", err,
				"user_id", user.ID,
				"organization_id", organizationID,
			)
			roles = []models.Role{}
		}

		// Get user's permissions in this organization
		permissions, err := oms.rbacService.GetUserPermissions(ctx, user.ID, &organizationID)
		if err != nil {
			oms.logger.Warn(
				"Failed to get user permissions",
				"error", err,
				"user_id", user.ID,
				"organization_id", organizationID,
			)
			permissions = []Permission{}
		}

		member := OrganizationMember{
			UserID:         user.ID,
			OrganizationID: organizationID,
			Status:         MembershipStatusActive, // Default status
			JoinedAt:       user.CreatedAt,         // Approximate join date
			Roles:          roles,
			Permissions:    permissions,
			User:           user,
		}

		members = append(members, member)
	}

	return members, nil
}

// GetUserOrganizations returns all organizations a user is a member of.
func (oms *OrganizationMembershipService) GetUserOrganizations(ctx context.Context, userID uint, requesterUserID uint) ([]OrganizationMember, error) {
	// Users can view their own organizations, or admin can view any user's organizations
	if userID != requesterUserID {
		hasPermission, err := oms.rbacService.HasPermission(ctx, requesterUserID, PermissionUserRead, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to check permissions: %w", err)
		}
		if !hasPermission {
			return nil, &PermissionError{
				UserID:     requesterUserID,
				Permission: PermissionUserRead,
				Message:    "insufficient permissions to view user organizations",
			}
		}
	}

	user, err := oms.userRepo.GetWithOrganizations(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user with organizations: %w", err)
	}

	var memberships []OrganizationMember
	for _, org := range user.Organizations {
		// Get user's roles in this organization
		roles, err := oms.rbacService.GetUserRoles(ctx, userID, &org.ID)
		if err != nil {
			oms.logger.Warn(
				"Failed to get user roles",
				"error", err,
				"user_id", userID,
				"organization_id", org.ID,
			)
			roles = []models.Role{}
		}

		// Get user's permissions in this organization
		permissions, err := oms.rbacService.GetUserPermissions(ctx, userID, &org.ID)
		if err != nil {
			oms.logger.Warn(
				"Failed to get user permissions",
				"error", err,
				"user_id", userID,
				"organization_id", org.ID,
			)
			permissions = []Permission{}
		}

		membership := OrganizationMember{
			UserID:         userID,
			OrganizationID: org.ID,
			Status:         MembershipStatusActive,
			JoinedAt:       user.CreatedAt, // Approximate join date
			Roles:          roles,
			Permissions:    permissions,
			Organization:   &org,
		}

		memberships = append(memberships, membership)
	}

	return memberships, nil
}

// AssignOrganizationRole assigns a role to a user within an organization context.
func (oms *OrganizationMembershipService) AssignOrganizationRole(ctx context.Context, userID, roleID, organizationID uint, adminUserID uint) error {
	oms.logger.Info(
		"Assigning organization role",
		"user_id", userID,
		"role_id", roleID,
		"organization_id", organizationID,
		"admin_user_id", adminUserID,
	)

	// Verify admin has permission to assign roles
	hasPermission, err := oms.rbacService.HasPermission(ctx, adminUserID, PermissionRoleAssign, &organizationID)
	if err != nil {
		return fmt.Errorf("failed to check admin permissions: %w", err)
	}
	if !hasPermission {
		return &PermissionError{
			UserID:     adminUserID,
			Permission: PermissionRoleAssign,
			OrgID:      &organizationID,
			Message:    "insufficient permissions to assign roles",
		}
	}

	// Verify user is a member of the organization
	isMember, err := oms.IsUserMember(ctx, userID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to check membership: %w", err)
	}
	if !isMember {
		return fmt.Errorf("user %d is not a member of organization %d", userID, organizationID)
	}

	// Assign the role
	err = oms.rbacService.AssignRoleToUser(ctx, userID, roleID, &organizationID)
	if err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	// Get details for audit log
	user, _ := oms.userRepo.GetByID(ctx, userID)
	role, _ := oms.roleRepo.GetByID(ctx, roleID)
	organization, _ := oms.organizationRepo.GetByID(ctx, organizationID)

	oms.auditLogger.Info("Organization role assigned",
		"user_id", userID,
		"user_email", func() string {
			if user != nil {
				return user.Email
			}
			return ""
		}(),
		"role_id", roleID,
		"role_name", func() string {
			if role != nil {
				return role.Name
			}
			return ""
		}(),
		"organization_id", organizationID,
		"organization_name", func() string {
			if organization != nil {
				return organization.Name
			}
			return ""
		}(),
		"admin_user_id", adminUserID)

	return nil
}

// RemoveOrganizationRole removes a role from a user within an organization context.
func (oms *OrganizationMembershipService) RemoveOrganizationRole(ctx context.Context, userID, roleID, organizationID uint, adminUserID uint) error {
	oms.logger.Info(
		"Removing organization role",
		"user_id", userID,
		"role_id", roleID,
		"organization_id", organizationID,
		"admin_user_id", adminUserID,
	)

	// Verify admin has permission to unassign roles
	hasPermission, err := oms.rbacService.HasPermission(ctx, adminUserID, PermissionRoleUnassign, &organizationID)
	if err != nil {
		return fmt.Errorf("failed to check admin permissions: %w", err)
	}
	if !hasPermission {
		return &PermissionError{
			UserID:     adminUserID,
			Permission: PermissionRoleUnassign,
			OrgID:      &organizationID,
			Message:    "insufficient permissions to unassign roles",
		}
	}

	// Remove the role
	err = oms.rbacService.RemoveRoleFromUser(ctx, userID, roleID, &organizationID)
	if err != nil {
		return fmt.Errorf("failed to remove role: %w", err)
	}

	// Get details for audit log
	user, _ := oms.userRepo.GetByID(ctx, userID)
	role, _ := oms.roleRepo.GetByID(ctx, roleID)
	organization, _ := oms.organizationRepo.GetByID(ctx, organizationID)

	oms.auditLogger.Info("Organization role removed",
		"user_id", userID,
		"user_email", func() string {
			if user != nil {
				return user.Email
			}
			return ""
		}(),
		"role_id", roleID,
		"role_name", func() string {
			if role != nil {
				return role.Name
			}
			return ""
		}(),
		"organization_id", organizationID,
		"organization_name", func() string {
			if organization != nil {
				return organization.Name
			}
			return ""
		}(),
		"admin_user_id", adminUserID)

	return nil
}

// SwitchOrganizationContext allows a user to switch their active organization context.
func (oms *OrganizationMembershipService) SwitchOrganizationContext(ctx context.Context, userID, organizationID uint) error {
	// Verify user is a member of the organization
	isMember, err := oms.IsUserMember(ctx, userID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to check membership: %w", err)
	}
	if !isMember {
		return fmt.Errorf("user %d is not a member of organization %d", userID, organizationID)
	}

	// Clear permission cache for this user to ensure fresh permissions are loaded
	oms.rbacService.clearUserPermissionCache(userID, &organizationID)
	oms.rbacService.clearUserPermissionCache(userID, nil) // Clear global cache too

	oms.logger.Info(
		"User switched organization context",
		"user_id", userID,
		"organization_id", organizationID,
	)

	return nil
}

// ValidateOrganizationAccess validates that a user has access to perform an action within an organization.
func (oms *OrganizationMembershipService) ValidateOrganizationAccess(ctx context.Context, userID, organizationID uint, permission Permission) error {
	// Verify user is a member of the organization
	isMember, err := oms.IsUserMember(ctx, userID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to check membership: %w", err)
	}
	if !isMember {
		return &UnauthorizedError{
			UserID:   userID,
			Resource: "organization",
			Action:   "access",
			Message:  fmt.Sprintf("user is not a member of organization %d", organizationID),
		}
	}

	// Check permission within organization context
	hasPermission, err := oms.rbacService.HasPermission(ctx, userID, permission, &organizationID)
	if err != nil {
		return fmt.Errorf("failed to check permission: %w", err)
	}
	if !hasPermission {
		return &PermissionError{
			UserID:     userID,
			Permission: permission,
			OrgID:      &organizationID,
			Message:    "insufficient permissions for organization action",
		}
	}

	return nil
}
