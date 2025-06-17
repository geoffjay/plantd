// Package handlers provides HTTP request handlers for the app service.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/geoffjay/plantd/app/internal/services"

	"github.com/gofiber/fiber/v2"
	log "github.com/sirupsen/logrus"
)

// SSEHandler handles Server-Sent Events for real-time updates.
type SSEHandler struct {
	brokerService  *services.BrokerService
	healthService  *services.HealthService
	metricsService *services.MetricsService
}

// NewSSEHandler creates a new SSE handler.
func NewSSEHandler(
	brokerService *services.BrokerService,
	healthService *services.HealthService,
	metricsService *services.MetricsService,
) *SSEHandler {
	return &SSEHandler{
		brokerService:  brokerService,
		healthService:  healthService,
		metricsService: metricsService,
	}
}

// DashboardSSE streams real-time dashboard updates via Server-Sent Events.
func (s *SSEHandler) DashboardSSE(c *fiber.Ctx) error {
	logger := log.WithField("handler", "sse.dashboard")
	logger.Debug("Starting SSE stream for dashboard updates")

	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Access-Control-Allow-Origin", "*")

	ctx := c.UserContext()

	// Create a ticker for periodic updates
	ticker := time.NewTicker(5 * time.Second) // Update every 5 seconds
	defer ticker.Stop()

	// Send initial data
	if err := s.sendDashboardUpdate(c, ctx); err != nil {
		logger.WithError(err).Error("Failed to send initial dashboard update")
		return err
	}

	// Send periodic updates
	for {
		select {
		case <-ctx.Done():
			logger.Debug("SSE client disconnected")
			return nil
		case <-ticker.C:
			if err := s.sendDashboardUpdate(c, ctx); err != nil {
				logger.WithError(err).Warn("Failed to send dashboard update")
				return err
			}
		}
	}
}

// SystemStatusSSE streams system status updates.
func (s *SSEHandler) SystemStatusSSE(c *fiber.Ctx) error {
	logger := log.WithField("handler", "sse.system_status")
	logger.Debug("Starting SSE stream for system status updates")

	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Access-Control-Allow-Origin", "*")

	ctx := c.UserContext()

	// Create a ticker for quick status updates
	ticker := time.NewTicker(2 * time.Second) // Update every 2 seconds
	defer ticker.Stop()

	// Send initial status
	if err := s.sendSystemStatusUpdate(c, ctx); err != nil {
		logger.WithError(err).Error("Failed to send initial system status")
		return err
	}

	// Send periodic updates
	for {
		select {
		case <-ctx.Done():
			logger.Debug("SSE client disconnected")
			return nil
		case <-ticker.C:
			if err := s.sendSystemStatusUpdate(c, ctx); err != nil {
				logger.WithError(err).Warn("Failed to send system status update")
				return err
			}
		}
	}
}

// sendDashboardUpdate sends a complete dashboard data update.
func (s *SSEHandler) sendDashboardUpdate(c *fiber.Ctx, ctx context.Context) error {
	dashboardData := &DashboardUpdateData{
		Timestamp: time.Now(),
	}

	// Get system health
	if s.healthService != nil {
		if systemHealth, err := s.healthService.GetSystemHealth(ctx); err == nil {
			dashboardData.SystemHealth = &HealthUpdate{
				Status:     systemHealth.Overall,
				Uptime:     systemHealth.Uptime.String(),
				Components: len(systemHealth.Components),
			}
		}
	}

	// Get service statuses
	if s.brokerService != nil {
		if services, err := s.brokerService.GetServiceStatuses(ctx); err == nil {
			dashboardData.Services = &ServiceUpdate{
				Count:   len(services),
				Healthy: s.countHealthyServices(services),
			}
		}
	}

	// Get system metrics
	if s.metricsService != nil {
		if metrics, err := s.metricsService.GetSystemMetrics(ctx); err == nil {
			dashboardData.Metrics = &MetricsUpdate{
				RequestRate:  metrics.Performance.RequestRate,
				ResponseTime: metrics.Performance.ResponseTime.Milliseconds(),
				ErrorRate:    metrics.Performance.ErrorRate,
				Memory:       metrics.System.MemoryUsage / 1024 / 1024, // Convert to MB
				CPU:          metrics.System.CPUUsage,
			}
		}
	}

	return s.sendSSEEvent(c, "dashboard-update", dashboardData)
}

// sendSystemStatusUpdate sends a quick system status update.
func (s *SSEHandler) sendSystemStatusUpdate(c *fiber.Ctx, ctx context.Context) error {
	status := &SystemStatusUpdate{
		Timestamp: time.Now(),
		Status:    "unknown",
		Services:  0,
		Healthy:   0,
	}

	if s.healthService != nil {
		if systemHealth, err := s.healthService.GetSystemHealth(ctx); err == nil {
			status.Status = systemHealth.Overall
		}
	}

	if s.brokerService != nil {
		if services, err := s.brokerService.GetServiceStatuses(ctx); err == nil {
			status.Services = len(services)
			status.Healthy = s.countHealthyServices(services)
		}
	}

	return s.sendSSEEvent(c, "status-update", status)
}

// sendSSEEvent sends an SSE event with the given event type and data.
func (s *SSEHandler) sendSSEEvent(c *fiber.Ctx, eventType string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal SSE data: %w", err)
	}

	sseMessage := fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, jsonData)

	if _, err := c.Response().BodyWriter().Write([]byte(sseMessage)); err != nil {
		return fmt.Errorf("failed to write SSE message: %w", err)
	}

	// Flush the response to ensure immediate delivery
	if flusher, ok := c.Response().BodyWriter().(interface{ Flush() }); ok {
		flusher.Flush()
	}

	return nil
}

// countHealthyServices counts the number of healthy services.
func (s *SSEHandler) countHealthyServices(services []services.ServiceStatus) int {
	count := 0
	for _, service := range services {
		if service.Status == "healthy" {
			count++
		}
	}
	return count
}

// DashboardUpdateData represents the data structure for dashboard SSE updates.
type DashboardUpdateData struct {
	Timestamp    time.Time      `json:"timestamp"`
	SystemHealth *HealthUpdate  `json:"system_health,omitempty"`
	Services     *ServiceUpdate `json:"services,omitempty"`
	Metrics      *MetricsUpdate `json:"metrics,omitempty"`
}

// HealthUpdate represents health status update data.
type HealthUpdate struct {
	Status     string `json:"status"`
	Uptime     string `json:"uptime"`
	Components int    `json:"components"`
}

// ServiceUpdate represents service status update data.
type ServiceUpdate struct {
	Count   int `json:"count"`
	Healthy int `json:"healthy"`
}

// MetricsUpdate represents metrics update data.
type MetricsUpdate struct {
	RequestRate  float64 `json:"request_rate"`
	ResponseTime int64   `json:"response_time_ms"`
	ErrorRate    float64 `json:"error_rate"`
	Memory       uint64  `json:"memory_mb"`
	CPU          float64 `json:"cpu_percent"`
}

// SystemStatusUpdate represents quick system status updates.
type SystemStatusUpdate struct {
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"`
	Services  int       `json:"services"`
	Healthy   int       `json:"healthy"`
}
