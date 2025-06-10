// Package auth provides authentication and authorization services.
package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/geoffjay/plantd/identity/internal/models"
	"github.com/geoffjay/plantd/identity/internal/repositories"
	"github.com/geoffjay/plantd/identity/internal/services"
	"github.com/sirupsen/logrus"
)

// AuthRequest represents a login request.
type AuthRequest struct { //nolint:revive
	Identifier string `json:"identifier" validate:"required"` // email or username
	Password   string `json:"password" validate:"required"`
	IPAddress  string `json:"ip_address,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
}

// AuthResponse represents the response to a successful authentication.
type AuthResponse struct { //nolint:revive
	User         *models.User `json:"user"`
	TokenPair    *TokenPair   `json:"token_pair"`
	ExpiresAt    time.Time    `json:"expires_at"`
	RefreshToken string       `json:"refresh_token"`
}

// RefreshRequest represents a token refresh request.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
	IPAddress    string `json:"ip_address,omitempty"`
}

// SecurityEvent represents a security-related event for logging.
type SecurityEvent struct {
	EventType     string                 `json:"event_type"`
	UserID        *uint                  `json:"user_id,omitempty"`
	Email         string                 `json:"email,omitempty"`
	IPAddress     string                 `json:"ip_address,omitempty"`
	UserAgent     string                 `json:"user_agent,omitempty"`
	Success       bool                   `json:"success"`
	FailureReason string                 `json:"failure_reason,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// AuthConfig holds configuration for the authentication service.
type AuthConfig struct { //nolint:revive
	Password       *PasswordConfig    `json:"password" yaml:"password"`
	JWT            *JWTConfig         `json:"jwt" yaml:"jwt"`
	RateLimit      *RateLimiterConfig `json:"rate_limit" yaml:"rate_limit"`
	SessionTimeout time.Duration      `json:"session_timeout" yaml:"session_timeout"`
}

// DefaultAuthConfig returns a secure default authentication configuration.
func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		Password:       DefaultPasswordConfig(),
		JWT:            DefaultJWTConfig(),
		RateLimit:      DefaultRateLimiterConfig(),
		SessionTimeout: 24 * time.Hour,
	}
}

// AuthService provides authentication functionality.
type AuthService struct { //nolint:revive
	config            *AuthConfig
	userRepo          repositories.UserRepository
	userService       services.UserService
	passwordValidator *PasswordValidator
	jwtManager        *JWTManager
	rateLimiter       *RateLimiter
	logger            *logrus.Logger
}

// NewAuthService creates a new authentication service.
func NewAuthService(
	config *AuthConfig,
	userRepo repositories.UserRepository,
	userService services.UserService,
	logger *logrus.Logger,
) *AuthService {
	if config == nil {
		config = DefaultAuthConfig()
	}

	// Create blacklist service
	blacklistService := NewInMemoryBlacklist()

	return &AuthService{
		config:            config,
		userRepo:          userRepo,
		userService:       userService,
		passwordValidator: NewPasswordValidator(config.Password),
		jwtManager:        NewJWTManager(config.JWT, blacklistService),
		rateLimiter:       NewRateLimiter(config.RateLimit),
		logger:            logger,
	}
}

