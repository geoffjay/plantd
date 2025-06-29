// Package services provides business logic for service integrations.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/geoffjay/plantd/app/config"
	"github.com/geoffjay/plantd/core/mdp"

	log "github.com/sirupsen/logrus"
)

// BrokerService handles communication with the plantd broker for service discovery and management.
type BrokerService struct {
	client         *mdp.Client
	config         *config.Config
	logger         *log.Entry
	circuitBreaker *CircuitBreaker
	mutex          sync.RWMutex
	lastError      error
	disabled       bool
}

// CircuitBreaker implements a simple circuit breaker pattern.
type CircuitBreaker struct {
	failureCount    int
	failureLimit    int
	resetTimeout    time.Duration
	lastFailureTime time.Time
	state           CircuitBreakerState
	mutex           sync.RWMutex
}

// CircuitBreakerState represents the state of the circuit breaker.
type CircuitBreakerState int

const (
	// CircuitClosed represents the closed state of the circuit breaker.
	CircuitClosed CircuitBreakerState = iota
	// CircuitOpen represents the open state of the circuit breaker.
	CircuitOpen
	// CircuitHalfOpen represents the half-open state of the circuit breaker.
	CircuitHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(failureLimit int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureLimit: failureLimit,
		resetTimeout: resetTimeout,
		state:        CircuitClosed,
	}
}

// Call executes a function with circuit breaker protection.
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	// Check if circuit should be reset
	if cb.state == CircuitOpen && time.Since(cb.lastFailureTime) > cb.resetTimeout {
		cb.state = CircuitHalfOpen
		cb.failureCount = 0
	}

	// If circuit is open, return error immediately
	if cb.state == CircuitOpen {
		return fmt.Errorf("circuit breaker is open")
	}

	// Execute function
	err := fn()
	if err != nil {
		cb.failureCount++
		cb.lastFailureTime = time.Now()

		// Open circuit if failure limit reached
		if cb.failureCount >= cb.failureLimit {
			cb.state = CircuitOpen
		}
		return err
	}

	// Success - reset circuit
	if cb.state == CircuitHalfOpen {
		cb.state = CircuitClosed
	}
	cb.failureCount = 0

	return nil
}

// ServiceStatus represents the status information for a plantd service.
type ServiceStatus struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"` // "healthy", "unhealthy", "unknown"
	Workers     int       `json:"workers"`
	LastSeen    time.Time `json:"last_seen"`
	Version     string    `json:"version"`
	Endpoint    string    `json:"endpoint"`
	Heartbeat   int       `json:"heartbeat_ms"`
	RequestRate float64   `json:"request_rate"`
	ErrorRate   float64   `json:"error_rate"`
}

// BrokerHealth represents broker-specific health and performance metrics.
type BrokerHealth struct {
	Status          string                   `json:"status"`
	TotalServices   int                      `json:"total_services"`
	TotalWorkers    int                      `json:"total_workers"`
	MessageRate     float64                  `json:"message_rate"`
	Uptime          time.Duration            `json:"uptime"`
	LastHeartbeat   time.Time                `json:"last_heartbeat"`
	ServiceStatuses map[string]ServiceStatus `json:"service_statuses"`
	WorkerDetails   map[string]WorkerDetails `json:"worker_details"`
	BrokerMetrics   BrokerMetrics            `json:"broker_metrics"`
}

// WorkerDetails represents detailed information about a worker.
type WorkerDetails struct {
	ID            string    `json:"id"`
	ServiceName   string    `json:"service_name"`
	LastSeen      time.Time `json:"last_seen"`
	TotalRequests int64     `json:"total_requests"`
	Status        string    `json:"status"`
}

// BrokerMetrics represents broker performance metrics.
type BrokerMetrics struct {
	MessagesProcessed int64         `json:"messages_processed"`
	AvgResponseTime   time.Duration `json:"avg_response_time"`
	ErrorCount        int64         `json:"error_count"`
	ActiveConnections int           `json:"active_connections"`
}

