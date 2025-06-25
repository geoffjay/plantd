package handlers

import (
	"github.com/geoffjay/plantd/app/internal/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	log "github.com/sirupsen/logrus"
)

// SessionStore app wide session store.
var SessionStore *session.Store

// SessionManager for checking authentication status
var SessionManager *auth.SessionManager

// Index renders the application index page.
//
//	@Summary     Index page
//	@Description The application index page
//	@Tags        pages
func Index(c *fiber.Ctx) error {
	fields := log.Fields{
		"service": "app",
		"context": "handlers.index",
		"path":    c.Path(),
		"ip":      c.IP(),
	}

	// Check if user is authenticated by checking session directly
	if SessionManager != nil {
		sessionData, err := SessionManager.GetSession(c)
		if err == nil && sessionData != nil && sessionData.UserID > 0 {
			// User is authenticated, redirect to dashboard
			log.WithFields(fields).WithField("user_id", sessionData.UserID).Debug("User authenticated, redirecting to dashboard")
			return c.Redirect("/dashboard")
		}
	}

	// User is not authenticated, redirect to login
	log.WithFields(fields).Debug("User not authenticated, redirecting to login")
	return c.Redirect("/login")
}
