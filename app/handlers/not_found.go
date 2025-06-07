package handlers

import (
	"net/http"

	"github.com/geoffjay/plantd/app/views"
	"github.com/geoffjay/plantd/app/views/pages"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
)

// NotFound handles 404 not found errors.
func NotFound(c *fiber.Ctx) error {
	return views.Render(c, pages.NotFound(), templ.WithStatus(http.StatusNotFound))
}
