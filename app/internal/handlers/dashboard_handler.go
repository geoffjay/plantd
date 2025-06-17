// Package handlers provides HTTP request handlers for the app service.
package handlers

import (
	"context"
	"fmt"

	"github.com/geoffjay/plantd/app/internal/auth"
	"github.com/geoffjay/plantd/app/internal/services"
	"github.com/geoffjay/plantd/app/views/pages"

	"github.com/gofiber/fiber/v2"
	log "github.com/sirupsen/logrus"
)

// DashboardHandler handles dashboard-related HTTP requests.
type DashboardHandler struct {
	brokerService  *services.BrokerService
	stateService   *services.StateService
	healthService  *services.HealthService
	metricsService *services.MetricsService
}

// Use the DashboardData type defined in the template
type DashboardData = pages.DashboardData

// NewDashboardHandler creates a new dashboard handler.
func NewDashboardHandler(
	brokerService *services.BrokerService,
	stateService *services.StateService,
	healthService *services.HealthService,
	metricsService *services.MetricsService,
) *DashboardHandler {
	return &DashboardHandler{
		brokerService:  brokerService,
		stateService:   stateService,
		healthService:  healthService,
		metricsService: metricsService,
	}
}

// ShowDashboard renders the main dashboard page with system overview.
func (dh *DashboardHandler) ShowDashboard(c *fiber.Ctx) error {
	logger := log.WithField("handler", "dashboard.show")
	logger.Debug("Rendering dashboard page")

	// Set content type to HTML
	c.Set("Content-Type", "text/html; charset=utf-8")

	ctx := context.Background()

	// Get user context from session
	var user *auth.UserContext
	if userInterface := c.Locals("user"); userInterface != nil {
		if userCtx, ok := userInterface.(*auth.UserContext); ok {
			user = userCtx
		}
	}

	// If no user context, create a default one (for development)
	if user == nil {
		logger.Warn("No user context found, using default")
		user = &auth.UserContext{
			ID:            1,
			Email:         "admin@plantd.local",
			Username:      "admin",
			Roles:         []string{"admin"},
			Organizations: []string{"plantd"},
			Permissions:   []string{"*"},
		}
	}

	// Collect dashboard data with error handling - convert to template-compatible types
	dashboardData := &pages.DashboardData{
		User:             user,
		ServiceCount:     0,
		HealthStatus:     "unknown",
		RequestRate:      "0/sec",
		Uptime:           "Unknown",
		Services:         make([]interface{}, 0),
		HealthComponents: make(map[string]interface{}),
	}

	// Get system health
	if dh.healthService != nil {
		if systemHealth, err := dh.healthService.GetSystemHealth(ctx); err == nil {
			dashboardData.SystemHealth = systemHealth
			dashboardData.HealthStatus = systemHealth.Overall
			dashboardData.Uptime = systemHealth.Uptime.String()

			// Convert components to interface{} map
			if systemHealth.Components != nil {
				for name, component := range systemHealth.Components {
					dashboardData.HealthComponents[name] = component
				}
			}
		} else {
			logger.WithError(err).Warn("Failed to get system health")
		}
	}

	// Get service statuses
	if dh.brokerService != nil {
		if services, err := dh.brokerService.GetServiceStatuses(ctx); err == nil {
			dashboardData.ServiceCount = len(services)
			// Convert services to interface{} slice
			for _, service := range services {
				dashboardData.Services = append(dashboardData.Services, service)
			}
		} else {
			logger.WithError(err).Warn("Failed to get service statuses")
		}
	}

	// Get system metrics
	if dh.metricsService != nil {
		if metrics, err := dh.metricsService.GetSystemMetrics(ctx); err == nil {
			dashboardData.Metrics = metrics
			dashboardData.PerformanceData = &metrics.Performance
			if metrics.Performance.RequestRate > 0 {
				dashboardData.RequestRate = formatRequestRate(metrics.Performance.RequestRate)
			}
		} else {
			logger.WithError(err).Warn("Failed to get system metrics")
		}
	}

	// Render dashboard using templ
	return pages.Dashboard(dashboardData).Render(c.Context(), c.Response().BodyWriter())
}

// GetDashboardData returns dashboard data as JSON for AJAX updates.
func (dh *DashboardHandler) GetDashboardData(c *fiber.Ctx) error {
	logger := log.WithField("handler", "dashboard.get_data")
	logger.Debug("Getting dashboard data")

	ctx := context.Background()

	// Get user context
	var user *auth.UserContext
	if userInterface := c.Locals("user"); userInterface != nil {
		if userCtx, ok := userInterface.(*auth.UserContext); ok {
			user = userCtx
		}
	}

	dashboardData := &pages.DashboardData{
		User:             user,
		ServiceCount:     0,
		HealthStatus:     "unknown",
		RequestRate:      "0/sec",
		Uptime:           "Unknown",
		Services:         make([]interface{}, 0),
		HealthComponents: make(map[string]interface{}),
	}

	// Collect data from all services
	if dh.healthService != nil {
		if systemHealth, err := dh.healthService.GetSystemHealth(ctx); err == nil {
			dashboardData.SystemHealth = systemHealth
			dashboardData.HealthStatus = systemHealth.Overall
			dashboardData.Uptime = systemHealth.Uptime.String()

			// Convert components to interface{} map
			if systemHealth.Components != nil {
				for name, component := range systemHealth.Components {
					dashboardData.HealthComponents[name] = component
				}
			}
		}
	}

	if dh.brokerService != nil {
		if services, err := dh.brokerService.GetServiceStatuses(ctx); err == nil {
			dashboardData.ServiceCount = len(services)
			// Convert services to interface{} slice
			for _, service := range services {
				dashboardData.Services = append(dashboardData.Services, service)
			}
		}
	}

	if dh.metricsService != nil {
		if metrics, err := dh.metricsService.GetSystemMetrics(ctx); err == nil {
			dashboardData.Metrics = metrics
			dashboardData.PerformanceData = &metrics.Performance
			if metrics.Performance.RequestRate > 0 {
				dashboardData.RequestRate = formatRequestRate(metrics.Performance.RequestRate)
			}
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dashboardData,
	})
}

// GetSystemStatus returns a quick system status check for health monitoring.
func (dh *DashboardHandler) GetSystemStatus(c *fiber.Ctx) error {
	logger := log.WithField("handler", "dashboard.system_status")
	logger.Debug("Getting system status")

	ctx := context.Background()

	status := fiber.Map{
		"status":           "unknown",
		"services":         0,
		"healthy_services": 0,
		"timestamp":        nil,
	}

	if dh.healthService != nil {
		if systemHealth, err := dh.healthService.GetSystemHealth(ctx); err == nil {
			status["status"] = systemHealth.Overall
			status["timestamp"] = systemHealth.Timestamp

			healthyCount := 0
			totalCount := len(systemHealth.Components)
			for _, component := range systemHealth.Components {
				if component.Status == "healthy" {
					healthyCount++
				}
			}
			status["services"] = totalCount
			status["healthy_services"] = healthyCount
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    status,
	})
}

// formatRequestRate formats request rate for display.
func formatRequestRate(rate float64) string {
	if rate < 1 {
		return "< 1/sec"
	}
	return fmt.Sprintf("%.1f/sec", rate)
}