// Login authenticates a user and returns tokens.
func (as *AuthService) Login(ctx context.Context, req *AuthRequest) (*AuthResponse, error) {
	// Check rate limiting.
	if req.IPAddress != "" {
		allowed, err := as.rateLimiter.AllowRequest(req.IPAddress)
		if err != nil || !allowed {
			as.logSecurityEvent(&SecurityEvent{
				EventType:     "login_rate_limited",
				Email:         req.Identifier,
				IPAddress:     req.IPAddress,
				UserAgent:     req.UserAgent,
				Success:       false,
				FailureReason: "rate limit exceeded",
				Timestamp:     time.Now(),
			})
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Check account lockout.
	locked, lockedUntil, err := as.rateLimiter.IsAccountLocked(req.Identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to check account lockout: %w", err)
	}
	if locked {
		as.logSecurityEvent(&SecurityEvent{
			EventType:     "login_account_locked",
			Email:         req.Identifier,
			IPAddress:     req.IPAddress,
			UserAgent:     req.UserAgent,
			Success:       false,
			FailureReason: fmt.Sprintf("account locked until %v", lockedUntil),
			Timestamp:     time.Now(),
		})
		return nil, fmt.Errorf("account is locked until %v", lockedUntil)
	}

	// Find user by email or username.
	user, err := as.userRepo.GetByEmail(ctx, req.Identifier)
	if err != nil {
		// Try by username if email lookup failed
		user, err = as.userRepo.GetByUsername(ctx, req.Identifier)
		if err != nil {
			// Record failed login attempt
			if rateLimitErr := as.rateLimiter.RecordFailedLogin(req.Identifier); rateLimitErr != nil {
				as.logger.WithError(rateLimitErr).Warn("Failed to record failed login attempt")
			}
			as.logSecurityEvent(&SecurityEvent{
				EventType:     "login_user_not_found",
				Email:         req.Identifier,
				IPAddress:     req.IPAddress,
				UserAgent:     req.UserAgent,
				Success:       false,
				FailureReason: "user not found",
				Timestamp:     time.Now(),
			})
			return nil, errors.New("invalid credentials")
		}
	}

	// Check if user is active
	if !user.IsActive {
		if rateLimitErr := as.rateLimiter.RecordFailedLogin(req.Identifier); rateLimitErr != nil {
			as.logger.WithError(rateLimitErr).Warn("Failed to record failed login attempt")
		}
		as.logSecurityEvent(&SecurityEvent{
			EventType:     "login_user_inactive",
			UserID:        &user.ID,
			Email:         user.Email,
			IPAddress:     req.IPAddress,
			UserAgent:     req.UserAgent,
			Success:       false,
			FailureReason: "user account is inactive",
			Timestamp:     time.Now(),
		})
		return nil, errors.New("account is inactive")
	}

	// Verify password
	if err := as.passwordValidator.VerifyPassword(user.HashedPassword, req.Password); err != nil {
		// Record failed login attempt
		if rateLimitErr := as.rateLimiter.RecordFailedLogin(req.Identifier); rateLimitErr != nil {
			as.logger.WithError(rateLimitErr).Warn("Failed to record failed login attempt")
		}
		as.logSecurityEvent(&SecurityEvent{
			EventType:     "login_invalid_password",
			UserID:        &user.ID,
			Email:         user.Email,
			IPAddress:     req.IPAddress,
			UserAgent:     req.UserAgent,
			Success:       false,
			FailureReason: "invalid password",
			Timestamp:     time.Now(),
		})
		return nil, errors.New("invalid credentials")
	}

	// Get user's organizations and roles
	organizations, err := as.getUserOrganizations(ctx, user.ID)
	if err != nil {
		as.logger.WithError(err).WithField("user_id", user.ID).Error("Failed to get user organizations")
		organizations = []uint{} // Continue with empty organizations
	}

	roles, permissions, err := as.getUserRolesAndPermissions(ctx, user.ID)
	if err != nil {
		as.logger.WithError(err).WithField("user_id", user.ID).Error("Failed to get user roles and permissions")
		roles = []string{}       // Continue with empty roles
		permissions = []string{} // Continue with empty permissions
	}

	// Create JWT claims
	claims := &CustomClaims{
		UserID:        user.ID,
		Email:         user.Email,
		Username:      user.Username,
		Organizations: organizations,
		Roles:         roles,
		Permissions:   permissions,
		EmailVerified: user.EmailVerified,
		IsActive:      user.IsActive,
		LastLoginAt:   time.Now().Unix(),
	}

	// Generate token pair
	tokenPair, err := as.jwtManager.GenerateTokenPair(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Update user's last login time
	if err := as.updateLastLogin(ctx, user.ID); err != nil {
		as.logger.WithError(err).WithField("user_id", user.ID).Warn("Failed to update last login time")
	}

	// Record successful login
	if rateLimitErr := as.rateLimiter.RecordSuccessfulLogin(req.Identifier); rateLimitErr != nil {
		as.logger.WithError(rateLimitErr).Warn("Failed to record successful login")
	}
	as.logSecurityEvent(&SecurityEvent{
		EventType: "login_success",
		UserID:    &user.ID,
		Email:     user.Email,
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
		Success:   true,
		Timestamp: time.Now(),
	})

	return &AuthResponse{
		User:         user,
		TokenPair:    tokenPair,
		ExpiresAt:    tokenPair.AccessTokenExpiresAt,
		RefreshToken: tokenPair.RefreshToken,
	}, nil
}

// RefreshToken generates a new token pair using a valid refresh token.
func (as *AuthService) RefreshToken(_ context.Context, req *RefreshRequest) (*TokenPair, error) {
	// Check rate limiting for refresh requests
	if req.IPAddress != "" {
		allowed, err := as.rateLimiter.AllowRequest(req.IPAddress)
		if err != nil || !allowed {
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Refresh the token pair
	tokenPair, err := as.jwtManager.RefreshTokenPair(req.RefreshToken)
	if err != nil {
		as.logSecurityEvent(&SecurityEvent{
			EventType:     "token_refresh_failed",
			IPAddress:     req.IPAddress,
			Success:       false,
			FailureReason: err.Error(),
			Timestamp:     time.Now(),
		})
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	as.logSecurityEvent(&SecurityEvent{
		EventType: "token_refresh_success",
		IPAddress: req.IPAddress,
		Success:   true,
		Timestamp: time.Now(),
	})

	return tokenPair, nil
}

// Logout invalidates the provided token.
func (as *AuthService) Logout(_ context.Context, accessToken string) error {
	// Revoke the access token.
	if err := as.jwtManager.RevokeToken(accessToken, AccessToken); err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	// Extract user info for logging
	claims, err := as.jwtManager.GetTokenClaims(accessToken)
	if err == nil {
		as.logSecurityEvent(&SecurityEvent{
			EventType: "logout_success",
			UserID:    &claims.UserID,
			Email:     claims.Email,
			Success:   true,
			Timestamp: time.Now(),
		})
	}

	return nil
}

// ValidateToken validates an access token and returns the claims.
func (as *AuthService) ValidateToken(_ context.Context, tokenString string) (*CustomClaims, error) {
	return as.jwtManager.ValidateToken(tokenString, AccessToken)
}

// ChangePassword allows a user to change their password.
func (as *AuthService) ChangePassword(ctx context.Context, userID uint, currentPassword, newPassword string) error {
	// Get user.
	user, err := as.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Verify current password.
	if err := as.passwordValidator.VerifyPassword(user.HashedPassword, currentPassword); err != nil {
		as.logSecurityEvent(&SecurityEvent{
			EventType:     "password_change_failed",
			UserID:        &userID,
			Email:         user.Email,
			Success:       false,
			FailureReason: "invalid current password",
			Timestamp:     time.Now(),
		})
		return errors.New("invalid current password")
	}

	// Hash new password.
	hashedPassword, err := as.passwordValidator.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update password in database.
	user.HashedPassword = hashedPassword
	user.UpdatedAt = time.Now()

	if err := as.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	as.logSecurityEvent(&SecurityEvent{
		EventType: "password_change_success",
		UserID:    &userID,
		Email:     user.Email,
		Success:   true,
		Timestamp: time.Now(),
	})

	return nil
}

// GetPasswordStrength returns the strength score of a password.
func (as *AuthService) GetPasswordStrength(password string) int {
	return as.passwordValidator.GetPasswordStrength(password)
}

// UnlockAccount manually unlocks a locked account (admin function).
func (as *AuthService) UnlockAccount(_ context.Context, identifier string) error {
	return as.rateLimiter.UnlockAccount(identifier)
}

// GetSecurityStats returns security-related statistics.
func (as *AuthService) GetSecurityStats() map[string]interface{} {
	return as.rateLimiter.GetStats()
}

// Helper methods.

func (as *AuthService) getUserOrganizations(_ context.Context, _ uint) ([]uint, error) {
	// This would query the user_organizations table.
	// For now, return empty slice
	return []uint{}, nil
}

func (as *AuthService) getUserRolesAndPermissions(_ context.Context, _ uint) ([]string, []string, error) {
	// This would query the user_roles table and extract permissions from roles.
	// For now, return empty slices.
	return []string{}, []string{}, nil
}

func (as *AuthService) updateLastLogin(ctx context.Context, userID uint) error {
	user, err := as.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	now := time.Now()
	user.LastLoginAt = &now
	user.UpdatedAt = now

	return as.userRepo.Update(ctx, user)
}

func (as *AuthService) logSecurityEvent(event *SecurityEvent) {
	as.logger.WithFields(logrus.Fields{
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

// Stop gracefully stops the authentication service.
func (as *AuthService) Stop() {
	if as.rateLimiter != nil {
		as.rateLimiter.Stop()
	}
}
