package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/geoffjay/plantd/identity/internal/models"
	"github.com/geoffjay/plantd/identity/internal/repositories"
	"github.com/geoffjay/plantd/identity/internal/services"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

// RegistrationRequest represents a user registration request
type RegistrationRequest struct {
	Email           string `json:"email" validate:"required,email"`
	Username        string `json:"username" validate:"required,min=3,max=50"`
	Password        string `json:"password" validate:"required"`
	FirstName       string `json:"first_name" validate:"max=100"`
	LastName        string `json:"last_name" validate:"max=100"`
	IPAddress       string `json:"ip_address,omitempty"`
	UserAgent       string `json:"user_agent,omitempty"`
	InvitationToken string `json:"invitation_token,omitempty"`
}

// RegistrationResponse represents the response to a successful registration
type RegistrationResponse struct {
	User                 *models.User `json:"user"`
	EmailVerification    string       `json:"email_verification_token,omitempty"`
	RequiresVerification bool         `json:"requires_verification"`
	Message              string       `json:"message"`
}

// EmailVerificationRequest represents an email verification request
type EmailVerificationRequest struct {
	Token     string `json:"token" validate:"required"`
	IPAddress string `json:"ip_address,omitempty"`
}

// ProfileUpdateRequest represents a user profile update request
type ProfileUpdateRequest struct {
	FirstName string `json:"first_name" validate:"max=100"`
	LastName  string `json:"last_name" validate:"max=100"`
	Username  string `json:"username" validate:"min=3,max=50"`
	IPAddress string `json:"ip_address,omitempty"`
}

// PasswordResetRequest represents a password reset request
type PasswordResetRequest struct {
	Email     string `json:"email" validate:"required,email"`
	IPAddress string `json:"ip_address,omitempty"`
}

// PasswordResetConfirmRequest represents a password reset confirmation
type PasswordResetConfirmRequest struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required"`
	IPAddress   string `json:"ip_address,omitempty"`
}

// RegistrationConfig holds configuration for user registration
type RegistrationConfig struct {
	// RequireEmailVerification determines if email verification is required
	RequireEmailVerification bool `json:"require_email_verification" yaml:"require_email_verification"`
	// AllowSelfRegistration determines if users can register themselves
	AllowSelfRegistration bool `json:"allow_self_registration" yaml:"allow_self_registration"`
	// EmailVerificationExpiry is how long email verification tokens are valid
	EmailVerificationExpiry time.Duration `json:"email_verification_expiry" yaml:"email_verification_expiry"`
	// PasswordResetExpiry is how long password reset tokens are valid
	PasswordResetExpiry time.Duration `json:"password_reset_expiry" yaml:"password_reset_expiry"`
	// DefaultUserRole is the default role assigned to new users
	DefaultUserRole string `json:"default_user_role" yaml:"default_user_role"`
}

// DefaultRegistrationConfig returns a secure default registration configuration
func DefaultRegistrationConfig() *RegistrationConfig {
	return &RegistrationConfig{
		RequireEmailVerification: true,
		AllowSelfRegistration:    true,
		EmailVerificationExpiry:  24 * time.Hour,
		PasswordResetExpiry:      2 * time.Hour,
		DefaultUserRole:          "user",
	}
}

// RegistrationService provides user registration and management functionality
type RegistrationService struct {
	config            *RegistrationConfig
	userRepo          repositories.UserRepository
	userService       services.UserService
	passwordValidator *PasswordValidator
	jwtManager        *JWTManager
	rateLimiter       *RateLimiter
	logger            *logrus.Logger
}

// NewRegistrationService creates a new registration service
func NewRegistrationService(
	config *RegistrationConfig,
	userRepo repositories.UserRepository,
	userService services.UserService,
	passwordValidator *PasswordValidator,
	jwtManager *JWTManager,
	rateLimiter *RateLimiter,
	logger *logrus.Logger,
) *RegistrationService {
	if config == nil {
		config = DefaultRegistrationConfig()
	}

	return &RegistrationService{
		config:            config,
		userRepo:          userRepo,
		userService:       userService,
		passwordValidator: passwordValidator,
		jwtManager:        jwtManager,
		rateLimiter:       rateLimiter,
		logger:            logger,
	}
}