// NewBrokerService creates a new broker service client.
func NewBrokerService(cfg *config.Config) (*BrokerService, error) {
	logger := log.WithField("service", "broker_client")

	// Workaround: If broker endpoint is empty, use default from config defaults
	brokerEndpoint := cfg.Services.BrokerEndpoint
	if brokerEndpoint == "" {
		brokerEndpoint = "tcp://127.0.0.1:9797"
		logger.Warn("Broker endpoint was empty, using default: tcp://127.0.0.1:9797")
	}

	// Debug logging for configuration
	logger.WithFields(log.Fields{
		"broker_endpoint": brokerEndpoint,
		"state_endpoint":  cfg.Services.StateEndpoint,
		"timeout":         cfg.Services.Timeout,
	}).Debug("Broker service configuration loaded")

	// Create circuit breaker to prevent memory corruption issues
	circuitBreaker := NewCircuitBreaker(3, 30*time.Second) // 3 failures, 30s reset

	// Create broker service without client initially
	bs := &BrokerService{
		config:         cfg,
		logger:         logger,
		circuitBreaker: circuitBreaker,
		disabled:       false,
	}

	// Try to create MDP client with circuit breaker protection
	err := circuitBreaker.Call(func() error {
		client, err := mdp.NewClient(brokerEndpoint)
		if err != nil {
			return fmt.Errorf("failed to create MDP client: %w", err)
		}

		// Parse timeout
		timeout, err := time.ParseDuration(cfg.Services.Timeout)
		if err != nil {
			timeout = 30 * time.Second
			logger.WithError(err).Warn("Failed to parse services timeout, using default 30s")
		}
		client.SetTimeout(timeout)

		bs.mutex.Lock()
		bs.client = client
		bs.mutex.Unlock()

		return nil
	})

	if err != nil {
		logger.WithError(err).Warn("Failed to initialize broker client, broker functionality will be disabled")
		bs.disabled = true
		bs.lastError = err
		// Don't return error - allow service to start without broker
	} else {
		logger.WithFields(log.Fields{
			"broker_endpoint": cfg.Services.BrokerEndpoint,
			"timeout":         cfg.Services.Timeout,
		}).Info("Broker service client initialized")
	}

	return bs, nil
}

// Close closes the broker service client connection.
func (bs *BrokerService) Close() error {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()

	if bs.client != nil {
		err := bs.client.Close()
		bs.client = nil
		return err
	}
	return nil
}

// GetServiceStatuses retrieves status information for all registered services.
func (bs *BrokerService) GetServiceStatuses(ctx context.Context) ([]ServiceStatus, error) {
	if bs.logger != nil {
		bs.logger.Debug("Requesting service statuses from broker")
	}

	// Check if broker service is disabled
	bs.mutex.RLock()
	disabled := bs.disabled
	lastError := bs.lastError
	bs.mutex.RUnlock()

	if disabled {
		if bs.logger != nil {
			bs.logger.WithError(lastError).Warn("Broker service is disabled, returning empty service list")
		}
		return []ServiceStatus{}, nil
	}

	// Query broker for service list using MMI (Management Interface)
	response, err := bs.queryMMI(ctx, "mmi.services")
	if err != nil {
		if bs.logger != nil {
			bs.logger.WithError(err).Warn("Failed to query services from broker")
		}
		return []ServiceStatus{}, nil // Return empty list instead of error to prevent app crash
	}

	if len(response) == 0 {
		if bs.logger != nil {
			bs.logger.Warn("No services found in broker")
		}
		return []ServiceStatus{}, nil
	}

	// Parse service names from response
	serviceNames := response

	// Get detailed status for each service
	var statuses []ServiceStatus
	for _, serviceName := range serviceNames {
		status, err := bs.getServiceDetails(ctx, serviceName)
		if err != nil {
			if bs.logger != nil {
				bs.logger.WithError(err).WithField("service", serviceName).
					Warn("Failed to get service details, using basic status")
			}

			// Create basic status if detailed query fails
			status = &ServiceStatus{
				Name:      serviceName,
				Status:    "unknown",
				Workers:   0,
				LastSeen:  time.Time{},
				Version:   "",
				Endpoint:  "",
				Heartbeat: 0,
			}
		}
		statuses = append(statuses, *status)
	}

	if bs.logger != nil {
		bs.logger.WithField("service_count", len(statuses)).Debug("Retrieved service statuses")
	}
	return statuses, nil
}

// GetServiceDetails retrieves detailed information for a specific service.
func (bs *BrokerService) GetServiceDetails(ctx context.Context, serviceName string) (*ServiceStatus, error) {
	return bs.getServiceDetails(ctx, serviceName)
}

