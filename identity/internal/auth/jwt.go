package auth

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenType represents the type of JWT token.
type TokenType string

const (
	// AccessToken represents an access token.
	AccessToken TokenType = "access"
	// RefreshToken represents a refresh token.
	RefreshToken TokenType = "refresh"
	// ResetToken represents a reset token.
	ResetToken TokenType = "reset"
)

// JWTConfig holds configuration for JWT token management.
type JWTConfig struct {
	// AccessTokenSecret is the secret key for signing access tokens
	AccessTokenSecret string `json:"access_token_secret" yaml:"access_token_secret"`
	// RefreshTokenSecret is the secret key for signing refresh tokens
	RefreshTokenSecret string `json:"refresh_token_secret" yaml:"refresh_token_secret"`
	// AccessTokenExpiry is the duration for access token validity
	AccessTokenExpiry time.Duration `json:"access_token_expiry" yaml:"access_token_expiry"`
	// RefreshTokenExpiry is the duration for refresh token validity
	RefreshTokenExpiry time.Duration `json:"refresh_token_expiry" yaml:"refresh_token_expiry"`
	// Issuer is the token issuer identifier
	Issuer string `json:"issuer" yaml:"issuer"`
}

// DefaultJWTConfig returns a secure default JWT configuration.
func DefaultJWTConfig() *JWTConfig {
	return &JWTConfig{
		AccessTokenSecret:  "change-me-in-production",
		RefreshTokenSecret: "change-me-in-production-too",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour, // 7 days
		Issuer:             "plantd-identity",
	}
}

// CustomClaims represents the custom JWT claims structure.
type CustomClaims struct {
	UserID        uint     `json:"user_id"`
	Email         string   `json:"email"`
	Username      string   `json:"username"`
	Organizations []uint   `json:"organizations"`
	Roles         []string `json:"roles"`
	Permissions   []string `json:"permissions"`
	TokenType     string   `json:"token_type"`
	EmailVerified bool     `json:"email_verified"`
	IsActive      bool     `json:"is_active"`
	LastLoginAt   int64    `json:"last_login_at,omitempty"`
	jwt.RegisteredClaims
}

// TokenPair represents a pair of access and refresh tokens.
type TokenPair struct {
	AccessToken           string    `json:"access_token"`
	RefreshToken          string    `json:"refresh_token"`
	AccessTokenExpiresAt  time.Time `json:"access_token_expires_at"`
	RefreshTokenExpiresAt time.Time `json:"refresh_token_expires_at"`
	TokenType             string    `json:"token_type"`
}

// TokenBlacklistService defines the interface for token blacklisting.
type TokenBlacklistService interface {
	BlacklistToken(tokenID string, expiry time.Time) error
	IsTokenBlacklisted(tokenID string) (bool, error)
	CleanupExpiredTokens() error
}

// JWTManager handles JWT token operations.
type JWTManager struct {
	config           *JWTConfig
	blacklistService TokenBlacklistService
}

// NewJWTManager creates a new JWT manager with the given configuration.
func NewJWTManager(config *JWTConfig, blacklistService TokenBlacklistService) *JWTManager {
	if config == nil {
		config = DefaultJWTConfig()
	}
	return &JWTManager{
		config:           config,
		blacklistService: blacklistService,
	}
}

