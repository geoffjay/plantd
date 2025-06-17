// Package auth provides authentication and authorization functionality.
package auth

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	log "github.com/sirupsen/logrus"
)

// AuthMiddleware provides authentication middleware for protected routes.
type AuthMiddleware struct {
	sessionManager *SessionManager
	identityClient *IdentityClient
	excludedPaths  []string
	roleHierarchy  map[string][]string
}

// NewAuthMiddleware creates a new authentication middleware.
func NewAuthMiddleware(sessionManager *SessionManager, identityClient *IdentityClient) *AuthMiddleware {
	// Define role hierarchy for authorization
	roleHierarchy := map[string][]string{
		"admin":  {"admin", "user", "viewer"},
		"user":   {"user", "viewer"},
		"viewer": {"viewer"},
	}

	// Define paths that don't require authentication
	excludedPaths := []string{
		"/api/health",
		"/api/ping",
		"/login",
		"/auth/login",
		"/auth/logout",
		"/static/",
		"/public/",
		"/favicon.ico",
		"/robots.txt",
		"/manifest.json",
	}

	return &AuthMiddleware{
		sessionManager: sessionManager,
		identityClient: identityClient,
		excludedPaths:  excludedPaths,
		roleHierarchy:  roleHierarchy,
	}
}

// RequireAuth returns middleware that requires authentication.
func (am *AuthMiddleware) RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		fields := log.Fields{
			"service": "app",
			"context": "auth_middleware.require_auth",
			"path":    c.Path(),
			"method":  c.Method(),
		}

		// Check if path is excluded from authentication
		if am.isPathExcluded(c.Path()) {
			log.WithFields(fields).Debug("Path excluded from authentication")
			return c.Next()
		}

		// Get session from cookie
		sessionData, err := am.sessionManager.GetSession(c)
		if err != nil {
			log.WithFields(fields).WithError(err).Debug("Authentication required - no valid session")
			return am.handleAuthenticationRequired(c)
		}

		// Validate access token with Identity Service if needed
		userContext, err := am.identityClient.ValidateToken(sessionData.AccessToken)
		if err != nil {
			log.WithFields(fields).WithError(err).Warn("Token validation failed, attempting refresh")

			// Try to refresh token
			if refreshErr := am.sessionManager.RefreshSession(c); refreshErr != nil {
				log.WithFields(fields).WithError(refreshErr).Error("Token refresh failed")
				am.sessionManager.DestroySession(c)
				return am.handleAuthenticationRequired(c)
			}

			// Get updated session after refresh
			sessionData, err = am.sessionManager.GetSession(c)
			if err != nil {
				log.WithFields(fields).WithError(err).Error("Failed to get session after refresh")
				return am.handleAuthenticationRequired(c)
			}

			// Validate refreshed token
			userContext, err = am.identityClient.ValidateToken(sessionData.AccessToken)
			if err != nil {
				log.WithFields(fields).WithError(err).Error("Token validation failed after refresh")
				am.sessionManager.DestroySession(c)
				return am.handleAuthenticationRequired(c)
			}
		}

		// Store user context in request locals for use in handlers
		c.Locals("user", userContext)
		c.Locals("session", sessionData)

		fields["user_id"] = userContext.ID
		fields["email"] = userContext.Email
		log.WithFields(fields).Debug("Authentication successful")

		return c.Next()
	}
}

// RequireRole returns middleware that requires specific roles.
func (am *AuthMiddleware) RequireRole(requiredRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		fields := log.Fields{
			"service":        "app",
			"context":        "auth_middleware.require_role",
			"path":           c.Path(),
			"required_roles": requiredRoles,
		}

		// Get user context from locals (set by RequireAuth middleware)
		userContext, ok := c.Locals("user").(*UserContext)
		if !ok {
			log.WithFields(fields).Error("User context not found in locals")
			return am.handleUnauthorized(c)
		}

		// Check if user has any of the required roles
		hasRole := false
		for _, requiredRole := range requiredRoles {
			if am.userHasRole(userContext, requiredRole) {
				hasRole = true
				break
			}
		}

		if !hasRole {
			fields["user_roles"] = userContext.Roles
			log.WithFields(fields).Warn("User does not have required role")
			return am.handleUnauthorized(c)
		}

		fields["user_id"] = userContext.ID
		fields["user_roles"] = userContext.Roles
		log.WithFields(fields).Debug("Role authorization successful")

		return c.Next()
	}
}

