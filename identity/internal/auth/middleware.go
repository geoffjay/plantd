package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// authContextKey is a typed key for storing auth context in request context.
type authContextKey struct{}

var authCtxKey = authContextKey{}

// AuthorizationMiddleware provides HTTP middleware for authorization.
type AuthorizationMiddleware struct {
	rbacService     *RBACService
	jwtManager      *JWTManager
	logger          *slog.Logger
	auditLogger     *slog.Logger
	skipAuthPaths   map[string]bool
	permissionCache map[string][]Permission
	cacheExpiry     time.Duration
}

// NewAuthorizationMiddleware creates a new authorization middleware.
func NewAuthorizationMiddleware(
	rbacService *RBACService,
	jwtManager *JWTManager,
	logger *slog.Logger,
	auditLogger *slog.Logger,
) *AuthorizationMiddleware {
	skipPaths := map[string]bool{
		"/health":   true,
		"/metrics":  true,
		"/login":    true,
		"/register": true,
		"/reset":    true,
		"/verify":   true,
	}

	return &AuthorizationMiddleware{
		rbacService:     rbacService,
		jwtManager:      jwtManager,
		logger:          logger,
		auditLogger:     auditLogger,
		skipAuthPaths:   skipPaths,
		permissionCache: make(map[string][]Permission),
		cacheExpiry:     2 * time.Minute,
	}
}

// AuthContext represents the authentication context extracted from request.
type AuthContext struct { //nolint:revive
	UserID         uint
	Email          string
	Username       string
	OrganizationID *uint
	Roles          []string
	Permissions    []Permission
	IsAdmin        bool
	Token          string
	RequestID      string
}

// RequireAuth is a middleware that requires authentication.
func (m *AuthorizationMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for certain paths.
		if m.skipAuthPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}

		// Extract and validate token
		authCtx, err := m.extractAuthContext(r)
		if err != nil {
			m.logger.Warn("Authentication failed",
				"error", err,
				"path", r.URL.Path,
				"method", r.Method,
				"ip", m.getClientIP(r))

			m.writeUnauthorizedResponse(w, "authentication required")
			return
		}

		// Add auth context to request
		ctx := context.WithValue(r.Context(), authCtxKey, authCtx)
		r = r.WithContext(ctx)

		// Log successful authentication
		m.auditLogger.Info("Authentication successful",
			"user_id", authCtx.UserID,
			"email", authCtx.Email,
			"path", r.URL.Path,
			"method", r.Method,
			"ip", m.getClientIP(r))

		next.ServeHTTP(w, r)
	})
}