// Register creates a new user account
func (rs *RegistrationService) Register(ctx context.Context, req *RegistrationRequest) (*RegistrationResponse, error) {
	// Check if self-registration is allowed
	if !rs.config.AllowSelfRegistration && req.InvitationToken == "" {
		return nil, errors.New("self-registration is not allowed")
	}

	// Check rate limiting
	if req.IPAddress != "" {
		allowed, err := rs.rateLimiter.AllowRequest(req.IPAddress)
		if err != nil || !allowed {
			rs.logSecurityEvent(&SecurityEvent{
				EventType:     "registration_rate_limited",
				Email:         req.Email,
				IPAddress:     req.IPAddress,
				UserAgent:     req.UserAgent,
				Success:       false,
				FailureReason: "rate limit exceeded",
				Timestamp:     time.Now(),
			})
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Validate password strength
	if err := rs.passwordValidator.Validate(req.Password); err != nil {
		return nil, fmt.Errorf("password validation failed: %w", err)
	}

	// Check if email already exists
	existingUser, err := rs.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		rs.logSecurityEvent(&SecurityEvent{
			EventType:     "registration_email_exists",
			Email:         req.Email,
			IPAddress:     req.IPAddress,
			UserAgent:     req.UserAgent,
			Success:       false,
			FailureReason: "email already exists",
			Timestamp:     time.Now(),
		})
		return nil, errors.New("email address is already registered")
	}

	// Check if username already exists
	existingUser, err = rs.userRepo.GetByUsername(ctx, req.Username)
	if err == nil && existingUser != nil {
		rs.logSecurityEvent(&SecurityEvent{
			EventType:     "registration_username_exists",
			Email:         req.Email,
			IPAddress:     req.IPAddress,
			UserAgent:     req.UserAgent,
			Success:       false,
			FailureReason: "username already exists",
			Timestamp:     time.Now(),
		})
		return nil, errors.New("username is already taken")
	}

	// Hash password
	hashedPassword, err := rs.passwordValidator.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &models.User{
		Email:          req.Email,
		Username:       req.Username,
		HashedPassword: hashedPassword,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		IsActive:       !rs.config.RequireEmailVerification, // Active immediately if verification not required
		EmailVerified:  !rs.config.RequireEmailVerification, // Verified immediately if verification not required
	}

	// Create user in database
	if err := rs.userRepo.Create(ctx, user); err != nil {
		rs.logSecurityEvent(&SecurityEvent{
			EventType:     "registration_failed",
			Email:         req.Email,
			IPAddress:     req.IPAddress,
			UserAgent:     req.UserAgent,
			Success:       false,
			FailureReason: "failed to create user",
			Timestamp:     time.Now(),
		})
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate email verification token if required
	var verificationToken string
	if rs.config.RequireEmailVerification {
		verificationToken, err = rs.generateEmailVerificationToken(user.ID, user.Email)
		if err != nil {
			rs.logger.WithError(err).WithField("user_id", user.ID).Error("Failed to generate email verification token")
		}
	}

	rs.logSecurityEvent(&SecurityEvent{
		EventType: "registration_success",
		UserID:    &user.ID,
		Email:     user.Email,
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
		Success:   true,
		Timestamp: time.Now(),
	})

	response := &RegistrationResponse{
		User:                 user,
		RequiresVerification: rs.config.RequireEmailVerification,
	}

	if rs.config.RequireEmailVerification {
		response.EmailVerification = verificationToken
		response.Message = "Registration successful. Please check your email to verify your account."
	} else {
		response.Message = "Registration successful. You can now log in."
	}

	return response, nil
}

// VerifyEmail verifies a user's email address using a verification token
func (rs *RegistrationService) VerifyEmail(ctx context.Context, req *EmailVerificationRequest) error {
	// Validate the verification token
	claims, err := rs.jwtManager.ValidateToken(req.Token, ResetToken)
	if err != nil {
		rs.logSecurityEvent(&SecurityEvent{
			EventType:     "email_verification_invalid_token",
			IPAddress:     req.IPAddress,
			Success:       false,
			FailureReason: "invalid verification token",
			Timestamp:     time.Now(),
		})
		return errors.New("invalid or expired verification token")
	}

	// Get user
	user, err := rs.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Check if email is already verified
	if user.EmailVerified {
		return errors.New("email is already verified")
	}

	// Mark email as verified and activate user
	user.MarkEmailAsVerified()
	user.IsActive = true

	if err := rs.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Revoke the verification token
	if err := rs.jwtManager.RevokeToken(req.Token, ResetToken); err != nil {
		rs.logger.WithError(err).Warn("Failed to revoke verification token")
	}

	rs.logSecurityEvent(&SecurityEvent{
		EventType: "email_verification_success",
		UserID:    &user.ID,
		Email:     user.Email,
		IPAddress: req.IPAddress,
		Success:   true,
		Timestamp: time.Now(),
	})

	return nil
}

// UpdateProfile updates a user's profile information
func (rs *RegistrationService) UpdateProfile(ctx context.Context, userID uint, req *ProfileUpdateRequest) (*models.User, error) {
	// Get existing user
	user, err := rs.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Check if username is being changed and if it's available
	if req.Username != "" && req.Username != user.Username {
		existingUser, err := rs.userRepo.GetByUsername(ctx, req.Username)
		if err == nil && existingUser != nil {
			return nil, errors.New("username is already taken")
		}
		user.Username = req.Username
	}

	// Update profile fields
	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}
	if req.LastName != "" {
		user.LastName = req.LastName
	}

	user.UpdatedAt = time.Now()

	// Save changes
	if err := rs.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	rs.logSecurityEvent(&SecurityEvent{
		EventType: "profile_update_success",
		UserID:    &userID,
		Email:     user.Email,
		IPAddress: req.IPAddress,
		Success:   true,
		Timestamp: time.Now(),
	})

	return user, nil
}

// InitiatePasswordReset initiates a password reset process
func (rs *RegistrationService) InitiatePasswordReset(ctx context.Context, req *PasswordResetRequest) error {
	// Check rate limiting
	if req.IPAddress != "" {
		allowed, err := rs.rateLimiter.AllowRequest(req.IPAddress)
		if err != nil || !allowed {
			return fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Get user by email
	user, err := rs.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		// Don't reveal if email exists or not for security
		rs.logSecurityEvent(&SecurityEvent{
			EventType:     "password_reset_unknown_email",
			Email:         req.Email,
			IPAddress:     req.IPAddress,
			Success:       false,
			FailureReason: "email not found",
			Timestamp:     time.Now(),
		})
		return nil // Return success to prevent email enumeration
	}

	// Generate password reset token
	resetToken, err := rs.generatePasswordResetToken(user.ID, user.Email)
	if err != nil {
		return fmt.Errorf("failed to generate reset token: %w", err)
	}

	// TODO: Send password reset email with token
	// For now, just log the token (in production, this would be sent via email)
	rs.logger.WithFields(logrus.Fields{
		"user_id":     user.ID,
		"email":       user.Email,
		"reset_token": resetToken,
	}).Info("Password reset token generated")

	rs.logSecurityEvent(&SecurityEvent{
		EventType: "password_reset_initiated",
		UserID:    &user.ID,
		Email:     user.Email,
		IPAddress: req.IPAddress,
		Success:   true,
		Timestamp: time.Now(),
	})

	return nil
}

// ConfirmPasswordReset confirms a password reset using a reset token
func (rs *RegistrationService) ConfirmPasswordReset(ctx context.Context, req *PasswordResetConfirmRequest) error {
	// Validate the reset token
	claims, err := rs.jwtManager.ValidateToken(req.Token, ResetToken)
	if err != nil {
		rs.logSecurityEvent(&SecurityEvent{
			EventType:     "password_reset_invalid_token",
			IPAddress:     req.IPAddress,
			Success:       false,
			FailureReason: "invalid reset token",
			Timestamp:     time.Now(),
		})
		return errors.New("invalid or expired reset token")
	}

	// Get user
	user, err := rs.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Hash new password
	hashedPassword, err := rs.passwordValidator.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update password
	user.HashedPassword = hashedPassword
	user.UpdatedAt = time.Now()

	if err := rs.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Revoke the reset token
	if err := rs.jwtManager.RevokeToken(req.Token, ResetToken); err != nil {
		rs.logger.WithError(err).Warn("Failed to revoke reset token")
	}

	rs.logSecurityEvent(&SecurityEvent{
		EventType: "password_reset_success",
		UserID:    &user.ID,
		Email:     user.Email,
		IPAddress: req.IPAddress,
		Success:   true,
		Timestamp: time.Now(),
	})

	return nil
}

// ResendEmailVerification resends an email verification token
func (rs *RegistrationService) ResendEmailVerification(ctx context.Context, email string, ipAddress string) error {
	// Check rate limiting
	if ipAddress != "" {
		allowed, err := rs.rateLimiter.AllowRequest(ipAddress)
		if err != nil || !allowed {
			return fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Get user
	user, err := rs.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return errors.New("email not found")
	}

	// Check if already verified
	if user.EmailVerified {
		return errors.New("email is already verified")
	}

	// Generate new verification token
	verificationToken, err := rs.generateEmailVerificationToken(user.ID, user.Email)
	if err != nil {
		return fmt.Errorf("failed to generate verification token: %w", err)
	}

	// TODO: Send verification email with token
	// For now, just log the token (in production, this would be sent via email)
	rs.logger.WithFields(logrus.Fields{
		"user_id":            user.ID,
		"email":              user.Email,
		"verification_token": verificationToken,
	}).Info("Email verification token generated")

	return nil
}

// Helper methods

func (rs *RegistrationService) generateEmailVerificationToken(userID uint, email string) (string, error) {
	now := time.Now()
	tokenID := fmt.Sprintf("verify_%d_%d", userID, now.UnixNano())

	claims := &CustomClaims{
		UserID:    userID,
		Email:     email,
		TokenType: string(ResetToken),
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			Subject:   fmt.Sprintf("%d", userID),
			Issuer:    rs.jwtManager.config.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(rs.config.EmailVerificationExpiry)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	// Generate token using the refresh token secret for reset tokens
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(rs.jwtManager.config.RefreshTokenSecret))
}

func (rs *RegistrationService) generatePasswordResetToken(userID uint, email string) (string, error) {
	now := time.Now()
	tokenID := fmt.Sprintf("reset_%d_%d", userID, now.UnixNano())

	claims := &CustomClaims{
		UserID:    userID,
		Email:     email,
		TokenType: string(ResetToken),
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			Subject:   fmt.Sprintf("%d", userID),
			Issuer:    rs.jwtManager.config.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(rs.config.PasswordResetExpiry)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	// Generate token using the refresh token secret for reset tokens
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(rs.jwtManager.config.RefreshTokenSecret))
}

func (rs *RegistrationService) logSecurityEvent(event *SecurityEvent) {
	rs.logger.WithFields(logrus.Fields{
		"event_type":     event.EventType,
		"user_id":        event.UserID,
		"email":          event.Email,
		"ip_address":     event.IPAddress,
		"user_agent":     event.UserAgent,
		"success":        event.Success,
		"failure_reason": event.FailureReason,
		"timestamp":      event.Timestamp,
		"metadata":       event.Metadata,
	}).Info("Security event")
}