// getServiceDetails is the internal implementation for getting service details.
func (bs *BrokerService) getServiceDetails(ctx context.Context, serviceName string) (*ServiceStatus, error) {
	bs.logger.WithField("service", serviceName).Debug("Getting service details")

	// Query service workers
	workersResponse, err := bs.queryMMI(ctx, "mmi.service."+serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to query service workers: %w", err)
	}

	// Parse worker count
	workerCount := len(workersResponse)

	// Determine service status based on worker availability
	var status string
	if workerCount > 0 {
		status = StatusHealthy
	} else {
		status = StatusUnhealthy
	}

	// For now, create basic service status
	// In a full implementation, this would query additional metrics
	serviceStatus := &ServiceStatus{
		Name:        serviceName,
		Status:      status,
		Workers:     workerCount,
		LastSeen:    time.Now(), // This would come from broker metrics
		Version:     "",         // This would require service-specific query
		Endpoint:    "",         // This would come from service registration
		Heartbeat:   2500,       // Default MDP heartbeat interval
		RequestRate: 0.0,        // This would come from broker metrics
		ErrorRate:   0.0,        // This would come from broker metrics
	}

	return serviceStatus, nil
}

// GetBrokerHealth retrieves comprehensive broker health and performance metrics.
func (bs *BrokerService) GetBrokerHealth(ctx context.Context) (*BrokerHealth, error) {
	bs.logger.Debug("Getting broker health status")

	// Query broker services
	services, err := bs.GetServiceStatuses(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get service statuses: %w", err)
	}

	// Build service status map
	serviceStatuses := make(map[string]ServiceStatus)
	totalWorkers := 0
	for _, service := range services {
		serviceStatuses[service.Name] = service
		totalWorkers += service.Workers
	}

	// Determine overall broker status
	brokerStatus := StatusHealthy
	if len(services) == 0 {
		brokerStatus = StatusUnhealthy
	}

	// Create broker health response
	health := &BrokerHealth{
		Status:          brokerStatus,
		TotalServices:   len(services),
		TotalWorkers:    totalWorkers,
		MessageRate:     0.0, // This would come from broker metrics
		Uptime:          0,   // This would come from broker
		LastHeartbeat:   time.Now(),
		ServiceStatuses: serviceStatuses,
		WorkerDetails:   make(map[string]WorkerDetails),
		BrokerMetrics: BrokerMetrics{
			MessagesProcessed: 0, // These would come from broker metrics
			AvgResponseTime:   0,
			ErrorCount:        0,
			ActiveConnections: totalWorkers,
		},
	}

	bs.logger.WithFields(log.Fields{
		"status":         health.Status,
		"total_services": health.TotalServices,
		"total_workers":  health.TotalWorkers,
	}).Debug("Retrieved broker health")

	return health, nil
}

// CheckConnectivity verifies connectivity to the broker.
func (bs *BrokerService) CheckConnectivity(ctx context.Context) error {
	bs.mutex.RLock()
	disabled := bs.disabled
	lastError := bs.lastError
	bs.mutex.RUnlock()

	if disabled {
		return fmt.Errorf("broker service is disabled due to previous errors: %w", lastError)
	}

	if bs.circuitBreaker != nil && bs.circuitBreaker.state == CircuitOpen {
		return fmt.Errorf("circuit breaker is open")
	}

	if bs.logger != nil {
		bs.logger.Debug("Checking broker connectivity")
	}

	// Create timeout context for this specific check
	timeoutCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	// Try to query broker status with timeout
	response, err := bs.queryMMIWithTimeout(timeoutCtx, "mmi.status")
	if err != nil {
		return fmt.Errorf("broker connectivity check failed: %w", err)
	}

	if len(response) == 0 {
		return fmt.Errorf("broker returned empty status response")
	}

	if bs.logger != nil {
		bs.logger.Debug("Broker connectivity check successful")
	}
	return nil
}

// RestartService attempts to restart a specific service (if supported by the service).
func (bs *BrokerService) RestartService(ctx context.Context, serviceName string) error {
	bs.logger.WithField("service", serviceName).Info("Attempting to restart service")

	// Create timeout context for this specific operation
	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// This would typically send a management command to the service
	// For now, this is a placeholder implementation
	response, err := bs.queryServiceWithTimeout(timeoutCtx, serviceName, "restart")
	if err != nil {
		return fmt.Errorf("failed to restart service %s: %w", serviceName, err)
	}

	bs.logger.WithField("service", serviceName).WithField("response", response).
		Info("Service restart command sent")
	return nil
}

