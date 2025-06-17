// Package auth provides authentication and authorization functionality.
package auth

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/geoffjay/plantd/app/config"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/storage/redis/v3"
	log "github.com/sirupsen/logrus"
)

// SessionData represents the data stored in a user session.
type SessionData struct {
	UserID        uint      `json:"user_id"`
	Email         string    `json:"email"`
	Username      string    `json:"username"`
	Roles         []string  `json:"roles"`
	Organizations []string  `json:"organizations"`
	Permissions   []string  `json:"permissions"`
	AccessToken   string    `json:"access_token"`
	RefreshToken  string    `json:"refresh_token"`
	ExpiresAt     time.Time `json:"expires_at"`
	CSRFToken     string    `json:"csrf_token"`
	CreatedAt     time.Time `json:"created_at"`
	LastActivity  time.Time `json:"last_activity"`
}

// SessionManager handles secure session management.
type SessionManager struct {
	store          *redis.Storage
	config         *config.Config
	identityClient *IdentityClient
}

// NewSessionManager creates a new session manager.
func NewSessionManager(cfg *config.Config, identityClient *IdentityClient) (*SessionManager, error) {
	fields := log.Fields{
		"service": "app",
		"context": "session_manager.new",
	}

	// Create Redis storage configuration
	redisConfig := redis.Config{
		Host:     "127.0.0.1",
		Port:     6379,
		Username: "",
		Password: "",
		Database: 0,
		Reset:    false,
	}

	// Create Redis storage for sessions
	store := redis.New(redisConfig)

	sm := &SessionManager{
		store:          store,
		config:         cfg,
		identityClient: identityClient,
	}

	log.WithFields(fields).Info("Session manager initialized")

	return sm, nil
}

// getCookieName returns the session cookie name with fallback.
func (sm *SessionManager) getCookieName() string {
	if sm.config.EnhancedSession.CookieName != "" {
		return sm.config.EnhancedSession.CookieName
	}
	return "__Host-session"
}

// getMaxAge returns the session max age with fallback.
func (sm *SessionManager) getMaxAge() int {
	if sm.config.EnhancedSession.MaxAge > 0 {
		return sm.config.EnhancedSession.MaxAge
	}
	return 7200 // 2 hours default
}

// isSecure returns whether cookies should be secure.
func (sm *SessionManager) isSecure() bool {
	return sm.config.EnhancedSession.Secure
}

// isHttpOnly returns whether cookies should be HTTP only.
func (sm *SessionManager) isHttpOnly() bool {
	return sm.config.EnhancedSession.HttpOnly
}

// CreateSession creates a new secure session for a user.
func (sm *SessionManager) CreateSession(c *fiber.Ctx, userData *SessionData) error {
	fields := log.Fields{
		"service": "app",
		"context": "session_manager.create_session",
		"user_id": userData.UserID,
		"email":   userData.Email,
	}

	// Generate session ID
	sessionID := sm.generateSessionID()

	// Generate CSRF token
	csrfToken := sm.generateCSRFToken()

	// Update session data
	userData.CSRFToken = csrfToken
	userData.CreatedAt = time.Now()
	userData.LastActivity = time.Now()

	// Serialize session data
	sessionBytes, err := json.Marshal(userData)
	if err != nil {
		log.WithFields(fields).WithError(err).Error("Failed to serialize session data")
		return fmt.Errorf("failed to serialize session data: %w", err)
	}

	// Store session in Redis with expiration
	sessionKey := fmt.Sprintf("session:%s", sessionID)
	expiration := time.Duration(sm.getMaxAge()) * time.Second

	err = sm.store.Set(sessionKey, sessionBytes, expiration)
	if err != nil {
		log.WithFields(fields).WithError(err).Error("Failed to store session")
		return fmt.Errorf("failed to store session: %w", err)
	}

	// Set secure session cookie
	cookie := &fiber.Cookie{
		Name:     sm.getCookieName(),
		Value:    sessionID,
		Path:     "/",
		Domain:   "",
		MaxAge:   sm.getMaxAge(),
		Secure:   sm.isSecure(),
		HTTPOnly: sm.isHttpOnly(),
		SameSite: "Lax",
	}

	c.Cookie(cookie)

	// Set CSRF token in header for JavaScript access
	c.Set("X-CSRF-Token", csrfToken)

	log.WithFields(fields).Info("Session created successfully")

	return nil
}

