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

// SSEHandler handles Server-Sent Events for real-time updates using Datastar.
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

// DashboardSSE handles Server-Sent Events for dashboard updates using Datastar.
func (h *SSEHandler) DashboardSSE(c *fiber.Ctx) error {
	logger := log.WithField("handler", "sse.dashboard")
	logger.Debug("Starting Datastar SSE stream for dashboard updates")

	// Set SSE headers for Datastar
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Access-Control-Allow-Origin", "*")
	c.Set("Access-Control-Allow-Headers", "Cache-Control")

	// Generate unique stream ID
	streamID := fmt.Sprintf("dashboard-%d", time.Now().UnixNano())

	// Create cancellable context for this stream
	ctx, cancel := context.WithCancel(c.UserContext())

	// Register this stream
	h.mu.Lock()
	h.activeStreams[streamID] = cancel
	h.mu.Unlock()

	// Cleanup function
	defer func() {
		logger.Debug("Cleaning up Datastar SSE dashboard stream")
		h.mu.Lock()
		delete(h.activeStreams, streamID)
		h.mu.Unlock()
		cancel()
	}()

	// Create tickers
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	keepAliveTicker := time.NewTicker(30 * time.Second)
	defer keepAliveTicker.Stop()

	// Send initial dashboard data
	if err := h.sendDashboardDatastarUpdate(c, ctx); err != nil {
		logger.WithError(err).Warn("Failed to send initial dashboard data")
	}

	// Main event loop
	for {
		select {
		case <-ctx.Done():
			logger.Debug("Datastar SSE dashboard client disconnected")
			return nil
		case <-ticker.C:
			if err := h.sendDashboardDatastarUpdate(c, ctx); err != nil {
				logger.WithError(err).Debug("Failed to send dashboard update, client likely disconnected")
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

// SystemStatusSSE streams system status updates using Datastar.
func (h *SSEHandler) SystemStatusSSE(c *fiber.Ctx) error {
	logger := log.WithField("handler", "sse.system_status")
	logger.Debug("Starting Datastar SSE stream for system status updates")

	// Set SSE headers for Datastar
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
		logger.Debug("Cleaning up Datastar SSE status stream")
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
	if err := h.sendSystemStatusDatastarUpdate(c, ctx); err != nil {
		logger.WithError(err).Warn("Failed to send initial system status")
	}

	// Main event loop
	for {
		select {
		case <-ctx.Done():
			logger.Debug("Datastar SSE status client disconnected")
			return nil
		case <-ticker.C:
			if err := h.sendSystemStatusDatastarUpdate(c, ctx); err != nil {
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

// sendDashboardDatastarUpdate sends dashboard data using Datastar format.
func (h *SSEHandler) sendDashboardDatastarUpdate(c *fiber.Ctx, ctx context.Context) error { //nolint:revive
	// Check circuit breaker
	if h.isCircuitOpen() {
		// Send minimal data when circuit is open
		signals := map[string]interface{}{
			"connectionStatus": "degraded",
			"lastUpdated":      time.Now().Format("15:04:05"),
		}

		return h.sendDatastarMergeSignals(c, signals)
	}

	// Create timeout context for service calls
	timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	signals := map[string]interface{}{
		"connectionStatus": "connected",
		"lastUpdated":      time.Now().Format("15:04:05"),
	}

	hasError := false

	// Get system health with error handling
	if h.healthService != nil {
		systemHealth, err := h.healthService.GetSystemHealth(timeoutCtx)
		if err != nil {
			log.WithError(err).Debug("Failed to get system health")
			hasError = true
		} else {
			signals["healthStatus"] = systemHealth.Overall
			signals["uptime"] = systemHealth.Uptime.String()
		}
	}

	// Get service statuses with error handling
	if h.brokerService != nil {
		services, err := h.brokerService.GetServiceStatuses(timeoutCtx)
		if err != nil {
			log.WithError(err).Debug("Failed to get service statuses")
			hasError = true
		} else {
			signals["serviceCount"] = len(services)
			signals["healthyServices"] = h.countHealthyServices(services)
		}
	}

	// Get system metrics with error handling
	if h.metricsService != nil {
		metrics, err := h.metricsService.GetSystemMetrics(timeoutCtx)
		if err != nil {
			log.WithError(err).Debug("Failed to get system metrics")
			hasError = true
		} else {
			signals["requestRate"] = fmt.Sprintf("%.1f/sec", metrics.Performance.RequestRate)
			signals["responseTime"] = fmt.Sprintf("%dms", metrics.Performance.ResponseTime.Milliseconds())
			signals["errorRate"] = fmt.Sprintf("%.1f%%", metrics.Performance.ErrorRate*100)
			signals["memoryUsage"] = fmt.Sprintf("%.1f MB", float64(metrics.System.MemoryUsage)/1024/1024)
			signals["cpuUsage"] = fmt.Sprintf("%.1f%%", metrics.System.CPUUsage)
		}
	}

	// Update circuit breaker state
	if hasError {
		h.recordFailure()
		signals["connectionStatus"] = "degraded"
	} else {
		h.recordSuccess()
	}

	return h.sendDatastarMergeSignals(c, signals)
}

// sendSystemStatusDatastarUpdate sends status data using Datastar format.
func (h *SSEHandler) sendSystemStatusDatastarUpdate(c *fiber.Ctx, ctx context.Context) error { //nolint:revive
	// Check circuit breaker
	if h.isCircuitOpen() {
		// Send minimal status when circuit is open
		signals := map[string]interface{}{
			"connectionStatus": "degraded",
		}

		return h.sendDatastarMergeSignals(c, signals)
	}

	// Create timeout context for service calls
	timeoutCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	signals := map[string]interface{}{
		"connectionStatus": "connected",
	}

	// Get quick health status
	if h.healthService != nil {
		systemHealth, err := h.healthService.GetSystemHealth(timeoutCtx)
		if err != nil {
			log.WithError(err).Debug("Failed to get system health for status update")
			signals["connectionStatus"] = "degraded"
		} else {
			signals["healthStatus"] = systemHealth.Overall
		}
	}

	return h.sendDatastarMergeSignals(c, signals)
}

// sendDatastarMergeSignals sends signals using Datastar's merge-signals event format.
func (h *SSEHandler) sendDatastarMergeSignals(c *fiber.Ctx, signals map[string]interface{}) error {
	jsonData, err := json.Marshal(signals)
	if err != nil {
		return fmt.Errorf("failed to marshal signals: %w", err)
	}

	// Send Datastar merge-signals event
	sseMessage := fmt.Sprintf("event: datastar-merge-signals\ndata: signals %s\n\n", jsonData)

	if _, err := c.WriteString(sseMessage); err != nil {
		return fmt.Errorf("failed to write SSE message: %w", err)
	}

	return nil
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
