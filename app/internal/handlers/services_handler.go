// Package handlers provides HTTP request handlers for the app service.
package handlers

import (
	"context"

	"github.com/geoffjay/plantd/app/internal/auth"
	"github.com/geoffjay/plantd/app/internal/services"
	"github.com/geoffjay/plantd/app/views/pages"

	"github.com/gofiber/fiber/v2"
	log "github.com/sirupsen/logrus"
)

// ServicesHandler handles service management related HTTP requests.
type ServicesHandler struct {
	brokerService *services.BrokerService
	stateService  *services.StateService
	healthService *services.HealthService
}

// NewServicesHandler creates a new services handler.
func NewServicesHandler(
	brokerService *services.BrokerService,
	stateService *services.StateService,
	healthService *services.HealthService,
) *ServicesHandler {
	return &ServicesHandler{
		brokerService: brokerService,
		stateService:  stateService,
		healthService: healthService,
	}
}

// ShowServices renders the services management page.
func (sh *ServicesHandler) ShowServices(c *fiber.Ctx) error {
	logger := log.WithField("handler", "services.show")
	logger.Debug("Rendering services page")

	ctx := context.Background()

	// Get user context
	var user *auth.UserContext
	if userInterface := c.Locals("user"); userInterface != nil {
		if userCtx, ok := userInterface.(*auth.UserContext); ok {
			user = userCtx
		}
	}

	// Get services list
	var servicesList []interface{}

	if sh.brokerService != nil {
		if services, err := sh.brokerService.GetServiceStatuses(ctx); err == nil {
			for _, service := range services {
				servicesList = append(servicesList, service)
			}
		} else {
			logger.WithError(err).Warn("Failed to get service statuses")
		}
	}

	// Prepare data for template
	servicesData := &pages.ServicesData{
		User:         user,
		Services:     servicesList,
		ServiceCount: len(servicesList),
		Filter:       c.Query("filter", "all"),
		Sort:         c.Query("sort", "name"),
	}

	return pages.Services(servicesData).Render(c.Context(), c.Response().BodyWriter())
}

// GetServicesAPI returns services list as JSON.
func (sh *ServicesHandler) GetServicesAPI(c *fiber.Ctx) error {
	logger := log.WithField("handler", "services.api_list")
	logger.Debug("Getting services list via API")

	ctx := context.Background()
	filter := c.Query("filter", "all")
	sortBy := c.Query("sort", "name")

	var servicesList []interface{}

	if sh.brokerService != nil {
		if services, err := sh.brokerService.GetServiceStatuses(ctx); err == nil {
			// Apply filtering
			for _, service := range services {
				if sh.shouldIncludeService(service, filter) {
					servicesList = append(servicesList, service)
				}
			}
		} else {
			logger.WithError(err).Warn("Failed to get service statuses")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   "Failed to retrieve services",
			})
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"services": servicesList,
			"count":    len(servicesList),
			"filter":   filter,
			"sort":     sortBy,
		},
	})
}

// RestartService handles service restart requests.
func (sh *ServicesHandler) RestartService(c *fiber.Ctx) error {
	logger := log.WithField("handler", "services.restart")
	serviceName := c.Params("name")
	logger.WithField("service", serviceName).Info("Service restart requested")

	// Check permissions
	if !sh.hasServicePermission(c, "restart", serviceName) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "Insufficient permissions to restart service",
		})
	}

	ctx := context.Background()

	// Attempt to restart service via broker
	if sh.brokerService != nil {
		if err := sh.brokerService.RestartService(ctx, serviceName); err != nil {
			logger.WithError(err).Error("Failed to restart service")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"success": false,
				"error":   "Failed to restart service: " + err.Error(),
			})
		}
	} else {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"success": false,
			"error":   "Broker service unavailable",
		})
	}

	logger.WithField("service", serviceName).Info("Service restart initiated")
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Service restart initiated",
	})
}

// Helper methods

// shouldIncludeService checks if a service should be included based on filter.
func (sh *ServicesHandler) shouldIncludeService(service services.ServiceStatus, filter string) bool {
	switch filter {
	case "healthy":
		return service.Status == "healthy"
	case "unhealthy":
		return service.Status == "unhealthy" || service.Status == "degraded"
	case "running":
		return service.Status != "stopped"
	case "stopped":
		return service.Status == "stopped"
	default: // "all"
		return true
	}
}

// hasServicePermission checks if the user has permission to perform the action on the service.
func (sh *ServicesHandler) hasServicePermission(c *fiber.Ctx, action, serviceName string) bool {
	// Get user context
	if userInterface := c.Locals("user"); userInterface != nil {
		if user, ok := userInterface.(*auth.UserContext); ok {
			// Check if user has admin permissions or specific service permissions
			for _, permission := range user.Permissions {
				if permission == "*" || permission == "admin" ||
					permission == "services:"+action || permission == "services:"+action+":"+serviceName {
					return true
				}
			}
		}
	}
	return false
}
