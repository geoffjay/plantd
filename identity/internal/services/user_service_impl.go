package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	log "github.com/sirupsen/logrus"

	"github.com/geoffjay/plantd/identity/internal/models"
	"github.com/geoffjay/plantd/identity/internal/repositories"
)

// userServiceImpl implements the UserService interface.
type userServiceImpl struct {
	userRepo  repositories.UserRepository
	roleRepo  repositories.RoleRepository
	orgRepo   repositories.OrganizationRepository
	validator *validator.Validate
}

// NewUserService creates a new UserService implementation.
func NewUserService(
	userRepo repositories.UserRepository,
	roleRepo repositories.RoleRepository,
	orgRepo repositories.OrganizationRepository,
) UserService {
	return &userServiceImpl{
		userRepo:  userRepo,
		roleRepo:  roleRepo,
		orgRepo:   orgRepo,
		validator: validator.New(),
	}
}

// CreateUser creates a new user with validation and business rules.
func (s *userServiceImpl) CreateUser(ctx context.Context, req *CreateUserRequest) (*models.User, error) {
	logger := log.WithFields(log.Fields{
		"service":  "user_service",
		"method":   "CreateUser",
		"email":    req.Email,
		"username": req.Username,
	})

	// Validate request
	if err := s.validator.Struct(req); err != nil {
		logger.WithError(err).Error("validation failed")
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check for duplicate email
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		logger.WithError(err).Error("failed to check email uniqueness")
		return nil, fmt.Errorf("failed to check email uniqueness: %w", err)
	}
	if existingUser != nil {
		logger.Error("email already exists")
		return nil, errors.New("email already exists")
	}

	// Check for duplicate username
	existingUser, err = s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		logger.WithError(err).Error("failed to check username uniqueness")
		return nil, fmt.Errorf("failed to check username uniqueness: %w", err)
	}
	if existingUser != nil {
		logger.Error("username already exists")
		return nil, errors.New("username already exists")
	}

	// Create user model
	user := &models.User{
		Email:          strings.ToLower(req.Email),
		Username:       req.Username,
		HashedPassword: req.Password, // TODO: Hash password when auth service is implemented
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		IsActive:       true,
		EmailVerified:  false,
	}

	// Create user in database
	if err := s.userRepo.Create(ctx, user); err != nil {
		logger.WithError(err).Error("failed to create user")
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	logger.WithField("user_id", user.ID).Info("user created successfully")
	return user, nil
}

// GetUserByID retrieves a user by ID.
func (s *userServiceImpl) GetUserByID(ctx context.Context, id uint) (*models.User, error) {
	logger := log.WithFields(log.Fields{
		"service": "user_service",
		"method":  "GetUserByID",
		"user_id": id,
	})

	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		logger.WithError(err).Error("failed to get user")
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		logger.Debug("user not found")
		return nil, errors.New("user not found")
	}

	return user, nil
}

// GetUserByEmail retrieves a user by email.
func (s *userServiceImpl) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	logger := log.WithFields(log.Fields{
		"service": "user_service",
		"method":  "GetUserByEmail",
		"email":   email,
	})

	if email == "" {
		return nil, errors.New("email cannot be empty")
	}

	user, err := s.userRepo.GetByEmail(ctx, strings.ToLower(email))
	if err != nil {
		logger.WithError(err).Error("failed to get user")
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		logger.Debug("user not found")
		return nil, errors.New("user not found")
	}

	return user, nil
}

// GetUserByUsername retrieves a user by username.
func (s *userServiceImpl) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	logger := log.WithFields(log.Fields{
		"service":  "user_service",
		"method":   "GetUserByUsername",
		"username": username,
	})

	if username == "" {
		return nil, errors.New("username cannot be empty")
	}

	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		logger.WithError(err).Error("failed to get user")
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		logger.Debug("user not found")
		return nil, errors.New("user not found")
	}

	return user, nil
}

