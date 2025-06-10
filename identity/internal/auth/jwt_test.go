package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultJWTConfig(t *testing.T) {
	config := DefaultJWTConfig()

	assert.NotNil(t, config)
	assert.Equal(t, "plantd-identity", config.Issuer)
	assert.Equal(t, 15*time.Minute, config.AccessTokenExpiry)
	assert.Equal(t, 7*24*time.Hour, config.RefreshTokenExpiry)
	assert.NotEmpty(t, config.AccessTokenSecret)
	assert.NotEmpty(t, config.RefreshTokenSecret)
}

func TestNewJWTManager(t *testing.T) {
	t.Run("with custom config", func(t *testing.T) {
		config := &JWTConfig{
			AccessTokenSecret:  "test-access-secret-32-chars!",
			RefreshTokenSecret: "test-refresh-secret-32-chars",
			Issuer:             "test-issuer",
			AccessTokenExpiry:  30 * time.Minute,
			RefreshTokenExpiry: 14 * 24 * time.Hour,
		}

		manager := NewJWTManager(config, nil)
		assert.NotNil(t, manager)
		assert.Equal(t, config, manager.config)
	})

	t.Run("with nil config uses default", func(t *testing.T) {
		manager := NewJWTManager(nil, nil)
		assert.NotNil(t, manager)
		assert.Equal(t, DefaultJWTConfig(), manager.config)
	})
}

func TestJWTManager_GenerateTokenPair(t *testing.T) {
	manager := NewJWTManager(DefaultJWTConfig(), nil)

	claims := &CustomClaims{
		UserID:        1,
		Email:         "test@example.com",
		Username:      "testuser",
		Organizations: []uint{1, 2},
		Roles:         []string{"user", "admin"},
		Permissions:   []string{"read", "write"},
		EmailVerified: true,
		IsActive:      true,
		LastLoginAt:   time.Now().Unix(),
	}

	tokenPair, err := manager.GenerateTokenPair(claims)
	require.NoError(t, err)
	assert.NotNil(t, tokenPair)

	// Verify token pair structure
	assert.NotEmpty(t, tokenPair.AccessToken)
	assert.NotEmpty(t, tokenPair.RefreshToken)
	assert.Equal(t, "Bearer", tokenPair.TokenType)
	assert.True(t, tokenPair.AccessTokenExpiresAt.After(time.Now()))
	assert.True(t, tokenPair.RefreshTokenExpiresAt.After(time.Now()))

	// Verify access token claims
	accessClaims, err := manager.ValidateToken(tokenPair.AccessToken, AccessToken)
	require.NoError(t, err)
	assert.Equal(t, claims.UserID, accessClaims.UserID)
	assert.Equal(t, claims.Email, accessClaims.Email)
	assert.Equal(t, string(AccessToken), accessClaims.TokenType)

	// Verify refresh token claims
	refreshClaims, err := manager.ValidateToken(tokenPair.RefreshToken, RefreshToken)
	require.NoError(t, err)
	assert.Equal(t, claims.UserID, refreshClaims.UserID)
	assert.Equal(t, claims.Email, refreshClaims.Email)
	assert.Equal(t, string(RefreshToken), refreshClaims.TokenType)
}

func TestJWTManager_ValidateToken(t *testing.T) {
	manager := NewJWTManager(DefaultJWTConfig(), nil)

	claims := &CustomClaims{
		UserID:        1,
		Email:         "test@example.com",
		Username:      "testuser",
		Organizations: []uint{1},
		Roles:         []string{"user"},
		Permissions:   []string{"read"},
		EmailVerified: true,
		IsActive:      true,
	}

	tokenPair, err := manager.GenerateTokenPair(claims)
	require.NoError(t, err)

	tests := []struct {
		name      string
		token     string
		tokenType TokenType
		wantErr   bool
	}{
		{
			name:      "valid access token",
			token:     tokenPair.AccessToken,
			tokenType: AccessToken,
			wantErr:   false,
		},
		{
			name:      "valid refresh token",
			token:     tokenPair.RefreshToken,
			tokenType: RefreshToken,
			wantErr:   false,
		},
		{
			name:      "access token with wrong type",
			token:     tokenPair.AccessToken,
			tokenType: RefreshToken,
			wantErr:   true,
		},
		{
			name:      "refresh token with wrong type",
			token:     tokenPair.RefreshToken,
			tokenType: AccessToken,
			wantErr:   true,
		},
		{
			name:      "invalid token format",
			token:     "invalid.token.format",
			tokenType: AccessToken,
			wantErr:   true,
		},
		{
			name:      "empty token",
			token:     "",
			tokenType: AccessToken,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validatedClaims, err := manager.ValidateToken(tt.token, tt.tokenType)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, validatedClaims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, validatedClaims)
				assert.Equal(t, claims.UserID, validatedClaims.UserID)
				assert.Equal(t, claims.Email, validatedClaims.Email)
			}
		})
	}
}