// queryMMIWithTimeout queries the broker's Management Interface (MMI) with timeout protection.
func (bs *BrokerService) queryMMIWithTimeout(ctx context.Context, query string) ([]string, error) {
	bs.mutex.RLock()
	disabled := bs.disabled
	lastError := bs.lastError
	client := bs.client
	bs.mutex.RUnlock()

	if disabled {
		return nil, fmt.Errorf("broker service is disabled due to previous errors: %w", lastError)
	}

	if client == nil {
		return nil, fmt.Errorf("broker client is not initialized")
	}

	bs.logger.WithField("query", query).Trace("Sending MMI query with timeout")

	var response []string
	err := bs.circuitBreaker.Call(func() error {
		// Create a result channel for safe goroutine communication
		type mdpResult struct {
			response []string
			err      error
		}

		resultChan := make(chan mdpResult, 1)

		// Run MDP operation in isolated goroutine with panic recovery
		go func() {
			defer func() {
				if r := recover(); r != nil {
					bs.logger.WithField("panic", r).Error("Panic in MMI query")
					resultChan <- mdpResult{
						response: nil,
						err:      fmt.Errorf("MMI query panicked: %v", r),
					}
				}
			}()

			// Send MMI query to broker
			if err := client.Send(query); err != nil {
				resultChan <- mdpResult{
					response: nil,
					err:      fmt.Errorf("failed to send MMI query: %w", err),
				}
				return
			}

			// Receive response
			resp, err := client.Recv()
			if err != nil {
				resultChan <- mdpResult{
					response: nil,
					err:      fmt.Errorf("failed to receive MMI response: %w", err),
				}
				return
			}

			resultChan <- mdpResult{
				response: resp,
				err:      nil,
			}
		}()

		// Wait for either completion or timeout
		select {
		case result := <-resultChan:
			if result.err != nil {
				return result.err
			}
			response = result.response
			return nil
		case <-ctx.Done():
			return fmt.Errorf("MMI query timed out: %w", ctx.Err())
		}
	})

	if err != nil {
		bs.logger.WithError(err).WithField("query", query).Error("MMI query failed")

		// If circuit breaker is now open, disable the service
		if bs.circuitBreaker.state == CircuitOpen {
			bs.mutex.Lock()
			bs.disabled = true
			bs.lastError = err
			bs.mutex.Unlock()
			bs.logger.Warn("Broker service disabled due to circuit breaker opening")
		}

		return nil, err
	}

	bs.logger.WithField("query", query).WithField("response", response).
		Trace("Received MMI response")

	return response, nil
}

// queryServiceWithTimeout sends a query to a specific service with timeout protection.
func (bs *BrokerService) queryServiceWithTimeout(ctx context.Context, serviceName string, command string, args ...string) ([]string, error) {
	bs.logger.WithFields(log.Fields{
		"service": serviceName,
		"command": command,
		"args":    args,
	}).Trace("Sending service query with timeout")

	// Prepare message
	message := append([]string{command}, args...)

	// Create a result channel for safe goroutine communication
	type mdpResult struct {
		response []string
		err      error
	}

	resultChan := make(chan mdpResult, 1)

	// Run MDP operation in isolated goroutine with panic recovery
	go func() {
		defer func() {
			if r := recover(); r != nil {
				bs.logger.WithField("panic", r).Error("Panic in service query")
				resultChan <- mdpResult{
					response: nil,
					err:      fmt.Errorf("service query panicked: %v", r),
				}
			}
		}()

		// Send service query
		if err := bs.client.Send(serviceName, message...); err != nil {
			resultChan <- mdpResult{
				response: nil,
				err:      fmt.Errorf("failed to send service query: %w", err),
			}
			return
		}

		// Receive response
		resp, err := bs.client.Recv()
		if err != nil {
			resultChan <- mdpResult{
				response: nil,
				err:      fmt.Errorf("failed to receive service response: %w", err),
			}
			return
		}

		resultChan <- mdpResult{
			response: resp,
			err:      nil,
		}
	}()

	// Wait for either completion or timeout
	select {
	case result := <-resultChan:
		if result.err != nil {
			return nil, result.err
		}
		bs.logger.WithFields(log.Fields{
			"service":  serviceName,
			"command":  command,
			"response": result.response,
		}).Trace("Received service response")
		return result.response, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("service query timed out: %w", ctx.Err())
	}
}

// Legacy methods for compatibility
func (bs *BrokerService) queryMMI(ctx context.Context, query string) ([]string, error) {
	return bs.queryMMIWithTimeout(ctx, query)
}

func (bs *BrokerService) queryService(ctx context.Context, serviceName string, command string, args ...string) ([]string, error) { //nolint:unused
	return bs.queryServiceWithTimeout(ctx, serviceName, command, args...)
}