// UpdateUser updates a user with validation and business rules.
func (s *userServiceImpl) UpdateUser(ctx context.Context, id uint, req *UpdateUserRequest) (*models.User, error) {
	logger := createServiceLogger("user_service", "UpdateUser", log.Fields{"user_id": id})

	// Validate request
	if err := s.validator.Struct(req); err != nil {
		return nil, logAndError(logger, "validation failed", err)
	}

	// Get existing user
	user, err := s.getExistingUser(ctx, id, logger)
	if err != nil {
		return nil, err
	}

	// Update user fields with validation
	if err := s.updateUserFields(ctx, req, user, logger); err != nil {
		return nil, err
	}

	// Save updated user
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, logAndError(logger, "failed to update user", err)
	}

	logSuccess(logger, "user updated successfully", nil)
	return user, nil
}

// getExistingUser retrieves and validates an existing user.
func (s *userServiceImpl) getExistingUser(ctx context.Context, id uint, logger *log.Entry) (*models.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, logAndError(logger, "failed to get user", err)
	}
	if user == nil {
		return nil, logAndErrorSimple(logger, "user "+ErrNotFound)
	}
	return user, nil
}

// updateUserFields updates user fields with validation.
func (s *userServiceImpl) updateUserFields(ctx context.Context, req *UpdateUserRequest, user *models.User, logger *log.Entry) error {
	// Check email uniqueness if email is being updated
	if req.Email != nil && strings.ToLower(*req.Email) != user.Email {
		if err := s.validateEmailUniqueness(ctx, *req.Email, logger); err != nil {
			return err
		}
		user.Email = strings.ToLower(*req.Email)
	}

	// Check username uniqueness if username is being updated
	if req.Username != nil && *req.Username != user.Username {
		if err := s.validateUsernameUniqueness(ctx, *req.Username, logger); err != nil {
			return err
		}
		user.Username = *req.Username
	}

	// Update other fields
	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		user.LastName = *req.LastName
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	return nil
}

// validateEmailUniqueness checks if email is unique.
func (s *userServiceImpl) validateEmailUniqueness(ctx context.Context, email string, logger *log.Entry) error {
	existingUser, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return logAndError(logger, "failed to check email uniqueness", err)
	}
	if existingUser != nil {
		return logAndErrorSimple(logger, FieldEmail+" "+ErrAlreadyExists)
	}
	return nil
}

// validateUsernameUniqueness checks if username is unique.
func (s *userServiceImpl) validateUsernameUniqueness(ctx context.Context, username string, logger *log.Entry) error {
	existingUser, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return logAndError(logger, "failed to check username uniqueness", err)
	}
	if existingUser != nil {
		return logAndErrorSimple(logger, FieldUsername+" "+ErrAlreadyExists)
	}
	return nil
}

// DeleteUser soft deletes a user.
func (s *userServiceImpl) DeleteUser(ctx context.Context, id uint) error {
	logger := log.WithFields(log.Fields{
		"service": "user_service",
		"method":  "DeleteUser",
		"user_id": id,
	})

	// Check if user exists
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		logger.WithError(err).Error("failed to get user")
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		logger.Error("user not found")
		return errors.New("user not found")
	}

	// Delete user
	if err := s.userRepo.Delete(ctx, id); err != nil {
		logger.WithError(err).Error("failed to delete user")
		return fmt.Errorf("failed to delete user: %w", err)
	}

	logger.Info("user deleted successfully")
	return nil
}

