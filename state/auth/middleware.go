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
type AuthMiddleware struct { //nolint:revive
	identityClient  *client.Client
	permissionCache map[string]*CachedPermissions
	accessChecker   *AccessChecker
	roleManager     *RoleManager
	cacheTTL        time.Duration
	cacheMutex      sync.RWMutex
	logger          *log.Logger
}

// Config holds configuration for the authentication middleware.
type Config struct {
	IdentityClient *client.Client
	AccessChecker  *AccessChecker
	RoleManager    *RoleManager
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

	// Create access checker if not provided
	accessChecker := config.AccessChecker
	if accessChecker == nil {
		accessChecker = NewAccessChecker(&AccessCheckerConfig{
			IdentityClient: config.IdentityClient,
			CacheTTL:       config.CacheTTL,
			Logger:         config.Logger,
		})
	}

	// Create role manager if not provided
	roleManager := config.RoleManager
	if roleManager == nil {
		roleManager = NewRoleManager(&RoleManagerConfig{
			IdentityClient: config.IdentityClient,
			Logger:         config.Logger,
		})
	}

	return &AuthMiddleware{
		identityClient:  config.IdentityClient,
		permissionCache: make(map[string]*CachedPermissions),
		accessChecker:   accessChecker,
		roleManager:     roleManager,
		cacheTTL:        config.CacheTTL,
		logger:          config.Logger,
	}
}

// ValidateRequest validates an authentication token and checks permissions.
func (am *AuthMiddleware) ValidateRequest(msgType, token, scope string) (*UserContext, error) {
	if token == "" {
		return nil, fmt.Errorf("authentication token required")
	}

	// Check cache first
	cacheKey := fmt.Sprintf("%s:%s", token, scope)
	am.cacheMutex.RLock()
	if cached, found := am.permissionCache[cacheKey]; found {
		if time.Now().Before(cached.ExpiresAt) {
			am.cacheMutex.RUnlock()
			log.WithFields(log.Fields{
				"user_email": cached.UserContext.UserEmail,
				"scope":      scope,
				"cache_hit":  true,
			}).Debug("Permission cache hit")
			return cached.UserContext, nil
		}
		// Cache entry expired, remove it
		delete(am.permissionCache, cacheKey)
	}
	am.cacheMutex.RUnlock()

	// Validate token with identity service
	validateResp, err := am.identityClient.ValidateToken(context.Background(), token)
	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	if !validateResp.Valid {
		return nil, fmt.Errorf("invalid token")
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

	// Check specific permissions for the operation using RBAC
	requiredPermission := am.getRequiredPermission(msgType)
	if err := am.accessChecker.CheckScopeAccess(userCtx, requiredPermission, scope); err != nil {
		return nil, fmt.Errorf("access denied for %s on scope %s: %w", msgType, scope, err)
	}

	// Cache the result
	am.cacheMutex.Lock()
	am.permissionCache[cacheKey] = &CachedPermissions{
		UserContext: userCtx,
		ExpiresAt:   time.Now().Add(am.cacheTTL),
	}
	am.cacheMutex.Unlock()

	log.WithFields(log.Fields{
		"user_email": userCtx.UserEmail,
		"user_id":    userCtx.UserID,
		"scope":      scope,
		"operation":  msgType,
		"permission": requiredPermission,
		"cache_miss": true,
	}).Debug("Authentication and authorization successful")

	return userCtx, nil
}

// getRequiredPermission maps operation types to required permissions.
func (am *AuthMiddleware) getRequiredPermission(msgType string) string {
	switch msgType {
	case "create-scope":
		return StateScopeCreate
	case "delete-scope":
		return StateScopeDelete
	case "set":
		return StateDataWrite
	case "get":
		return StateDataRead
	case "delete":
		return StateDataDelete
	case "list-scopes":
		return StateScopeList
	case "list-keys":
		return StateDataRead // Reading keys requires read permission
	case "health":
		return StateHealthRead
	default:
		// For unknown operations, require admin permission
		return StateAdminFull
	}
}

// checkPermission checks if a user has the required permission for a scope.
func (am *AuthMiddleware) checkPermission(userCtx *UserContext, requiredPermission, scope string) bool { //nolint:unused
	// Check for admin permission (overrides all)
	if am.hasPermission(userCtx, StateAdminFull) {
		return true
	}

	// Check for global permission
	if am.hasPermission(userCtx, requiredPermission) {
		return true
	}

	// Check for scoped permission
	scopedPermission := fmt.Sprintf("%s:scope:%s", requiredPermission, scope)
	if am.hasPermission(userCtx, scopedPermission) {
		return true
	}

	// Check for scope-specific permissions
	switch requiredPermission {
	case StateDataRead:
		// Allow if user has any data permission on the scope
		if am.hasPermission(userCtx, fmt.Sprintf("state:scope:%s:read", scope)) ||
			am.hasPermission(userCtx, fmt.Sprintf("state:scope:%s:admin", scope)) {
			return true
		}
	case StateDataWrite:
		// Allow if user has write or admin permission on the scope
		if am.hasPermission(userCtx, fmt.Sprintf("state:scope:%s:write", scope)) ||
			am.hasPermission(userCtx, fmt.Sprintf("state:scope:%s:admin", scope)) {
			return true
		}
	case StateDataDelete:
		// Allow if user has delete or admin permission on the scope
		if am.hasPermission(userCtx, fmt.Sprintf("state:scope:%s:delete", scope)) ||
			am.hasPermission(userCtx, fmt.Sprintf("state:scope:%s:admin", scope)) {
			return true
		}
	}

	return false
}

// hasPermission checks if a user has a specific permission.
func (am *AuthMiddleware) hasPermission(userCtx *UserContext, permission string) bool { //nolint:unused
	for _, userPerm := range userCtx.Permissions {
		if userPerm.Name == permission {
			return true
		}
		// Check for wildcard permissions
		if userPerm.Name == "state:*" {
			return true
		}
	}
	return false
}

// ClearCache clears the permission cache.
func (am *AuthMiddleware) ClearCache() {
	am.cacheMutex.Lock()
	defer am.cacheMutex.Unlock()
	am.permissionCache = make(map[string]*CachedPermissions)
}
