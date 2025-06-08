package handlers

import (
	"net/http"

	"github.com/geoffjay/plantd/app/views"

	"github.com/a-h/templ"
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
	session, err := SessionStore.Get(c)
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	loggedIn, _ := session.Get("loggedIn").(bool)
	if !loggedIn {
		return c.Redirect("/login")
	}

	c.Locals("title", "App")

	return views.Render(c, views.Index(), templ.WithStatus(http.StatusOK))
}