func TestJWTManager_RefreshTokenPair(t *testing.T) {
	manager := NewJWTManager(DefaultJWTConfig(), nil)

	claims := &CustomClaims{
		UserID:        1,
		Email:         "test@example.com",
		Username:      "testuser",
		Organizations: []uint{1},
		Roles:         []string{"user"},
		Permissions:   []string{"read"},
		EmailVerified: true,
		IsActive:      true,
	}

	originalTokenPair, err := manager.GenerateTokenPair(claims)
	require.NoError(t, err)

	tests := []struct {
		name         string
		refreshToken string
		wantErr      bool
	}{
		{
			name:         "valid refresh token",
			refreshToken: originalTokenPair.RefreshToken,
			wantErr:      false,
		},
		{
			name:         "empty refresh token",
			refreshToken: "",
			wantErr:      true,
		},
		{
			name:         "invalid refresh token",
			refreshToken: "invalid.token.format",
			wantErr:      true,
		},
		{
			name:         "access token instead of refresh",
			refreshToken: originalTokenPair.AccessToken,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newTokenPair, err := manager.RefreshTokenPair(tt.refreshToken)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, newTokenPair)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, newTokenPair)

				// Verify new token pair is different from original
				assert.NotEqual(t, originalTokenPair.AccessToken, newTokenPair.AccessToken)
				assert.NotEqual(t, originalTokenPair.RefreshToken, newTokenPair.RefreshToken)

				// Verify claims are preserved
				newAccessClaims, err := manager.ValidateToken(newTokenPair.AccessToken, AccessToken)
				assert.NoError(t, err)
				assert.Equal(t, claims.UserID, newAccessClaims.UserID)
				assert.Equal(t, claims.Email, newAccessClaims.Email)
			}
		})
	}
}

func TestJWTManager_ExtractTokenFromAuthHeader(t *testing.T) {
	manager := NewJWTManager(DefaultJWTConfig(), nil)

	tests := []struct {
		name       string
		authHeader string
		expected   string
		wantErr    bool
	}{
		{
			name:       "valid Bearer token",
			authHeader: "Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.token",
			expected:   "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.token",
			wantErr:    false,
		},
		{
			name:       "bearer token (lowercase) - should fail",
			authHeader: "bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ0ZXN0In0.token",
			expected:   "",
			wantErr:    true,
		},
		{
			name:       "empty auth header",
			authHeader: "",
			expected:   "",
			wantErr:    true,
		},
		{
			name:       "invalid auth header format",
			authHeader: "InvalidToken",
			expected:   "",
			wantErr:    true,
		},
		{
			name:       "Bearer without token",
			authHeader: "Bearer ",
			expected:   "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := manager.ExtractTokenFromAuthHeader(tt.authHeader)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, token)
			}
		})
	}
}

func TestJWTManager_GetTokenClaims(t *testing.T) {
	manager := NewJWTManager(DefaultJWTConfig(), nil)

	claims := &CustomClaims{
		UserID:        42,
		Email:         "test@example.com",
		Username:      "testuser",
		Organizations: []uint{1, 2, 3},
		Roles:         []string{"admin", "user"},
		Permissions:   []string{"create", "read", "update", "delete"},
		EmailVerified: true,
		IsActive:      true,
		LastLoginAt:   time.Now().Unix(),
	}

	tokenPair, err := manager.GenerateTokenPair(claims)
	require.NoError(t, err)

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "valid access token",
			token:   tokenPair.AccessToken,
			wantErr: false,
		},
		{
			name:    "valid refresh token",
			token:   tokenPair.RefreshToken,
			wantErr: false,
		},
		{
			name:    "invalid token",
			token:   "invalid.token.format",
			wantErr: true,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractedClaims, err := manager.GetTokenClaims(tt.token)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, extractedClaims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, extractedClaims)
				assert.Equal(t, claims.UserID, extractedClaims.UserID)
				assert.Equal(t, claims.Email, extractedClaims.Email)
				assert.Equal(t, claims.Username, extractedClaims.Username)
				assert.Equal(t, claims.EmailVerified, extractedClaims.EmailVerified)
				assert.Equal(t, claims.IsActive, extractedClaims.IsActive)

				// Access tokens contain full claims, refresh tokens don't include orgs/roles/perms
				switch tt.name {
				case "valid access token":
					assert.ElementsMatch(t, claims.Organizations, extractedClaims.Organizations)
					assert.ElementsMatch(t, claims.Roles, extractedClaims.Roles)
					assert.ElementsMatch(t, claims.Permissions, extractedClaims.Permissions)
				case "valid refresh token":
					// Refresh tokens don't include organizations, roles, and permissions
					assert.Nil(t, extractedClaims.Organizations)
					assert.Nil(t, extractedClaims.Roles)
					assert.Nil(t, extractedClaims.Permissions)
				}
			}
		})
	}
}

