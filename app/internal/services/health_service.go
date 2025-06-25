// Package services provides business logic for service integrations.
package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/geoffjay/plantd/app/config"
	"github.com/geoffjay/plantd/app/internal/auth"

	log "github.com/sirupsen/logrus"
)

// HealthService aggregates health information from all plantd services and provides comprehensive system health monitoring.
type HealthService struct {
	brokerService  *BrokerService
	stateService   *StateService
	identityClient *auth.IdentityClient
	config         *config.Config
	logger         *log.Entry

	// Health monitoring state
	lastHealthCheck time.Time
	healthHistory   []SystemHealth
	maxHistorySize  int
	alertThresholds map[string]interface{}
}

// SystemHealth represents overall system health status.
type SystemHealth struct {
	Overall    string               `json:"overall"` // "healthy", "degraded", "unhealthy"
	Components map[string]Component `json:"components"`
	Timestamp  time.Time            `json:"timestamp"`
	Uptime     time.Duration        `json:"uptime"`
	Summary    HealthSummary        `json:"summary"`
}

// Component represents the health status of a single component.
type Component struct {
	Status      string            `json:"status"` // "healthy", "degraded", "unhealthy"
	Message     string            `json:"message"`
	Latency     time.Duration     `json:"latency"`
	LastCheck   time.Time         `json:"last_check"`
	Metadata    map[string]string `json:"metadata"`
	ErrorCount  int               `json:"error_count"`
	SuccessRate float64           `json:"success_rate"`
}

// HealthSummary provides aggregated health information.
type HealthSummary struct {
	TotalComponents     int     `json:"total_components"`
	HealthyComponents   int     `json:"healthy_components"`
	DegradedComponents  int     `json:"degraded_components"`
	UnhealthyComponents int     `json:"unhealthy_components"`
	OverallScore        float64 `json:"overall_score"` // 0-100
}

