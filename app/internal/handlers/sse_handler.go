// Package handlers provides HTTP request handlers for the app service.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
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
	mu             sync.RWMutex
	activeStreams  map[string]context.CancelFunc

	// Circuit breaker state
	serviceFailures int64
	lastFailureTime time.Time
	circuitOpen     bool
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
		activeStreams:  make(map[string]context.CancelFunc),
	}
}

// DashboardSSE handles Server-Sent Events for dashboard updates.
func (h *SSEHandler) DashboardSSE(c *fiber.Ctx) error {
	// Temporarily disable SSE to prevent ZeroMQ CGO signal stack corruption
	return c.Status(503).JSON(fiber.Map{
		"error":   "SSE temporarily disabled",
		"message": "Server-Sent Events are temporarily disabled due to ZeroMQ stability issues",
		"code":    503,
	})
}

// SystemStatusSSE streams system status updates.
func (h *SSEHandler) SystemStatusSSE(c *fiber.Ctx) error {
	logger := log.WithField("handler", "sse.system_status")
	logger.Debug("Starting SSE stream for system status updates")

	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Access-Control-Allow-Origin", "*")
	c.Set("Access-Control-Allow-Headers", "Cache-Control")

	// Generate unique stream ID
	streamID := fmt.Sprintf("status-%d", time.Now().UnixNano())

	// Create cancellable context for this stream
	ctx, cancel := context.WithCancel(c.UserContext())

	// Register this stream
	h.mu.Lock()
	h.activeStreams[streamID] = cancel
	h.mu.Unlock()

	// Cleanup function
	defer func() {
		logger.Debug("Cleaning up SSE status stream")
		h.mu.Lock()
		delete(h.activeStreams, streamID)
		h.mu.Unlock()
		cancel()
	}()

	// Create tickers
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	keepAliveTicker := time.NewTicker(30 * time.Second)
	defer keepAliveTicker.Stop()

	// Send initial status
	if err := h.sendSystemStatusUpdateSimple(c, ctx); err != nil {
		logger.WithError(err).Warn("Failed to send initial system status")
	}

	// Main event loop
	for {
		select {
		case <-ctx.Done():
			logger.Debug("SSE status client disconnected")
			return nil
		case <-ticker.C:
			if err := h.sendSystemStatusUpdateSimple(c, ctx); err != nil {
				logger.WithError(err).Debug("Failed to send system status update, client likely disconnected")
				return nil
			}
		case <-keepAliveTicker.C:
			// Send keep-alive comment
			if _, err := c.WriteString(": keep-alive\n\n"); err != nil {
				logger.WithError(err).Debug("Failed to send keep-alive, client disconnected")
				return nil
			}
		}
	}
}

// isCircuitOpen checks if the circuit breaker is open
func (h *SSEHandler) isCircuitOpen() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// If circuit is open and it's been more than 30 seconds, try to close it
	if h.circuitOpen && time.Since(h.lastFailureTime) > 30*time.Second {
		h.circuitOpen = false
		atomic.StoreInt64(&h.serviceFailures, 0)
		log.Info("Circuit breaker closed, attempting to reconnect services")
		return false
	}

	return h.circuitOpen
}

// recordFailure records a service failure and potentially opens the circuit
func (h *SSEHandler) recordFailure() { //nolint:unused
	failures := atomic.AddInt64(&h.serviceFailures, 1)

	h.mu.Lock()
	h.lastFailureTime = time.Now()

	// Open circuit after 5 failures
	if failures >= 5 && !h.circuitOpen {
		h.circuitOpen = true
		log.Warn("Circuit breaker opened due to repeated service failures")
	}
	h.mu.Unlock()
}

// recordSuccess resets failure count
func (h *SSEHandler) recordSuccess() { //nolint:unused
	atomic.StoreInt64(&h.serviceFailures, 0)
}

// sendDashboardUpdateSimple sends dashboard data using simpler approach
func (h *SSEHandler) sendDashboardUpdateSimple(c *fiber.Ctx, ctx context.Context) error { //nolint:revive, unused
	// Check circuit breaker
	if h.isCircuitOpen() {
		// Send minimal data when circuit is open
		data := &DashboardUpdateData{
			Timestamp: time.Now(),
		}
		return h.sendSSEEventSimple(c, "dashboard-update", data)
	}

	// Create timeout context for service calls
	timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	dashboardData := &DashboardUpdateData{
		Timestamp: time.Now(),
	}

	hasError := false

	// Get system health with error handling
	if h.healthService != nil {
		systemHealth, err := h.healthService.GetSystemHealth(timeoutCtx)
		if err != nil {
			log.WithError(err).Debug("Failed to get system health")
			hasError = true
		} else {
			dashboardData.SystemHealth = &HealthUpdate{
				Status:     systemHealth.Overall,
				Uptime:     systemHealth.Uptime.String(),
				Components: len(systemHealth.Components),
			}
		}
	}

	// Get service statuses with error handling
	if h.brokerService != nil {
		services, err := h.brokerService.GetServiceStatuses(timeoutCtx)
		if err != nil {
			log.WithError(err).Debug("Failed to get service statuses")
			hasError = true
		} else {
			dashboardData.Services = &ServiceUpdate{
				Count:   len(services),
				Healthy: h.countHealthyServices(services),
			}
		}
	}

	// Get system metrics with error handling
	if h.metricsService != nil {
		metrics, err := h.metricsService.GetSystemMetrics(timeoutCtx)
		if err != nil {
			log.WithError(err).Debug("Failed to get system metrics")
			hasError = true
		} else {
			dashboardData.Metrics = &MetricsUpdate{
				RequestRate:  metrics.Performance.RequestRate,
				ResponseTime: metrics.Performance.ResponseTime.Milliseconds(),
				ErrorRate:    metrics.Performance.ErrorRate,
				Memory:       metrics.System.MemoryUsage / 1024 / 1024, // Convert to MB
				CPU:          metrics.System.CPUUsage,
			}
		}
	}

	// Update circuit breaker state
	if hasError {
		h.recordFailure()
	}

	return h.sendSSEEventSimple(c, "dashboard-update", dashboardData)
}

