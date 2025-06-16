// Package handlers provides HTTP request handlers for the application.
package handlers

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/a-h/templ"
	"github.com/geoffjay/plantd/app/internal/auth"
	"github.com/geoffjay/plantd/app/types"
	"github.com/geoffjay/plantd/app/views"
	"github.com/geoffjay/plantd/app/views/pages"

	"github.com/gofiber/fiber/v2"
	log "github.com/sirupsen/logrus"
)

// AuthHandlers contains authentication-related handlers.
type AuthHandlers struct {
	identityClient *auth.IdentityClient
	sessionManager *auth.SessionManager
}

// NewAuthHandlers creates a new auth handlers instance.
func NewAuthHandlers(identityClient *auth.IdentityClient, sessionManager *auth.SessionManager) *AuthHandlers {
	return &AuthHandlers{
		identityClient: identityClient,
		sessionManager: sessionManager,
	}
}

// LoginPage renders the login page for users.
func (ah *AuthHandlers) LoginPage(c *fiber.Ctx) error {
	fields := log.Fields{
		"service": "app",
		"context": "handlers.login_page",
		"ip":      c.IP(),
	}

	// Check if user is already authenticated
	if sessionData, err := ah.sessionManager.GetSession(c); err == nil && sessionData != nil {
		log.WithFields(fields).Debug("User already authenticated, redirecting to dashboard")

		// Get redirect URL from query parameters
		redirectURL := c.Query("redirect", "/")
		return c.Redirect(redirectURL)
	}

	// Get any error messages from query parameters
	errorMsg := c.Query("error", "")
	successMsg := c.Query("success", "")

	// Prepare page data
	pageData := types.PageData{
		Title:       "Login - PlantD",
		Description: "Login to access your PlantD dashboard",
		Keywords:    "login, plantd, authentication",
		Error:       errorMsg,
		Success:     successMsg,
	}

	log.WithFields(fields).Debug("Rendering login page")

	// Render login page
	// Set context values for template
	c.Locals("csrf", pageData.CSRFToken)
	c.Locals("error", pageData.Error)

	return views.Render(c, pages.Login(), templ.WithStatus(http.StatusOK))
}

// Login handles user login requests.
func (ah *AuthHandlers) Login(c *fiber.Ctx) error {
	fields := log.Fields{
		"service": "app",
		"context": "handlers.login",
		"ip":      c.IP(),
	}

	// Parse form data
	email := strings.TrimSpace(c.FormValue("email"))
	password := c.FormValue("password")
	redirectURL := c.FormValue("redirect")

	if redirectURL == "" {
		redirectURL = "/"
	}

	fields["email"] = email
	fields["redirect_url"] = redirectURL

	// Validate input
	if email == "" || password == "" {
		log.WithFields(fields).Warn("Login attempt with missing credentials")
		return c.Redirect("/login?error=" + url.QueryEscape("Email and password are required"))
	}

	// Check if this is an API request
	if ah.isAPIRequest(c) {
		return ah.handleAPILogin(c, email, password)
	}

	// Authenticate with Identity Service
	tokenPair, userContext, err := ah.identityClient.Login(email, password)
	if err != nil {
		log.WithFields(fields).WithError(err).Warn("Login attempt failed")
		return c.Redirect("/login?error=" + url.QueryEscape("Invalid email or password"))
	}

	// Create session data
	sessionData := &auth.SessionData{
		UserID:        userContext.ID,
		Email:         userContext.Email,
		Username:      userContext.Username,
		Roles:         userContext.Roles,
		Organizations: userContext.Organizations,
		Permissions:   userContext.Permissions,
		AccessToken:   tokenPair.AccessToken,
		RefreshToken:  tokenPair.RefreshToken,
		ExpiresAt:     tokenPair.ExpiresAt,
	}

	// Create session
	err = ah.sessionManager.CreateSession(c, sessionData)
	if err != nil {
		log.WithFields(fields).WithError(err).Error("Failed to create session")
		return c.Redirect("/login?error=" + url.QueryEscape("Login failed, please try again"))
	}

	log.WithFields(fields).WithField("user_id", userContext.ID).Info("User logged in successfully")

	// Redirect to intended destination
	return c.Redirect(redirectURL)
}

// Logout handles user logout requests.
func (ah *AuthHandlers) Logout(c *fiber.Ctx) error {
	fields := log.Fields{
		"service": "app",
		"context": "handlers.logout",
		"ip":      c.IP(),
	}

	// Get current session if exists
	sessionData, _ := ah.sessionManager.GetSession(c)
	if sessionData != nil {
		fields["user_id"] = sessionData.UserID
		fields["email"] = sessionData.Email

		// Invalidate token with Identity Service
		if sessionData.AccessToken != "" {
			if err := ah.identityClient.Logout(sessionData.AccessToken); err != nil {
				log.WithFields(fields).WithError(err).Warn("Failed to logout from identity service")
			}
		}
	}

	// Destroy local session
	err := ah.sessionManager.DestroySession(c)
	if err != nil {
		log.WithFields(fields).WithError(err).Error("Failed to destroy session")
	}

	log.WithFields(fields).Info("User logged out successfully")

	// Check if this is an API request
	if ah.isAPIRequest(c) {
		return c.JSON(fiber.Map{
			"success": true,
			"message": "Logged out successfully",
		})
	}

	// Redirect to login page with success message
	return c.Redirect("/login?success=" + url.QueryEscape("You have been logged out successfully"))
}

