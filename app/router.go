package main

import (
	"net/http"
	"strings"
	"time"

	cfg "github.com/geoffjay/plantd/app/config"
	_ "github.com/geoffjay/plantd/app/docs"
	"github.com/geoffjay/plantd/app/handlers"
	"github.com/geoffjay/plantd/app/internal/auth"
	internalHandlers "github.com/geoffjay/plantd/app/internal/handlers"
	"github.com/geoffjay/plantd/app/views"
	"github.com/geoffjay/plantd/app/views/pages"
	"github.com/geoffjay/plantd/core/util"

	"github.com/a-h/templ"
	"github.com/gofiber/adaptor/v2"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/csrf"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/swagger"
	log "github.com/sirupsen/logrus"
)

const (
	// Development represents the development environment name.
	Development = "development"
)

// Global SSE handler for cleanup
var globalSSEHandler *internalHandlers.SSEHandler

func csrfErrorHandler(c *fiber.Ctx, err error) error {
	// Log the error so we can track who is trying to perform CSRF attacks
	// customize this to your needs
	log.WithFields(log.Fields{
		"service": "app",
		"context": "router.csrfErrorHandler",
		"error":   err,
		"ip":      c.IP(),
		"request": c.OriginalURL(),
	}).Error("CSRF Error")

	log.Debugf("ctx: %v", c)

	// check accepted content types
	switch c.Accepts("html", "json") {
	case "json":
		// Return a 403 Forbidden response for JSON requests
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "403 Forbidden",
		})
	case "html":
		c.Locals("title", "Error")
		c.Locals("error", "403 Forbidden")
		c.Locals("errorCode", "403")

		// Return a 403 Forbidden response for HTML requests
		return views.Render(c, pages.Error(),
			templ.WithStatus(http.StatusForbidden))
	default:
		// Return a 403 Forbidden response for all other requests
		return c.Status(fiber.StatusForbidden).SendString("403 Forbidden")
	}
}

func httpHandler(f http.HandlerFunc) http.Handler {
	return http.HandlerFunc(f)
}

// CleanupSSEHandler performs cleanup of SSE connections
func CleanupSSEHandler() {
	if globalSSEHandler != nil {
		log.Info("Cleaning up SSE handler connections")
		globalSSEHandler.CleanupActiveStreams()
	}
}

func initializeRouter(app *fiber.App, authHandlers *handlers.AuthHandlers, authMiddleware *auth.AuthMiddleware, service *service, sessionStore *session.Store) { //nolint:revive
	staticContents := util.Getenv("PLANTD_APP_PUBLIC_PATH", "./app/static")

	csrfConfig := csrf.Config{
		KeyLookup:      "form:_csrf",
		CookieName:     "__Host-csrf",
		CookieSameSite: "Lax",
		CookieSecure:   true,
		CookieHTTPOnly: true,
		ContextKey:     "csrf",
		ErrorHandler:   csrfErrorHandler,
		Expiration:     30 * time.Minute,
	}
	csrfMiddleware := csrf.New(csrfConfig)

	app.Static("/public", staticContents)
	app.Static("/static", staticContents) // Keep both for compatibility

	// Create dashboard handler
	dashboardHandler := internalHandlers.NewDashboardHandler(
		service.brokerService,
		service.stateService,
		service.healthService,
		service.metricsService,
	)

	// Create SSE handler for real-time updates and store globally for cleanup
	sseHandler := internalHandlers.NewSSEHandler(
		service.brokerService,
		service.healthService,
		service.metricsService,
	)
	globalSSEHandler = sseHandler // Store for cleanup

	// Create services handler
	servicesHandler := internalHandlers.NewServicesHandler(
		service.brokerService,
		service.stateService,
		service.healthService,
	)

	// Public routes
	app.Get("/", csrfMiddleware, handlers.Index)
	app.Get("/login", csrfMiddleware, authHandlers.LoginPage)
	app.Post("/login", csrfMiddleware, authHandlers.Login)
	app.Get("/logout", authHandlers.Logout)
	app.Post("/register", authHandlers.Register)

	// Protected routes
	app.Get("/dashboard", authMiddleware.RequireAuth(), csrfMiddleware, dashboardHandler.ShowDashboard)
	app.Get("/services", authMiddleware.RequireAuth(), csrfMiddleware, servicesHandler.ShowServices)

	// Real-time update routes (SSE) with timeout middleware
	app.Get("/dashboard/sse", authMiddleware.RequireAuth(), sseTimeoutMiddleware, sseHandler.DashboardSSE)
	app.Get("/system/status/sse", authMiddleware.RequireAuth(), sseTimeoutMiddleware, sseHandler.SystemStatusSSE)

	app.Get("/sse", handlers.ReloadSSE)

	// API routes
	api := app.Group("/api")
	api.Post("/auth/login", authHandlers.Login)
	api.Post("/auth/logout", authHandlers.Logout)
	api.Post("/auth/refresh", authHandlers.RefreshToken)
	api.Get("/auth/profile", authMiddleware.RequireAuth(), authHandlers.UserProfile)
	api.Get("/docs/*", swagger.HandlerDefault)

	// Dashboard API routes
	api.Get("/dashboard/data", authMiddleware.RequireAuth(), dashboardHandler.GetDashboardData)
	api.Get("/system/status", authMiddleware.RequireAuth(), dashboardHandler.GetSystemStatus)

	// Services API routes
	api.Get("/services", authMiddleware.RequireAuth(), servicesHandler.GetServicesAPI)
	api.Post("/services/:name/restart", authMiddleware.RequireAuth(), servicesHandler.RestartService)

	v1 := api.Group("/v1", func(c *fiber.Ctx) error {
		c.Set("Version", "v1")
		return c.Next()
	})

	initializeBrokerRoutes(&v1)
	initializeDevRoutes(app)

	app.Use(handlers.NotFound)
}

// sseTimeoutMiddleware adds timeout handling for SSE endpoints
func sseTimeoutMiddleware(c *fiber.Ctx) error {
	// Set reasonable timeouts for SSE connections
	c.Set("X-Accel-Buffering", "no") // Disable nginx buffering
	c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Set("Pragma", "no-cache")
	c.Set("Expires", "0")

	return c.Next()
}

func initializeBrokerRoutes(app *fiber.Router) {
	// TODO: this is just here until the API is implemented.
	defaultHandler := func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	}

	broker := (*app).Group("/broker")
	broker.Get("/status", defaultHandler)
	broker.Get("/errors", defaultHandler)
	broker.Get("/workers", defaultHandler)
	broker.Get("/workers/:id", defaultHandler)
	broker.Get("/info", defaultHandler)
}

func initializeDevRoutes(app *fiber.App) {
	config := cfg.GetConfig()
	if strings.ToLower(config.Env) == Development {
		log.Debug("Development routes enabled")

		dev := (*app).Group("/dev")
		dev.Get("/reload", adaptor.HTTPHandler(httpHandler(handlers.Reload)))
		dev.Use("/reload2", handlers.UpgradeWS)
		dev.Get("/reload2", websocket.New(handlers.ReloadWS))

		// dev.Get("/connections", func(c *fiber.Ctx) error {
		//     m := map[string]any{
		// 	    "open-connections": app.Server().GetOpenConnectionsCount(),
		// 	    "sessions":         len(currentSessions.sessions),
		//     }
		//     return c.JSON(m)
		//    })
	}
}