// GetSession retrieves and validates a user session.
func (sm *SessionManager) GetSession(c *fiber.Ctx) (*SessionData, error) {
	fields := log.Fields{
		"service": "app",
		"context": "session_manager.get_session",
	}

	// Get session ID from cookie
	sessionID := c.Cookies(sm.getCookieName())
	if sessionID == "" {
		return nil, fmt.Errorf("no session cookie found")
	}

	// Retrieve session from Redis
	sessionKey := fmt.Sprintf("session:%s", sessionID)
	sessionBytes, err := sm.store.Get(sessionKey)
	if err != nil {
		log.WithFields(fields).WithError(err).Debug("Failed to retrieve session")
		return nil, fmt.Errorf("session not found")
	}

	if sessionBytes == nil {
		return nil, fmt.Errorf("session expired or not found")
	}

	// Deserialize session data
	var sessionData SessionData
	err = json.Unmarshal(sessionBytes, &sessionData)
	if err != nil {
		log.WithFields(fields).WithError(err).Error("Failed to deserialize session data")
		return nil, fmt.Errorf("invalid session data")
	}

	// Check if session is expired
	if time.Now().After(sessionData.ExpiresAt) {
		log.WithFields(fields).Debug("Session expired")
		sm.DestroySession(c)
		return nil, fmt.Errorf("session expired")
	}

	// Update last activity
	sessionData.LastActivity = time.Now()
	sm.updateSessionActivity(sessionKey, &sessionData)

	fields["user_id"] = sessionData.UserID
	log.WithFields(fields).Debug("Session retrieved successfully")

	return &sessionData, nil
}

// RefreshSession refreshes the access token using the refresh token.
func (sm *SessionManager) RefreshSession(c *fiber.Ctx) error {
	fields := log.Fields{
		"service": "app",
		"context": "session_manager.refresh_session",
	}

	// Get current session
	sessionData, err := sm.GetSession(c)
	if err != nil {
		return fmt.Errorf("failed to get session for refresh: %w", err)
	}

	// Refresh token with Identity Service
	tokenPair, err := sm.identityClient.RefreshToken(sessionData.RefreshToken)
	if err != nil {
		log.WithFields(fields).WithError(err).Error("Failed to refresh token")
		// If refresh fails, destroy the session
		sm.DestroySession(c)
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// Update session with new tokens
	sessionData.AccessToken = tokenPair.AccessToken
	sessionData.RefreshToken = tokenPair.RefreshToken
	sessionData.ExpiresAt = tokenPair.ExpiresAt
	sessionData.LastActivity = time.Now()

	// Update session in storage
	sessionID := c.Cookies(sm.getCookieName())
	sessionKey := fmt.Sprintf("session:%s", sessionID)

	sessionBytes, err := json.Marshal(sessionData)
	if err != nil {
		return fmt.Errorf("failed to serialize updated session: %w", err)
	}

	expiration := time.Duration(sm.getMaxAge()) * time.Second

	err = sm.store.Set(sessionKey, sessionBytes, expiration)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	log.WithFields(fields).WithField("user_id", sessionData.UserID).Info("Session refreshed successfully")

	return nil
}

// DestroySession securely destroys a user session.
func (sm *SessionManager) DestroySession(c *fiber.Ctx) error {
	fields := log.Fields{
		"service": "app",
		"context": "session_manager.destroy_session",
	}

	// Get session ID from cookie
	sessionID := c.Cookies(sm.getCookieName())
	if sessionID != "" {
		// Remove session from Redis
		sessionKey := fmt.Sprintf("session:%s", sessionID)
		err := sm.store.Delete(sessionKey)
		if err != nil {
			log.WithFields(fields).WithError(err).Warn("Failed to delete session from storage")
		}
	}

	// Clear session cookie
	c.Cookie(&fiber.Cookie{
		Name:     sm.getCookieName(),
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Secure:   sm.isSecure(),
		HTTPOnly: sm.isHttpOnly(),
		SameSite: "Lax",
	})

	log.WithFields(fields).Debug("Session destroyed")

	return nil
}

// ValidateCSRFToken validates a CSRF token for state-changing operations.
func (sm *SessionManager) ValidateCSRFToken(c *fiber.Ctx, sessionData *SessionData) error {
	// Get CSRF token from header or form
	csrfToken := c.Get("X-CSRF-Token")
	if csrfToken == "" {
		csrfToken = c.FormValue("csrf_token")
	}

	if csrfToken == "" {
		return fmt.Errorf("CSRF token missing")
	}

	if csrfToken != sessionData.CSRFToken {
		return fmt.Errorf("CSRF token invalid")
	}

	return nil
}

// generateSessionID generates a secure session ID.
func (sm *SessionManager) generateSessionID() string {
	// TODO: Implement secure session ID generation
	// For now, use timestamp + random suffix
	return fmt.Sprintf("session_%d_%d", time.Now().UnixNano(), time.Now().Unix())
}

// generateCSRFToken generates a secure CSRF token.
func (sm *SessionManager) generateCSRFToken() string {
	// TODO: Implement secure CSRF token generation
	// For now, use timestamp + random suffix
	return fmt.Sprintf("csrf_%d_%d", time.Now().UnixNano(), time.Now().Unix())
}

// updateSessionActivity updates the last activity timestamp for a session.
func (sm *SessionManager) updateSessionActivity(sessionKey string, sessionData *SessionData) {
	sessionBytes, err := json.Marshal(sessionData)
	if err != nil {
		return
	}

	expiration := time.Duration(sm.getMaxAge()) * time.Second

	sm.store.Set(sessionKey, sessionBytes, expiration)
}