// GenerateTokenPair creates a new access and refresh token pair.
func (jm *JWTManager) GenerateTokenPair(claims *CustomClaims) (*TokenPair, error) {
	now := time.Now()

	// Generate unique token IDs
	accessTokenID := fmt.Sprintf("acc_%d_%d", claims.UserID, now.UnixNano())
	refreshTokenID := fmt.Sprintf("ref_%d_%d", claims.UserID, now.UnixNano())

	// Access token claims
	accessClaims := &CustomClaims{
		UserID:        claims.UserID,
		Email:         claims.Email,
		Username:      claims.Username,
		Organizations: claims.Organizations,
		Roles:         claims.Roles,
		Permissions:   claims.Permissions,
		TokenType:     string(AccessToken),
		EmailVerified: claims.EmailVerified,
		IsActive:      claims.IsActive,
		LastLoginAt:   claims.LastLoginAt,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        accessTokenID,
			Subject:   strconv.Itoa(int(claims.UserID)),
			Issuer:    jm.config.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(jm.config.AccessTokenExpiry)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	// Refresh token claims
	refreshClaims := &CustomClaims{
		UserID:        claims.UserID,
		Email:         claims.Email,
		Username:      claims.Username,
		TokenType:     string(RefreshToken),
		EmailVerified: claims.EmailVerified,
		IsActive:      claims.IsActive,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        refreshTokenID,
			Subject:   strconv.Itoa(int(claims.UserID)),
			Issuer:    jm.config.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(jm.config.RefreshTokenExpiry)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	// Generate access token
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(jm.config.AccessTokenSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Generate refresh token
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(jm.config.RefreshTokenSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:           accessTokenString,
		RefreshToken:          refreshTokenString,
		AccessTokenExpiresAt:  accessClaims.ExpiresAt.Time,
		RefreshTokenExpiresAt: refreshClaims.ExpiresAt.Time,
		TokenType:             "Bearer",
	}, nil
}

// ValidateToken validates and parses a JWT token.
func (jm *JWTManager) ValidateToken(tokenString string, tokenType TokenType) (*CustomClaims, error) {
	var secret string
	switch tokenType {
	case AccessToken:
		secret = jm.config.AccessTokenSecret
	case RefreshToken:
		secret = jm.config.RefreshTokenSecret
	case ResetToken:
		secret = jm.config.AccessTokenSecret // Use access token secret for reset tokens
	default:
		return nil, errors.New("invalid token type")
	}

	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	// Check if token type matches expected type
	if claims.TokenType != string(tokenType) {
		return nil, fmt.Errorf("token type mismatch: expected %s, got %s", tokenType, claims.TokenType)
	}

	// Check if token is blacklisted
	if jm.blacklistService != nil {
		blacklisted, err := jm.blacklistService.IsTokenBlacklisted(claims.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to check token blacklist: %w", err)
		}
		if blacklisted {
			return nil, errors.New("token has been revoked")
		}
	}

	// Validate token expiry
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("token has expired")
	}

	// Validate not before
	if claims.NotBefore != nil && claims.NotBefore.After(time.Now()) {
		return nil, errors.New("token not yet valid")
	}

	return claims, nil
}

// RefreshTokenPair generates a new token pair using a valid refresh token.
func (jm *JWTManager) RefreshTokenPair(refreshTokenString string) (*TokenPair, error) {
	// Validate refresh token
	claims, err := jm.ValidateToken(refreshTokenString, RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Blacklist the old refresh token
	if jm.blacklistService != nil && claims.ExpiresAt != nil {
		if err := jm.blacklistService.BlacklistToken(claims.ID, claims.ExpiresAt.Time); err != nil {
			return nil, fmt.Errorf("failed to blacklist old refresh token: %w", err)
		}
	}

	// Generate new token pair with current timestamp
	newClaims := &CustomClaims{
		UserID:        claims.UserID,
		Email:         claims.Email,
		Username:      claims.Username,
		Organizations: claims.Organizations,
		Roles:         claims.Roles,
		Permissions:   claims.Permissions,
		EmailVerified: claims.EmailVerified,
		IsActive:      claims.IsActive,
		LastLoginAt:   time.Now().Unix(),
	}

	return jm.GenerateTokenPair(newClaims)
}

// RevokeToken adds a token to the blacklist.
func (jm *JWTManager) RevokeToken(tokenString string, tokenType TokenType) error {
	claims, err := jm.ValidateToken(tokenString, tokenType)
	if err != nil {
		return fmt.Errorf("invalid token for revocation: %w", err)
	}

	if jm.blacklistService == nil {
		return errors.New("blacklist service not available")
	}

	if claims.ExpiresAt != nil {
		return jm.blacklistService.BlacklistToken(claims.ID, claims.ExpiresAt.Time)
	}

	return errors.New("token has no expiration time")
}

// RevokeAllUserTokens revokes all tokens for a specific user.
func (jm *JWTManager) RevokeAllUserTokens(userID uint) error {
	// This would typically involve invalidating all tokens for a user
	// Implementation depends on the blacklist service capabilities
	// For now, we'll return an error indicating this needs to be implemented
	return fmt.Errorf("revoking all tokens for user %d not yet implemented", userID)
}

// ExtractTokenFromAuthHeader extracts JWT token from Authorization header.
func (jm *JWTManager) ExtractTokenFromAuthHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", errors.New("authorization header is required")
	}

	// Expected format: "Bearer <token>"
	const bearerPrefix = "Bearer "
	if len(authHeader) < len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
		return "", errors.New("authorization header must start with 'Bearer '")
	}

	token := authHeader[len(bearerPrefix):]
	if token == "" {
		return "", errors.New("token is required")
	}

	return token, nil
}

// GetTokenClaims is a convenience method to get claims from a token string.
func (jm *JWTManager) GetTokenClaims(tokenString string) (*CustomClaims, error) {
	// Try as access token first
	if claims, err := jm.ValidateToken(tokenString, AccessToken); err == nil {
		return claims, nil
	}

	// Try as refresh token
	if claims, err := jm.ValidateToken(tokenString, RefreshToken); err == nil {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// CleanupExpiredTokens removes expired tokens from blacklist.
func (jm *JWTManager) CleanupExpiredTokens() error {
	if jm.blacklistService == nil {
		return errors.New("blacklist service not available")
	}
	return jm.blacklistService.CleanupExpiredTokens()
}