// HealthAlert represents a health alert condition.
type HealthAlert struct {
	ID         string                 `json:"id"`
	Component  string                 `json:"component"`
	Severity   string                 `json:"severity"` // "warning", "critical"
	Message    string                 `json:"message"`
	Timestamp  time.Time              `json:"timestamp"`
	Resolved   bool                   `json:"resolved"`
	ResolvedAt time.Time              `json:"resolved_at,omitempty"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// HealthTrend represents health status over time.
type HealthTrend struct {
	Component  string      `json:"component"`
	Timeframe  string      `json:"timeframe"` // "1h", "24h", "7d"
	DataPoints []DataPoint `json:"data_points"`
	Trend      string      `json:"trend"` // "improving", "stable", "degrading"
}

// DataPoint represents a single health measurement point.
type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Status    string    `json:"status"`
}

// NewHealthService creates a new health service instance.
func NewHealthService(brokerService *BrokerService, stateService *StateService, identityClient *auth.IdentityClient, cfg *config.Config) *HealthService { //nolint:revive
	logger := log.WithField("service", "health_aggregator")

	return &HealthService{
		brokerService:   brokerService,
		stateService:    stateService,
		identityClient:  identityClient,
		config:          cfg,
		logger:          logger,
		healthHistory:   make([]SystemHealth, 0),
		maxHistorySize:  100, // Keep last 100 health checks
		alertThresholds: getDefaultAlertThresholds(),
	}
}

// GetSystemHealth performs a comprehensive system health check.
func (hs *HealthService) GetSystemHealth(ctx context.Context) (*SystemHealth, error) { //nolint:funlen
	startTime := time.Now()
	hs.logger.Debug("Starting system health check")

	// Use shorter timeout for individual health checks
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// Track components being checked
	components := make(map[string]Component)
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}

	// Check broker service health with error recovery
	wg.Add(1)
	go func() {
		defer func() {
			wg.Done()
			if r := recover(); r != nil {
				hs.logger.WithField("panic", r).Error("Panic in broker health check")
				mu.Lock()
				components["broker"] = Component{
					Status:    StatusUnhealthy,
					Message:   fmt.Sprintf("Health check failed due to panic: %v", r),
					LastCheck: time.Now(),
					Metadata:  make(map[string]string),
				}
				mu.Unlock()
			}
		}()

		// Create individual timeout for this check
		brokerCtx, brokerCancel := context.WithTimeout(checkCtx, 500*time.Millisecond)
		defer brokerCancel()

		component := hs.checkBrokerHealthSafe(brokerCtx)
		mu.Lock()
		components["broker"] = component
		mu.Unlock()
	}()

	// Check state service health with error recovery
	wg.Add(1)
	go func() {
		defer func() {
			wg.Done()
			if r := recover(); r != nil {
				hs.logger.WithField("panic", r).Error("Panic in state service health check")
				mu.Lock()
				components["state"] = Component{
					Status:    StatusUnhealthy,
					Message:   fmt.Sprintf("Health check failed due to panic: %v", r),
					LastCheck: time.Now(),
					Metadata:  make(map[string]string),
				}
				mu.Unlock()
			}
		}()

		// Create individual timeout for this check
		stateCtx, stateCancel := context.WithTimeout(checkCtx, 500*time.Millisecond)
		defer stateCancel()

		component := hs.checkStateServiceHealthSafe(stateCtx)
		mu.Lock()
		components["state"] = component
		mu.Unlock()
	}()

	// Check identity service health with error recovery
	wg.Add(1)
	go func() {
		defer func() {
			wg.Done()
			if r := recover(); r != nil {
				hs.logger.WithField("panic", r).Error("Panic in identity service health check")
				mu.Lock()
				components["identity"] = Component{
					Status:    StatusUnhealthy,
					Message:   fmt.Sprintf("Health check failed due to panic: %v", r),
					LastCheck: time.Now(),
					Metadata:  make(map[string]string),
				}
				mu.Unlock()
			}
		}()

		// Create individual timeout for this check
		identityCtx, identityCancel := context.WithTimeout(checkCtx, 500*time.Millisecond)
		defer identityCancel()

		component := hs.checkIdentityServiceHealthSafe(identityCtx)
		mu.Lock()
		components["identity"] = component
		mu.Unlock()
	}()

	// Check app service health (self-check) with error recovery
	wg.Add(1)
	go func() {
		defer func() {
			wg.Done()
			if r := recover(); r != nil {
				hs.logger.WithField("panic", r).Error("Panic in app service health check")
				mu.Lock()
				components["app"] = Component{
					Status:    StatusHealthy, // Self-check should always be healthy if we can run this
					Message:   "App service is operational (self-check)",
					LastCheck: time.Now(),
					Metadata:  make(map[string]string),
				}
				mu.Unlock()
			}
		}()

		component := hs.checkAppServiceHealth(checkCtx)
		mu.Lock()
		components["app"] = component
		mu.Unlock()
	}()

	// Wait for all health checks to complete or timeout
	done := make(chan bool, 1)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		// All checks completed
	case <-checkCtx.Done():
		// Overall timeout reached
		hs.logger.Warn("Health check timeout reached, some checks may be incomplete")
	}

	// Calculate overall system health
	overall := hs.calculateOverallHealth(components)
	summary := hs.calculateHealthSummary(components)

	systemHealth := &SystemHealth{
		Overall:    overall,
		Components: components,
		Timestamp:  time.Now(),
		Uptime:     time.Since(startTime), // This would be app uptime in real implementation
		Summary:    summary,
	}

	// Add to history
	hs.addToHistory(*systemHealth)
	hs.lastHealthCheck = time.Now()

	hs.logger.WithFields(log.Fields{
		"overall":            overall,
		"total_components":   summary.TotalComponents,
		"healthy_components": summary.HealthyComponents,
		"check_duration":     time.Since(startTime),
	}).Debug("System health check completed")

	return systemHealth, nil
}

// CheckComponentHealth checks the health of a specific component.
func (hs *HealthService) CheckComponentHealth(ctx context.Context, componentName string) (*Component, error) {
	hs.logger.WithField("component", componentName).Debug("Checking component health")

	var component Component

	switch componentName {
	case "broker":
		component = hs.checkBrokerHealthSafe(ctx)
	case "state":
		component = hs.checkStateServiceHealthSafe(ctx)
	case "identity":
		component = hs.checkIdentityServiceHealthSafe(ctx)
	case "app":
		component = hs.checkAppServiceHealth(ctx)
	default:
		return nil, fmt.Errorf("unknown component: %s", componentName)
	}

	return &component, nil
}

// RunHealthCheck executes a comprehensive health check and returns the results.
func (hs *HealthService) RunHealthCheck(ctx context.Context) (*SystemHealth, error) {
	return hs.GetSystemHealth(ctx)
}

// GetHealthHistory returns historical health data.
func (hs *HealthService) GetHealthHistory(timeframe string) ([]SystemHealth, error) { //nolint:revive
	// For now, return all history
	// In a real implementation, this would filter by timeframe
	return hs.healthHistory, nil
}

// GetHealthTrends returns health trends for analysis.
func (hs *HealthService) GetHealthTrends(ctx context.Context, componentName, timeframe string) (*HealthTrend, error) { //nolint:revive
	hs.logger.WithFields(log.Fields{
		"component": componentName,
		"timeframe": timeframe,
	}).Debug("Calculating health trends")

	// Filter history for the component
	var dataPoints []DataPoint
	for _, health := range hs.healthHistory {
		if comp, exists := health.Components[componentName]; exists {
			score := hs.componentStatusToScore(comp.Status)
			dataPoints = append(dataPoints, DataPoint{
				Timestamp: health.Timestamp,
				Value:     score,
				Status:    comp.Status,
			})
		}
	}

	// Calculate trend direction
	trend := StatusStable
	if len(dataPoints) > 1 {
		recent := dataPoints[len(dataPoints)-1].Value
		older := dataPoints[0].Value
		if recent > older {
			trend = "improving"
		} else if recent < older {
			trend = "degrading"
		}
	}

	return &HealthTrend{
		Component:  componentName,
		Timeframe:  timeframe,
		DataPoints: dataPoints,
		Trend:      trend,
	}, nil
}

// checkBrokerHealthSafe performs health check on the broker service with error recovery.
func (hs *HealthService) checkBrokerHealthSafe(ctx context.Context) Component { //nolint:revive
	defer func() {
		if r := recover(); r != nil {
			hs.logger.WithField("panic", r).Error("Panic in broker health check")
		}
	}()

	startTime := time.Now()
	component := Component{
		Status:    StatusDegraded,
		Message:   "Broker health check temporarily disabled (ZeroMQ stability)",
		LastCheck: time.Now(),
		Metadata:  make(map[string]string),
	}

	// Temporarily disable broker connectivity checks to prevent CGO crashes
	// TODO: Re-enable once ZeroMQ stability issues are resolved
	component.Metadata["note"] = "Health check disabled due to ZeroMQ CGO signal stack issues"
	component.Latency = time.Since(startTime)
	component.SuccessRate = 1.0 // Assume healthy for now
	return component
}

// checkStateServiceHealthSafe performs health check on the state service with error recovery.
func (hs *HealthService) checkStateServiceHealthSafe(ctx context.Context) Component { //nolint:revive
	defer func() {
		if r := recover(); r != nil {
			hs.logger.WithField("panic", r).Error("Panic in state service health check")
		}
	}()

	startTime := time.Now()
	component := Component{
		Status:    StatusDegraded,
		Message:   "State service health check temporarily disabled (ZeroMQ stability)",
		LastCheck: time.Now(),
		Metadata:  make(map[string]string),
	}

	// Temporarily disable state service connectivity checks to prevent CGO crashes
	// TODO: Re-enable once ZeroMQ stability issues are resolved
	component.Metadata["note"] = "Health check disabled due to ZeroMQ CGO signal stack issues"
	component.Latency = time.Since(startTime)
	component.SuccessRate = 1.0 // Assume healthy for now
	return component
}

// checkIdentityServiceHealthSafe performs health check on the identity service with error recovery.
func (hs *HealthService) checkIdentityServiceHealthSafe(ctx context.Context) Component { //nolint:revive
	defer func() {
		if r := recover(); r != nil {
			hs.logger.WithField("panic", r).Error("Panic in identity service health check")
		}
	}()

	startTime := time.Now()
	component := Component{
		Status:    StatusHealthy,
		Message:   "Identity service is operational",
		LastCheck: time.Now(),
		Metadata:  make(map[string]string),
	}

	if hs.identityClient == nil {
		component.Status = StatusDegraded
		component.Message = "Identity service client not configured"
		return component
	}

	// Check identity service health with timeout
	err := hs.identityClient.HealthCheck()
	if err != nil {
		component.Status = StatusUnhealthy
		component.Message = fmt.Sprintf("Identity service health check failed: %v", err)
		component.ErrorCount++
	} else {
		component.Metadata["authentication"] = "available"
	}

	component.Latency = time.Since(startTime)
	component.SuccessRate = hs.calculateSuccessRate(component.ErrorCount, 1)
	return component
}

// checkAppServiceHealth performs self health check on the app service.
func (hs *HealthService) checkAppServiceHealth(ctx context.Context) Component { //nolint:revive
	startTime := time.Now()
	component := Component{
		Status:      StatusHealthy,
		Message:     "App service is operational",
		LastCheck:   time.Now(),
		Metadata:    make(map[string]string),
		ErrorCount:  0,
		SuccessRate: 100.0,
	}

	// Basic self-checks
	component.Metadata["uptime"] = time.Since(startTime).String()
	component.Metadata["status"] = "running"

	component.Latency = time.Since(startTime)
	return component
}

// calculateOverallHealth determines the overall system health status.
func (hs *HealthService) calculateOverallHealth(components map[string]Component) string {
	healthyCount := 0
	degradedCount := 0
	unhealthyCount := 0

	for _, component := range components {
		switch component.Status {
		case StatusHealthy:
			healthyCount++
		case StatusDegraded:
			degradedCount++
		case StatusUnhealthy:
			unhealthyCount++
		}
	}

	totalComponents := len(components)
	if totalComponents == 0 {
		return StatusUnknown
	}

	// Calculate overall status
	if unhealthyCount > 0 {
		if unhealthyCount >= totalComponents/2 {
			return StatusUnhealthy
		}
		return StatusDegraded
	}

	if degradedCount > 0 {
		return StatusDegraded
	}

	return StatusHealthy
}

// calculateHealthSummary creates a summary of component health.
func (hs *HealthService) calculateHealthSummary(components map[string]Component) HealthSummary {
	summary := HealthSummary{
		TotalComponents: len(components),
	}

	for _, component := range components {
		switch component.Status {
		case StatusHealthy:
			summary.HealthyComponents++
		case StatusDegraded:
			summary.DegradedComponents++
		case StatusUnhealthy:
			summary.UnhealthyComponents++
		}
	}

	// Calculate overall score (0-100)
	if summary.TotalComponents > 0 {
		healthyWeight := float64(summary.HealthyComponents) * 100
		degradedWeight := float64(summary.DegradedComponents) * 50
		summary.OverallScore = (healthyWeight + degradedWeight) / float64(summary.TotalComponents)
	}

	return summary
}

// addToHistory adds a health check result to history.
func (hs *HealthService) addToHistory(health SystemHealth) {
	hs.healthHistory = append(hs.healthHistory, health)

	// Keep only the last maxHistorySize entries
	if len(hs.healthHistory) > hs.maxHistorySize {
		hs.healthHistory = hs.healthHistory[1:]
	}
}

// componentStatusToScore converts component status to numeric score.
func (hs *HealthService) componentStatusToScore(status string) float64 {
	switch status {
	case StatusHealthy:
		return 100.0
	case StatusDegraded:
		return 50.0
	case StatusUnhealthy:
		return 0.0
	default:
		return 25.0 // unknown
	}
}

// calculateSuccessRate calculates success rate based on error count and total attempts.
func (hs *HealthService) calculateSuccessRate(errorCount, totalAttempts int) float64 {
	if totalAttempts == 0 {
		return 100.0
	}
	successCount := totalAttempts - errorCount
	return (float64(successCount) / float64(totalAttempts)) * 100.0
}

// getDefaultAlertThresholds returns default alert threshold configuration.
func getDefaultAlertThresholds() map[string]interface{} {
	return map[string]interface{}{
		"response_time_warning_ms":  1000,
		"response_time_critical_ms": 5000,
		"error_rate_warning":        5.0,  // 5%
		"error_rate_critical":       15.0, // 15%
		"success_rate_warning":      95.0, // Below 95%
		"success_rate_critical":     85.0, // Below 85%
	}
}

// GenerateHealthReport generates a comprehensive health report.
func (hs *HealthService) GenerateHealthReport(ctx context.Context) (map[string]interface{}, error) {
	hs.logger.Debug("Generating comprehensive health report")

	// Get current system health
	systemHealth, err := hs.GetSystemHealth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get system health: %w", err)
	}

	// Get health trends
	trends := make(map[string]*HealthTrend)
	for componentName := range systemHealth.Components {
		if trend, err := hs.GetHealthTrends(ctx, componentName, "24h"); err == nil {
			trends[componentName] = trend
		}
	}

	report := map[string]interface{}{
		"timestamp":     time.Now(),
		"system_health": systemHealth,
		"trends":        trends,
		"history_size":  len(hs.healthHistory),
		"last_check":    hs.lastHealthCheck,
	}

	hs.logger.WithFields(log.Fields{
		"overall_status": systemHealth.Overall,
		"components":     len(systemHealth.Components),
		"trends":         len(trends),
	}).Debug("Health report generated")

	return report, nil
}