// GetMetrics retrieves real-time performance metrics from the broker.
func (bs *BrokerService) GetMetrics(ctx context.Context) (*BrokerMetrics, error) {
	bs.logger.Debug("Getting broker metrics")

	// Query broker for metrics
	response, err := bs.queryMMI(ctx, "mmi.metrics")
	if err != nil {
		return nil, fmt.Errorf("failed to query broker metrics: %w", err)
	}

	// Parse metrics response
	metrics := &BrokerMetrics{
		MessagesProcessed: 0,
		AvgResponseTime:   0,
		ErrorCount:        0,
		ActiveConnections: 0,
	}

	// If we get a JSON response, try to parse it
	if len(response) > 0 && response[0] != "" {
		if err := json.Unmarshal([]byte(response[0]), metrics); err != nil {
			// If JSON parsing fails, try to parse as simple key-value pairs
			bs.logger.WithError(err).Debug("Failed to parse metrics as JSON, using fallback")
			metrics = bs.parseSimpleMetrics(response)
		}
	}

	bs.logger.WithField("metrics", metrics).Debug("Retrieved broker metrics")
	return metrics, nil
}

// parseSimpleMetrics parses metrics from simple key-value format.
func (bs *BrokerService) parseSimpleMetrics(response []string) *BrokerMetrics {
	metrics := &BrokerMetrics{}

	for _, item := range response {
		if len(item) == 0 {
			continue
		}

		// Parse key:value format
		parts := strings.SplitN(item, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "messages_processed":
			if val, err := strconv.ParseInt(value, 10, 64); err == nil {
				metrics.MessagesProcessed = val
			}
		case "avg_response_time":
			if duration, err := time.ParseDuration(value); err == nil {
				metrics.AvgResponseTime = duration
			}
		case "error_count":
			if val, err := strconv.ParseInt(value, 10, 64); err == nil {
				metrics.ErrorCount = val
			}
		case "active_connections":
			if val, err := strconv.Atoi(value); err == nil {
				metrics.ActiveConnections = val
			}
		}
	}

	return metrics
}

// HealthCheck performs a comprehensive health check of the broker service.
func (bs *BrokerService) HealthCheck(ctx context.Context) (map[string]interface{}, error) {
	healthStatus := map[string]interface{}{
		"status":    StatusHealthy,
		"timestamp": time.Now(),
		"checks":    map[string]interface{}{},
	}

	checks := healthStatus["checks"].(map[string]interface{})

	// Check connectivity
	if err := bs.CheckConnectivity(ctx); err != nil {
		checks["connectivity"] = map[string]interface{}{
			"status": "failed",
			"error":  err.Error(),
		}
		healthStatus["status"] = StatusUnhealthy
	} else {
		checks["connectivity"] = map[string]interface{}{
			"status": "passed",
		}
	}

	// Check services
	services, err := bs.GetServiceStatuses(ctx)
	if err != nil {
		checks["services"] = map[string]interface{}{
			"status": "failed",
			"error":  err.Error(),
		}
		healthStatus["status"] = StatusDegraded
	} else {
		healthyServices := 0
		for _, service := range services {
			if service.Status == StatusHealthy {
				healthyServices++
			}
		}

		checks["services"] = map[string]interface{}{
			"status":           "passed",
			"total_services":   len(services),
			"healthy_services": healthyServices,
		}

		if healthyServices == 0 && len(services) > 0 {
			healthStatus["status"] = StatusDegraded
		}
	}

	return healthStatus, nil
}

// IsAvailable returns whether the broker service is available for use.
func (bs *BrokerService) IsAvailable() bool {
	bs.mutex.RLock()
	defer bs.mutex.RUnlock()
	return !bs.disabled && bs.client != nil
}

// GetStatus returns the current status of the broker service.
func (bs *BrokerService) GetStatus() map[string]interface{} {
	bs.mutex.RLock()
	defer bs.mutex.RUnlock()

	status := map[string]interface{}{
		"available": !bs.disabled && bs.client != nil,
		"disabled":  bs.disabled,
	}

	// Add status and endpoint fields for compatibility
	if bs.disabled {
		status["status"] = "disabled"
	} else {
		status["status"] = StatusHealthy
	}

	if bs.config != nil {
		status["endpoint"] = bs.config.Services.BrokerEndpoint
	}

	if bs.lastError != nil {
		status["last_error"] = bs.lastError.Error()
	} else {
		status["last_error"] = nil
	}

	if bs.circuitBreaker != nil {
		bs.circuitBreaker.mutex.RLock()
		status["circuit_breaker"] = map[string]interface{}{
			"state":         bs.circuitBreaker.state,
			"failure_count": bs.circuitBreaker.failureCount,
		}
		bs.circuitBreaker.mutex.RUnlock()
	}

	return status
}