// ListUsers lists users with pagination and filtering.
func (s *userServiceImpl) ListUsers(ctx context.Context, req *ListUsersRequest) ([]*models.User, error) {
	logger := log.WithFields(log.Fields{
		"service": "user_service",
		"method":  "ListUsers",
		"offset":  req.Offset,
		"limit":   req.Limit,
	})

	// Validate request
	if err := s.validator.Struct(req); err != nil {
		logger.WithError(err).Error("validation failed")
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// For now, use basic list without complex filtering
	users, err := s.userRepo.List(ctx, req.Offset, req.Limit)
	if err != nil {
		logger.WithError(err).Error("failed to list users")
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	logger.WithField("count", len(users)).Debug("users listed successfully")
	return users, nil
}

// SearchUsers searches users by query and fields.
func (s *userServiceImpl) SearchUsers(ctx context.Context, req *SearchUsersRequest) ([]*models.User, error) {
	logger := log.WithFields(log.Fields{
		"service": "user_service",
		"method":  "SearchUsers",
		"query":   req.Query,
		"fields":  req.SearchFields,
	})

	// Validate request
	if err := s.validator.Struct(req); err != nil {
		logger.WithError(err).Error("validation failed")
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Simple implementation using repository search methods
	var users []*models.User
	var err error

	// Search by specified fields
	for _, field := range req.SearchFields {
		switch field {
		case FieldEmail:
			users, err = s.userRepo.SearchByEmail(ctx, req.Query, req.Offset, req.Limit)
		case FieldUsername:
			users, err = s.userRepo.SearchByUsername(ctx, req.Query, req.Offset, req.Limit)
		case FieldName:
			users, err = s.userRepo.SearchByName(ctx, req.Query, req.Offset, req.Limit)
		}
		if err != nil {
			logger.WithError(err).Error("search failed")
			return nil, fmt.Errorf("search failed: %w", err)
		}
		if len(users) > 0 {
			break // Return first successful search
		}
	}

	logger.WithField("count", len(users)).Debug("users searched successfully")
	return users, nil
}

// CountUsers returns the total number of users.
func (s *userServiceImpl) CountUsers(ctx context.Context) (int64, error) {
	logger := log.WithFields(log.Fields{
		"service": "user_service",
		"method":  "CountUsers",
	})

	count, err := s.userRepo.Count(ctx)
	if err != nil {
		logger.WithError(err).Error("failed to count users")
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	logger.WithField("count", count).Debug("users counted successfully")
	return count, nil
}

// ActivateUser activates a user.
func (s *userServiceImpl) ActivateUser(ctx context.Context, id uint) error {
	logger := log.WithFields(log.Fields{
		"service": "user_service",
		"method":  "ActivateUser",
		"user_id": id,
	})

	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		logger.WithError(err).Error("failed to get user")
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		logger.Error("user not found")
		return errors.New("user not found")
	}

	if user.IsActive {
		logger.Debug("user already active")
		return nil
	}

	user.IsActive = true
	if err := s.userRepo.Update(ctx, user); err != nil {
		logger.WithError(err).Error("failed to activate user")
		return fmt.Errorf("failed to activate user: %w", err)
	}

	logger.Info("user activated successfully")
	return nil
}

// DeactivateUser deactivates a user.
func (s *userServiceImpl) DeactivateUser(ctx context.Context, id uint) error {
	logger := log.WithFields(log.Fields{
		"service": "user_service",
		"method":  "DeactivateUser",
		"user_id": id,
	})

	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		logger.WithError(err).Error("failed to get user")
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		logger.Error("user not found")
		return errors.New("user not found")
	}

	if !user.IsActive {
		logger.Debug("user already inactive")
		return nil
	}

	user.IsActive = false
	if err := s.userRepo.Update(ctx, user); err != nil {
		logger.WithError(err).Error("failed to deactivate user")
		return fmt.Errorf("failed to deactivate user: %w", err)
	}

	logger.Info("user deactivated successfully")
	return nil
}

// VerifyUserEmail marks a user's email as verified.
func (s *userServiceImpl) VerifyUserEmail(ctx context.Context, id uint) error {
	logger := log.WithFields(log.Fields{
		"service": "user_service",
		"method":  "VerifyUserEmail",
		"user_id": id,
	})

	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		logger.WithError(err).Error("failed to get user")
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		logger.Error("user not found")
		return errors.New("user not found")
	}

	if user.EmailVerified {
		logger.Debug("email already verified")
		return nil
	}

	user.MarkEmailAsVerified()
	if err := s.userRepo.Update(ctx, user); err != nil {
		logger.WithError(err).Error("failed to verify email")
		return fmt.Errorf("failed to verify email: %w", err)
	}

	logger.Info("email verified successfully")
	return nil
}

// GetActiveUsers returns active users with pagination.
func (s *userServiceImpl) GetActiveUsers(ctx context.Context, offset, limit int) ([]*models.User, error) {
	logger := log.WithFields(log.Fields{
		"service": "user_service",
		"method":  "GetActiveUsers",
		"offset":  offset,
		"limit":   limit,
	})

	users, err := s.userRepo.GetActiveUsers(ctx, offset, limit)
	if err != nil {
		logger.WithError(err).Error("failed to get active users")
		return nil, fmt.Errorf("failed to get active users: %w", err)
	}

	logger.WithField("count", len(users)).Debug("active users retrieved successfully")
	return users, nil
}

// GetVerifiedUsers returns verified users with pagination.
func (s *userServiceImpl) GetVerifiedUsers(ctx context.Context, offset, limit int) ([]*models.User, error) {
	logger := log.WithFields(log.Fields{
		"service": "user_service",
		"method":  "GetVerifiedUsers",
		"offset":  offset,
		"limit":   limit,
	})

	users, err := s.userRepo.GetVerifiedUsers(ctx, offset, limit)
	if err != nil {
		logger.WithError(err).Error("failed to get verified users")
		return nil, fmt.Errorf("failed to get verified users: %w", err)
	}

	logger.WithField("count", len(users)).Debug("verified users retrieved successfully")
	return users, nil
}

// AssignUserToRole assigns a user to a role.
func (s *userServiceImpl) AssignUserToRole(_ context.Context, _, _ uint) error { //nolint:revive
	// TODO: Implement when role assignment methods are added to repositories
	return errors.New("not implemented yet")
}

// RemoveUserFromRole removes a user from a role.
func (s *userServiceImpl) RemoveUserFromRole(_ context.Context, _, _ uint) error { //nolint:revive
	// TODO: Implement when role assignment methods are added to repositories
	return errors.New("not implemented yet")
}

// GetUserRoles returns the roles assigned to a user.
func (s *userServiceImpl) GetUserRoles(ctx context.Context, userID uint) ([]*models.Role, error) { //nolint:revive
	logger := log.WithFields(log.Fields{
		"service": "user_service",
		"method":  "GetUserRoles",
		"user_id": userID,
	})

	if userID == 0 {
		logger.Error("userID must be provided")
		return nil, errors.New("userID must be provided")
	}

	// Use limit -1 to disable LIMIT clause and return all roles for the user
	roles, err := s.roleRepo.GetByUser(ctx, userID, 0, -1)
	if err != nil {
		logger.WithError(err).Error("failed to get user roles")
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	logger.WithField("count", len(roles)).Debug("user roles retrieved successfully")
	return roles, nil
}

// AssignUserToOrganization assigns a user to an organization.
func (s *userServiceImpl) AssignUserToOrganization(_ context.Context, _, _ uint) error { //nolint:revive
	// TODO: Implement when organization membership methods are added to repositories
	return errors.New("not implemented yet")
}

// RemoveUserFromOrganization removes a user from an organization.
func (s *userServiceImpl) RemoveUserFromOrganization(_ context.Context, _, _ uint) error { //nolint:revive
	// TODO: Implement when organization membership methods are added to repositories
	return errors.New("not implemented yet")
}

// GetUserOrganizations returns the organizations a user belongs to.
func (s *userServiceImpl) GetUserOrganizations(_ context.Context, _ uint) ([]*models.Organization, error) { //nolint:revive
	// TODO: Implement when organization membership methods are added to repositories
	return nil, errors.New("not implemented yet")
}
