package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
)

// SessionStore app wide session store.
var SessionStore *session.Store

// Index renders the application index page.
//
//	@Summary     Index page
//	@Description The application index page
//	@Tags        pages
func Index(c *fiber.Ctx) error {
	// Check if user is authenticated
	if user := c.Locals("user"); user != nil {
		// User is authenticated, redirect to dashboard
		return c.Redirect("/dashboard")
	}

	// User is not authenticated, redirect to login
	return c.Redirect("/login")
}