// RequirePermission creates middleware that requires specific permissions.
func (m *AuthorizationMiddleware) RequirePermission(permissions ...Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx, err := m.getAuthContext(r)
			if err != nil {
				m.writeUnauthorizedResponse(w, "authentication required")
				return
			}

			// Get organization ID from request if present
			orgID := m.extractOrganizationID(r)

			// Check if user has any of the required permissions
			hasPermission, err := m.rbacService.HasAnyPermission(r.Context(), authCtx.UserID, permissions, orgID)
			if err != nil {
				m.logger.Error("Permission check failed",
					"error", err,
					"user_id", authCtx.UserID,
					"permissions", permissions,
					"org_id", orgID)
				m.writeErrorResponse(w, "permission check failed", http.StatusInternalServerError)
				return
			}

			if !hasPermission {
				m.auditLogger.Warn("Permission denied",
					"user_id", authCtx.UserID,
					"email", authCtx.Email,
					"required_permissions", permissions,
					"org_id", orgID,
					"path", r.URL.Path,
					"method", r.Method,
					"ip", m.getClientIP(r))

				m.writeUnauthorizedResponse(w, "insufficient permissions")
				return
			}

			// Log successful authorization
			m.auditLogger.Info("Authorization successful",
				"user_id", authCtx.UserID,
				"email", authCtx.Email,
				"permissions", permissions,
				"org_id", orgID,
				"path", r.URL.Path,
				"method", r.Method)

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole creates middleware that requires specific roles.
func (m *AuthorizationMiddleware) RequireRole(roleNames ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx, err := m.getAuthContext(r)
			if err != nil {
				m.writeUnauthorizedResponse(w, "authentication required")
				return
			}

			// Get organization ID from request if present
			orgID := m.extractOrganizationID(r)

			// Get user roles
			userRoles, err := m.rbacService.GetUserRoles(r.Context(), authCtx.UserID, orgID)
			if err != nil {
				m.logger.Error("Failed to get user roles",
					"error", err,
					"user_id", authCtx.UserID,
					"org_id", orgID)
				m.writeErrorResponse(w, "failed to check roles", http.StatusInternalServerError)
				return
			}

			// Check if user has any of the required roles
			hasRole := false
			for _, userRole := range userRoles {
				for _, requiredRole := range roleNames {
					if userRole.Name == requiredRole {
						hasRole = true
						break
					}
				}
				if hasRole {
					break
				}
			}

			if !hasRole {
				m.auditLogger.Warn("Role access denied",
					"user_id", authCtx.UserID,
					"email", authCtx.Email,
					"required_roles", roleNames,
					"user_roles", func() []string {
						var names []string
						for _, role := range userRoles {
							names = append(names, role.Name)
						}
						return names
					}(),
					"org_id", orgID,
					"path", r.URL.Path,
					"method", r.Method,
					"ip", m.getClientIP(r))

				m.writeUnauthorizedResponse(w, "insufficient role permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireResourceAccess creates middleware for resource-level authorization.
func (m *AuthorizationMiddleware) RequireResourceAccess(resource string, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCtx, err := m.getAuthContext(r)
			if err != nil {
				m.writeUnauthorizedResponse(w, "authentication required")
				return
			}

			// Get organization ID from request if present
			orgID := m.extractOrganizationID(r)

			// Create authorization context
			authorizationCtx := &AuthorizationContext{
				UserID:         authCtx.UserID,
				OrganizationID: orgID,
				Resource:       resource,
				Action:         action,
				RequestContext: r.Context(),
			}

			// Check resource access
			canAccess, err := m.rbacService.CanAccessResource(r.Context(), authorizationCtx)
			if err != nil {
				m.logger.Error("Resource access check failed",
					"error", err,
					"user_id", authCtx.UserID,
					"resource", resource,
					"action", action,
					"org_id", orgID)
				m.writeErrorResponse(w, "access check failed", http.StatusInternalServerError)
				return
			}

			if !canAccess {
				m.auditLogger.Warn("Resource access denied",
					"user_id", authCtx.UserID,
					"email", authCtx.Email,
					"resource", resource,
					"action", action,
					"org_id", orgID,
					"path", r.URL.Path,
					"method", r.Method,
					"ip", m.getClientIP(r))

				m.writeUnauthorizedResponse(w, "access denied to resource")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAdmin creates middleware that requires admin privileges.
func (m *AuthorizationMiddleware) RequireAdmin() func(http.Handler) http.Handler {
	return m.RequirePermission(PermissionSystemAdmin)
}

// RequireOrganizationAdmin creates middleware that requires organization admin privileges.
func (m *AuthorizationMiddleware) RequireOrganizationAdmin() func(http.Handler) http.Handler {
	return m.RequirePermission(PermissionOrganizationAdmin)
}

// Helper methods.

func (m *AuthorizationMiddleware) extractAuthContext(r *http.Request) (*AuthContext, error) {
	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, fmt.Errorf("missing authorization header")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader {
		return nil, fmt.Errorf("invalid authorization header format")
	}

	// Validate and parse token
	claims, err := m.jwtManager.ValidateToken(tokenString, AccessToken)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Create auth context
	authCtx := &AuthContext{
		UserID:    claims.UserID,
		Email:     claims.Email,
		Username:  claims.Username,
		Roles:     claims.Roles,
		Token:     tokenString,
		RequestID: r.Header.Get("X-Request-ID"),
	}

	// Convert string permissions to Permission type
	for _, permStr := range claims.Permissions {
		authCtx.Permissions = append(authCtx.Permissions, Permission(permStr))
	}

	// Check if user is admin
	for _, perm := range authCtx.Permissions {
		if perm == PermissionSystemAdmin {
			authCtx.IsAdmin = true
			break
		}
	}

	// Extract organization from claims or request
	if len(claims.Organizations) > 0 {
		// Use first organization from claims
		orgID := uint(claims.Organizations[0])
		authCtx.OrganizationID = &orgID
	}

	return authCtx, nil
}

func (m *AuthorizationMiddleware) getAuthContext(r *http.Request) (*AuthContext, error) {
	authCtx, ok := r.Context().Value(authCtxKey).(*AuthContext)
	if !ok {
		return nil, fmt.Errorf("authentication context not found")
	}
	return authCtx, nil
}

func (m *AuthorizationMiddleware) extractOrganizationID(r *http.Request) *uint {
	// Try to get organization ID from URL path
	if orgIDStr := r.URL.Query().Get("org_id"); orgIDStr != "" {
		if orgID, err := strconv.ParseUint(orgIDStr, 10, 32); err == nil {
			id := uint(orgID)
			return &id
		}
	}

	// Try to get from header
	if orgIDStr := r.Header.Get("X-Organization-ID"); orgIDStr != "" {
		if orgID, err := strconv.ParseUint(orgIDStr, 10, 32); err == nil {
			id := uint(orgID)
			return &id
		}
	}

	// Try to get from auth context
	if authCtx, err := m.getAuthContext(r); err == nil && authCtx.OrganizationID != nil {
		return authCtx.OrganizationID
	}

	return nil
}

func (m *AuthorizationMiddleware) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in case of multiple
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		ip = ip[:colon]
	}
	return ip
}

func (m *AuthorizationMiddleware) writeUnauthorizedResponse(w http.ResponseWriter, message string) {
	m.writeErrorResponse(w, message, http.StatusUnauthorized)
}

func (m *AuthorizationMiddleware) writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := fmt.Sprintf(`{"error": "%s", "status": %d}`, message, statusCode)
	if _, err := w.Write([]byte(response)); err != nil {
		// Log error but don't return it as the HTTP response has already started
		m.logger.Error("Failed to write error response", "error", err)
	}
}

// GetAuthContext extracts the authentication context from a request.
func GetAuthContext(r *http.Request) (*AuthContext, error) {
	authCtx, ok := r.Context().Value(authCtxKey).(*AuthContext)
	if !ok {
		return nil, fmt.Errorf("authentication context not found")
	}
	return authCtx, nil
}

// HasPermissionInContext checks if the current request context has a specific permission.
func HasPermissionInContext(r *http.Request, permission Permission) bool {
	authCtx, err := GetAuthContext(r)
	if err != nil {
		return false
	}

	for _, perm := range authCtx.Permissions {
		if perm == permission {
			return true
		}
	}

	return authCtx.IsAdmin // Admin has all permissions
}

// GetUserIDFromContext extracts the user ID from the request context.
func GetUserIDFromContext(r *http.Request) (uint, error) {
	authCtx, err := GetAuthContext(r)
	if err != nil {
		return 0, err
	}
	return authCtx.UserID, nil
}

// GetOrganizationIDFromContext extracts the organization ID from the request context.
func GetOrganizationIDFromContext(r *http.Request) (*uint, error) {
	authCtx, err := GetAuthContext(r)
	if err != nil {
		return nil, err
	}
	return authCtx.OrganizationID, nil
}