// sendSystemStatusUpdateSimple sends status data using simpler approach
func (h *SSEHandler) sendSystemStatusUpdateSimple(c *fiber.Ctx, ctx context.Context) error { //nolint:revive
	// Check circuit breaker
	if h.isCircuitOpen() {
		// Send minimal status when circuit is open
		status := &SystemStatusUpdate{
			Timestamp: time.Now(),
			Status:    "degraded",
			Services:  0,
			Healthy:   0,
		}
		return h.sendSSEEventSimple(c, "status-update", status)
	}

	// Create timeout context for service calls
	timeoutCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	status := &SystemStatusUpdate{
		Timestamp: time.Now(),
		Status:    "unknown",
		Services:  0,
		Healthy:   0,
	}

	// Get system health with error handling
	if h.healthService != nil {
		systemHealth, err := h.healthService.GetSystemHealth(timeoutCtx)
		if err != nil {
			log.WithError(err).Debug("Failed to get system health for status")
		} else {
			status.Status = systemHealth.Overall
		}
	}

	// Get service statuses with error handling
	if h.brokerService != nil {
		services, err := h.brokerService.GetServiceStatuses(timeoutCtx)
		if err != nil {
			log.WithError(err).Debug("Failed to get service statuses for status")
		} else {
			status.Services = len(services)
			status.Healthy = h.countHealthyServices(services)
		}
	}

	return h.sendSSEEventSimple(c, "status-update", status)
}

// sendSSEEventSimple sends an SSE event using simple string writes
func (h *SSEHandler) sendSSEEventSimple(c *fiber.Ctx, eventType string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal SSE data: %w", err)
	}

	sseMessage := fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, jsonData)

	if _, err := c.WriteString(sseMessage); err != nil {
		return fmt.Errorf("failed to write SSE message: %w", err)
	}

	return nil
}

// Legacy StreamWriter methods (keeping for compatibility but using simpler approach)

// sendDashboardUpdateToWriter sends a complete dashboard data update to the writer.
func (h *SSEHandler) sendDashboardUpdateToWriter(w interface{}, ctx context.Context) error { //nolint:revive, unused
	// Redirect to simple implementation
	return fmt.Errorf("streamwriter implementation disabled for stability")
}

// sendSystemStatusUpdateToWriter sends a quick system status update to the writer.
func (h *SSEHandler) sendSystemStatusUpdateToWriter(w interface{}, ctx context.Context) error { //nolint:revive, unused
	// Redirect to simple implementation
	return fmt.Errorf("streamwriter implementation disabled for stability")
}

// sendSSEEventToWriter sends an SSE event with the given event type and data to the writer.
func (h *SSEHandler) sendSSEEventToWriter(w interface{}, eventType string, data interface{}) error { //nolint:revive, unused
	// Redirect to simple implementation
	return fmt.Errorf("streamwriter implementation disabled for stability")
}

// CleanupActiveStreams closes all active SSE streams (for graceful shutdown).
func (h *SSEHandler) CleanupActiveStreams() {
	h.mu.Lock()
	defer h.mu.Unlock()

	log.WithField("count", len(h.activeStreams)).Info("Cleaning up active SSE streams")

	for streamID, cancel := range h.activeStreams {
		log.WithField("stream_id", streamID).Debug("Canceling SSE stream")
		cancel()
	}

	// Clear the map
	h.activeStreams = make(map[string]context.CancelFunc)
}

// Legacy methods for compatibility (now redirects to simple methods)

// sendDashboardUpdate sends a complete dashboard data update.
func (h *SSEHandler) sendDashboardUpdate(c *fiber.Ctx, ctx context.Context) error { //nolint:revive, unused
	return h.sendDashboardUpdateSimple(c, ctx)
}

// sendSystemStatusUpdate sends a quick system status update.
func (h *SSEHandler) sendSystemStatusUpdate(c *fiber.Ctx, ctx context.Context) error { //nolint:revive, unused
	return h.sendSystemStatusUpdateSimple(c, ctx)
}

// sendSSEEvent sends an SSE event with the given event type and data.
func (h *SSEHandler) sendSSEEvent(c *fiber.Ctx, eventType string, data interface{}) error { //nolint:revive, unused
	return h.sendSSEEventSimple(c, eventType, data)
}

// countHealthyServices counts the number of healthy services.
func (h *SSEHandler) countHealthyServices(serviceStatuses []services.ServiceStatus) int {
	count := 0
	for _, service := range serviceStatuses {
		if service.Status == services.StatusHealthy {
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