// RefreshToken handles token refresh requests (API only).
func (ah *AuthHandlers) RefreshToken(c *fiber.Ctx) error {
	fields := log.Fields{
		"service": "app",
		"context": "handlers.refresh_token",
		"ip":      c.IP(),
	}

	// This endpoint is API-only
	if !ah.isAPIRequest(c) {
		return c.Status(fiber.StatusNotFound).SendString("Not Found")
	}

	// Refresh the session
	err := ah.sessionManager.RefreshSession(c)
	if err != nil {
		log.WithFields(fields).WithError(err).Error("Token refresh failed")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "Token refresh failed",
			"message": "Please login again",
		})
	}

	// Get updated session
	sessionData, err := ah.sessionManager.GetSession(c)
	if err != nil {
		log.WithFields(fields).WithError(err).Error("Failed to get session after refresh")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Internal server error",
		})
	}

	log.WithFields(fields).WithField("user_id", sessionData.UserID).Info("Token refreshed successfully")

	return c.JSON(fiber.Map{
		"success":      true,
		"access_token": sessionData.AccessToken,
		"expires_at":   sessionData.ExpiresAt,
	})
}

// UserProfile returns the current user's profile information.
func (ah *AuthHandlers) UserProfile(c *fiber.Ctx) error {
	fields := log.Fields{
		"service": "app",
		"context": "handlers.user_profile",
	}

	// Get user context (should be set by auth middleware)
	userContext, ok := auth.GetUserContext(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "User not authenticated",
		})
	}

	fields["user_id"] = userContext.ID
	log.WithFields(fields).Debug("Returning user profile")

	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":            userContext.ID,
			"email":         userContext.Email,
			"username":      userContext.Username,
			"roles":         userContext.Roles,
			"organizations": userContext.Organizations,
			"permissions":   userContext.Permissions,
		},
	})
}

// Register handles user registration requests.
// TODO: Implement actual registration with Identity Service in Phase 3
func (ah *AuthHandlers) Register(c *fiber.Ctx) error {
	fields := log.Fields{
		"service": "app",
		"context": "handlers.register",
		"ip":      c.IP(),
	}

	log.WithFields(fields).Info("Registration attempt - redirecting to Identity Service")

	// For now, redirect to Identity Service registration
	// TODO: Implement registration workflow in Phase 3
	return c.Redirect("/identity/register", fiber.StatusTemporaryRedirect)
}

// handleAPILogin handles login requests from API clients.
func (ah *AuthHandlers) handleAPILogin(c *fiber.Ctx, email, password string) error {
	fields := log.Fields{
		"service": "app",
		"context": "handlers.api_login",
		"email":   email,
	}

	// Authenticate with Identity Service
	tokenPair, userContext, err := ah.identityClient.Login(email, password)
	if err != nil {
		log.WithFields(fields).WithError(err).Warn("API login attempt failed")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error":   "Authentication failed",
			"message": "Invalid email or password",
		})
	}

	// Create session data
	sessionData := &auth.SessionData{
		UserID:        userContext.ID,
		Email:         userContext.Email,
		Username:      userContext.Username,
		Roles:         userContext.Roles,
		Organizations: userContext.Organizations,
		Permissions:   userContext.Permissions,
		AccessToken:   tokenPair.AccessToken,
		RefreshToken:  tokenPair.RefreshToken,
		ExpiresAt:     tokenPair.ExpiresAt,
	}

	// Create session
	err = ah.sessionManager.CreateSession(c, sessionData)
	if err != nil {
		log.WithFields(fields).WithError(err).Error("Failed to create session for API login")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Session creation failed",
		})
	}

	log.WithFields(fields).WithField("user_id", userContext.ID).Info("API login successful")

	return c.JSON(fiber.Map{
		"success":      true,
		"message":      "Login successful",
		"access_token": tokenPair.AccessToken,
		"expires_at":   tokenPair.ExpiresAt,
		"user": fiber.Map{
			"id":            userContext.ID,
			"email":         userContext.Email,
			"username":      userContext.Username,
			"roles":         userContext.Roles,
			"organizations": userContext.Organizations,
		},
	})
}

// isAPIRequest checks if a request is from an API client.
func (ah *AuthHandlers) isAPIRequest(c *fiber.Ctx) bool {
	acceptHeader := c.Get("Accept")
	contentType := c.Get("Content-Type")

	return strings.Contains(acceptHeader, "application/json") ||
		strings.Contains(contentType, "application/json") ||
		strings.HasPrefix(c.Path(), "/api/")
}