func TestJWTManager_TokensWithDifferentSecrets(t *testing.T) {
	manager1 := NewJWTManager(&JWTConfig{
		AccessTokenSecret:  "secret-1-access-32-characters",
		RefreshTokenSecret: "secret-1-refresh-32-character",
		Issuer:             "test-issuer",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}, nil)

	manager2 := NewJWTManager(&JWTConfig{
		AccessTokenSecret:  "secret-2-access-32-characters",
		RefreshTokenSecret: "secret-2-refresh-32-character",
		Issuer:             "test-issuer",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}, nil)

	claims := &CustomClaims{
		UserID:   1,
		Email:    "test@example.com",
		Username: "testuser",
	}

	// Generate token with manager1
	tokenPair, err := manager1.GenerateTokenPair(claims)
	require.NoError(t, err)

	// Try to validate with manager2 (different secrets)
	_, err = manager2.ValidateToken(tokenPair.AccessToken, AccessToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse token")

	_, err = manager2.ValidateToken(tokenPair.RefreshToken, RefreshToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse token")
}

func TestJWTManager_ExpiredTokens(t *testing.T) {
	// Create manager with very short expiry for testing
	config := &JWTConfig{
		AccessTokenSecret:  "test-access-secret-32-chars!",
		RefreshTokenSecret: "test-refresh-secret-32-chars",
		Issuer:             "test-issuer",
		AccessTokenExpiry:  1 * time.Nanosecond, // Immediately expired
		RefreshTokenExpiry: 1 * time.Nanosecond,
	}

	manager := NewJWTManager(config, nil)

	claims := &CustomClaims{
		UserID:   1,
		Email:    "test@example.com",
		Username: "testuser",
	}

	// Generate tokens that will be expired
	tokenPair, err := manager.GenerateTokenPair(claims)
	require.NoError(t, err)

	// Wait to ensure expiration
	time.Sleep(2 * time.Nanosecond)

	// Try to validate expired tokens
	_, err = manager.ValidateToken(tokenPair.AccessToken, AccessToken)
	assert.Error(t, err)

	_, err = manager.ValidateToken(tokenPair.RefreshToken, RefreshToken)
	assert.Error(t, err)
}

func TestCustomClaims_Structure(t *testing.T) {
	claims := &CustomClaims{
		UserID:        123,
		Email:         "user@company.com",
		Username:      "username123",
		Organizations: []uint{1, 2, 3},
		Roles:         []string{"admin", "moderator"},
		Permissions:   []string{"read", "write", "delete"},
		TokenType:     "access",
		EmailVerified: true,
		IsActive:      false,
		LastLoginAt:   1234567890,
	}

	// Verify structure integrity
	assert.Equal(t, uint(123), claims.UserID)
	assert.Equal(t, "user@company.com", claims.Email)
	assert.Equal(t, "username123", claims.Username)
	assert.ElementsMatch(t, []uint{1, 2, 3}, claims.Organizations)
	assert.ElementsMatch(t, []string{"admin", "moderator"}, claims.Roles)
	assert.ElementsMatch(t, []string{"read", "write", "delete"}, claims.Permissions)
	assert.Equal(t, "access", claims.TokenType)
	assert.True(t, claims.EmailVerified)
	assert.False(t, claims.IsActive)
	assert.Equal(t, int64(1234567890), claims.LastLoginAt)
}

func TestTokenPair_Structure(t *testing.T) {
	now := time.Now()
	tokenPair := &TokenPair{
		AccessToken:           "access.token.string",
		RefreshToken:          "refresh.token.string",
		AccessTokenExpiresAt:  now.Add(15 * time.Minute),
		RefreshTokenExpiresAt: now.Add(7 * 24 * time.Hour),
		TokenType:             "Bearer",
	}

	// Verify structure integrity
	assert.Equal(t, "access.token.string", tokenPair.AccessToken)
	assert.Equal(t, "refresh.token.string", tokenPair.RefreshToken)
	assert.Equal(t, "Bearer", tokenPair.TokenType)
	assert.True(t, tokenPair.AccessTokenExpiresAt.After(now))
	assert.True(t, tokenPair.RefreshTokenExpiresAt.After(now))
	assert.True(t, tokenPair.RefreshTokenExpiresAt.After(tokenPair.AccessTokenExpiresAt))
}