// RequirePermission returns middleware that requires specific permissions.
func (am *AuthMiddleware) RequirePermission(requiredPermissions ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		fields := log.Fields{
			"service":              "app",
			"context":              "auth_middleware.require_permission",
			"path":                 c.Path(),
			"required_permissions": requiredPermissions,
		}

		// Get user context from locals (set by RequireAuth middleware)
		userContext, ok := c.Locals("user").(*UserContext)
		if !ok {
			log.WithFields(fields).Error("User context not found in locals")
			return am.handleUnauthorized(c)
		}

		// Check if user has any of the required permissions
		hasPermission := false
		for _, requiredPermission := range requiredPermissions {
			if am.userHasPermission(userContext, requiredPermission) {
				hasPermission = true
				break
			}
		}

		if !hasPermission {
			fields["user_permissions"] = userContext.Permissions
			log.WithFields(fields).Warn("User does not have required permission")
			return am.handleUnauthorized(c)
		}

		fields["user_id"] = userContext.ID
		fields["user_permissions"] = userContext.Permissions
		log.WithFields(fields).Debug("Permission authorization successful")

		return c.Next()
	}
}

// RequireCSRF returns middleware that validates CSRF tokens for state-changing operations.
func (am *AuthMiddleware) RequireCSRF() fiber.Handler {
	return func(c *fiber.Ctx) error {
		fields := log.Fields{
			"service": "app",
			"context": "auth_middleware.require_csrf",
			"path":    c.Path(),
			"method":  c.Method(),
		}

		// Only check CSRF for state-changing methods
		if c.Method() == "GET" || c.Method() == "HEAD" || c.Method() == "OPTIONS" {
			return c.Next()
		}

		// Get session data from locals (set by RequireAuth middleware)
		sessionData, ok := c.Locals("session").(*SessionData)
		if !ok {
			log.WithFields(fields).Error("Session data not found in locals")
			return am.handleUnauthorized(c)
		}

		// Validate CSRF token
		if err := am.sessionManager.ValidateCSRFToken(c, sessionData); err != nil {
			log.WithFields(fields).WithError(err).Warn("CSRF token validation failed")
			return fiber.NewError(fiber.StatusForbidden, "CSRF token invalid")
		}

		log.WithFields(fields).Debug("CSRF validation successful")

		return c.Next()
	}
}

// GetUserContext returns the authenticated user context from the request.
func GetUserContext(c *fiber.Ctx) (*UserContext, bool) {
	userContext, ok := c.Locals("user").(*UserContext)
	return userContext, ok
}

// GetSessionData returns the session data from the request.
func GetSessionData(c *fiber.Ctx) (*SessionData, bool) {
	sessionData, ok := c.Locals("session").(*SessionData)
	return sessionData, ok
}

// isPathExcluded checks if a path should be excluded from authentication.
func (am *AuthMiddleware) isPathExcluded(path string) bool {
	for _, excludedPath := range am.excludedPaths {
		if strings.HasPrefix(path, excludedPath) {
			return true
		}
	}
	return false
}

// userHasRole checks if a user has a specific role (considering role hierarchy).
func (am *AuthMiddleware) userHasRole(user *UserContext, requiredRole string) bool {
	// Check if user has the exact role
	for _, userRole := range user.Roles {
		if userRole == requiredRole {
			return true
		}

		// Check role hierarchy - if user has a higher role that includes the required role
		if allowedRoles, exists := am.roleHierarchy[userRole]; exists {
			for _, allowedRole := range allowedRoles {
				if allowedRole == requiredRole {
					return true
				}
			}
		}
	}
	return false
}

// userHasPermission checks if a user has a specific permission.
func (am *AuthMiddleware) userHasPermission(user *UserContext, requiredPermission string) bool {
	// Check for wildcard permission (admin access)
	for _, permission := range user.Permissions {
		if permission == "*" {
			return true
		}
		if permission == requiredPermission {
			return true
		}
	}
	return false
}

// handleAuthenticationRequired handles requests that require authentication.
func (am *AuthMiddleware) handleAuthenticationRequired(c *fiber.Ctx) error {
	// Check if this is an API request (JSON content type or Accept header)
	acceptHeader := c.Get("Accept")
	contentType := c.Get("Content-Type")

	if strings.Contains(acceptHeader, "application/json") ||
		strings.Contains(contentType, "application/json") ||
		strings.HasPrefix(c.Path(), "/api/") {
		// Return JSON error for API requests
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "Authentication required",
			"message": "Please login to access this resource",
		})
	}

	// Redirect to login page for web requests
	return c.Redirect("/login?redirect=" + c.OriginalURL())
}

// handleUnauthorized handles unauthorized requests.
func (am *AuthMiddleware) handleUnauthorized(c *fiber.Ctx) error {
	// Check if this is an API request
	acceptHeader := c.Get("Accept")
	contentType := c.Get("Content-Type")

	if strings.Contains(acceptHeader, "application/json") ||
		strings.Contains(contentType, "application/json") ||
		strings.HasPrefix(c.Path(), "/api/") {
		// Return JSON error for API requests
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   "Access denied",
			"message": "You don't have permission to access this resource",
		})
	}

	// Return forbidden page for web requests
	return c.Status(fiber.StatusForbidden).SendString("Access Denied: You don't have permission to access this resource")
}
