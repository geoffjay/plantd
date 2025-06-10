package auth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/geoffjay/plantd/identity/pkg/client"
	log "github.com/sirupsen/logrus"
)

// UserContext contains authenticated user information.
type UserContext struct {
	UserID      uint         `json:"user_id"`
	UserEmail   string       `json:"user_email"`
	Username    string       `json:"username"`
	Permissions []Permission `json:"permissions"`
	ValidUntil  time.Time    `json:"valid_until"`
}

// CachedPermissions holds cached permission data with TTL.
type CachedPermissions struct {
	UserContext *UserContext
	ExpiresAt   time.Time
}

// AuthMiddleware handles authentication and authorization for state service requests.
type AuthMiddleware struct {
	identityClient  *client.Client
	permissionCache map[string]*CachedPermissions
	cacheTTL        time.Duration
	cacheLock       sync.RWMutex
	logger          *log.Logger
}

// Config holds configuration for the authentication middleware.
type Config struct {
	IdentityClient *client.Client
	CacheTTL       time.Duration
	Logger         *log.Logger
}

// NewAuthMiddleware creates a new authentication middleware instance.
func NewAuthMiddleware(config *Config) *AuthMiddleware {
	if config.CacheTTL == 0 {
		config.CacheTTL = 5 * time.Minute
	}

	if config.Logger == nil {
		config.Logger = log.New()
	}

	return &AuthMiddleware{
		identityClient:  config.IdentityClient,
		permissionCache: make(map[string]*CachedPermissions),
		cacheTTL:        config.CacheTTL,
		logger:          config.Logger,
	}
}

// ValidateRequest validates a request token and checks permissions.
func (am *AuthMiddleware) ValidateRequest(msgType, token, scope string) (*UserContext, error) {
	if token == "" {
		return nil, fmt.Errorf("authentication token required")
	}

	// Check cache first
	if userCtx := am.getCachedPermissions(token); userCtx != nil {
		// Verify required permission
		requiredPerm := getRequiredPermission(msgType)
		if requiredPerm != "" && !am.hasPermission(userCtx, requiredPerm, scope) {
			return nil, fmt.Errorf("insufficient permissions: %s", requiredPerm)
		}
		return userCtx, nil
	}

	// Validate token with identity service
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	validateResp, err := am.identityClient.ValidateToken(ctx, token)
	if err != nil {
		am.logger.WithFields(log.Fields{
			"error": err,
		}).Error("Token validation failed")
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Create user context
	var expiresAt int64
	if validateResp.ExpiresAt != nil {
		expiresAt = *validateResp.ExpiresAt
	}

	// Convert string permissions to Permission structs
	permissions := make([]Permission, len(validateResp.Permissions))
	for i, perm := range validateResp.Permissions {
		permissions[i] = Permission{
			Name:  perm,
			Scope: "", // Global scope by default
		}
	}

	userCtx := &UserContext{
		UserID:      *validateResp.UserID,
		UserEmail:   validateResp.Email,
		Username:    "", // Username not returned in ValidateTokenResponse
		Permissions: permissions,
		ValidUntil:  time.Unix(expiresAt, 0),
	}

	// Cache the permissions
	am.cachePermissions(token, userCtx)

	// Check required permissions
	requiredPerm := getRequiredPermission(msgType)
	if requiredPerm != "" && !am.hasPermission(userCtx, requiredPerm, scope) {
		return nil, fmt.Errorf("insufficient permissions: required=%s scope=%s", requiredPerm, scope)
	}

	am.logger.WithFields(log.Fields{
		"user_email": userCtx.UserEmail,
		"user_id":    userCtx.UserID,
		"operation":  msgType,
		"scope":      scope,
	}).Debug("Request authenticated successfully")

	return userCtx, nil
}

// getCachedPermissions retrieves cached permissions for a token.
func (am *AuthMiddleware) getCachedPermissions(token string) *UserContext {
	am.cacheLock.RLock()
	defer am.cacheLock.RUnlock()

	cached, exists := am.permissionCache[token]
	if !exists || time.Now().After(cached.ExpiresAt) {
		return nil
	}

	return cached.UserContext
}

// cachePermissions caches permissions for a token.
func (am *AuthMiddleware) cachePermissions(token string, userCtx *UserContext) {
	am.cacheLock.Lock()
	defer am.cacheLock.Unlock()

	am.permissionCache[token] = &CachedPermissions{
		UserContext: userCtx,
		ExpiresAt:   time.Now().Add(am.cacheTTL),
	}
}

// hasPermission checks if a user has the required permission for a scope.
func (am *AuthMiddleware) hasPermission(userCtx *UserContext, permission, scope string) bool {
	for _, perm := range userCtx.Permissions {
		// Check for exact permission match
		if perm.Name == permission {
			// Global permission (no scope restriction)
			if perm.Scope == "" || perm.Scope == "*" {
				return true
			}
			// Scoped permission
			if perm.Scope == scope {
				return true
			}
		}

		// Check for wildcard permissions
		if perm.Name == "state:*" || perm.Name == "*" {
			if perm.Scope == "" || perm.Scope == "*" || perm.Scope == scope {
				return true
			}
		}
	}

	return false
}

// ClearCache clears the permission cache.
func (am *AuthMiddleware) ClearCache() {
	am.cacheLock.Lock()
	defer am.cacheLock.Unlock()
	am.permissionCache = make(map[string]*CachedPermissions)
}

// getRequiredPermission maps message types to required permissions.
func getRequiredPermission(msgType string) string {
	switch msgType {
	case "create_scope":
		return StateScopeCreate
	case "delete_scope":
		return StateScopeDelete
	case "set":
		return StateDataWrite
	case "get":
		return StateDataRead
	case "delete":
		return StateDataDelete
	case "health":
		return StateHealthRead
	default:
		return ""
	}
}
