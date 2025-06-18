// Package services provides business logic for service integrations.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/geoffjay/plantd/app/config"

	log "github.com/sirupsen/logrus"
)

// MetricsService handles collection and analysis of system performance metrics.
type MetricsService struct {
	brokerService *BrokerService
	stateService  *StateService
	config        *config.Config
	logger        *log.Entry

	// Metrics storage
	systemMetrics    []SystemMetrics
	serviceMetrics   map[string][]ServiceMetrics
	alertThresholds  map[string]float64
	maxHistorySize   int
	metricsInterval  time.Duration
	collectionActive bool
	mu               sync.RWMutex
}

// SystemMetrics represents comprehensive system performance metrics.
type SystemMetrics struct {
	Timestamp   time.Time                 `json:"timestamp"`
	Performance PerformanceMetrics        `json:"performance"`
	Services    map[string]ServiceMetrics `json:"services"`
	System      SystemStats               `json:"system"`
	AppMetrics  AppSpecificMetrics        `json:"app_metrics"`
}

// PerformanceMetrics represents overall system performance indicators.
type PerformanceMetrics struct {
	RequestRate     float64       `json:"request_rate"`      // Requests per second
	ResponseTime    time.Duration `json:"response_time"`     // Average response time
	ErrorRate       float64       `json:"error_rate"`        // Error percentage
	Throughput      float64       `json:"throughput"`        // Messages per second
	ActiveSessions  int           `json:"active_sessions"`   // Active user sessions
	P95ResponseTime time.Duration `json:"p95_response_time"` // 95th percentile response time
	P99ResponseTime time.Duration `json:"p99_response_time"` // 99th percentile response time
}

// ServiceMetrics represents performance metrics for individual services.
type ServiceMetrics struct {
	ServiceName     string        `json:"service_name"`
	Workers         int           `json:"workers"`
	RequestRate     float64       `json:"request_rate"`
	ErrorRate       float64       `json:"error_rate"`
	Latency         time.Duration `json:"latency"`
	Memory          uint64        `json:"memory"` // Memory usage in bytes
	CPU             float64       `json:"cpu"`    // CPU usage percentage
	TotalRequests   int64         `json:"total_requests"`
	FailedRequests  int64         `json:"failed_requests"`
	AvgResponseTime time.Duration `json:"avg_response_time"`
	Uptime          time.Duration `json:"uptime"`
}

// SystemStats represents system-level resource usage.
type SystemStats struct {
	CPUUsage      float64       `json:"cpu_usage"`      // Overall CPU usage %
	MemoryUsage   uint64        `json:"memory_usage"`   // Memory usage in bytes
	MemoryTotal   uint64        `json:"memory_total"`   // Total memory in bytes
	MemoryPercent float64       `json:"memory_percent"` // Memory usage %
	DiskUsage     uint64        `json:"disk_usage"`     // Disk usage in bytes
	DiskTotal     uint64        `json:"disk_total"`     // Total disk in bytes
	DiskPercent   float64       `json:"disk_percent"`   // Disk usage %
	NetworkRx     uint64        `json:"network_rx"`     // Network bytes received
	NetworkTx     uint64        `json:"network_tx"`     // Network bytes transmitted
	Goroutines    int           `json:"goroutines"`     // Number of goroutines
	OpenFiles     int           `json:"open_files"`     // Open file descriptors
	LoadAverage   []float64     `json:"load_average"`   // System load average
	Uptime        time.Duration `json:"uptime"`         // System uptime
}

// AppSpecificMetrics represents metrics specific to the app service.
type AppSpecificMetrics struct {
	HTTPRequests            int64 `json:"http_requests"`
	HTTPErrors              int64 `json:"http_errors"`
	AuthenticationAttempts  int64 `json:"auth_attempts"`
	AuthenticationSuccesses int64 `json:"auth_successes"`
	SessionsCreated         int64 `json:"sessions_created"`
	SessionsDestroyed       int64 `json:"sessions_destroyed"`
	TemplateRenders         int64 `json:"template_renders"`
	DatabaseQueries         int64 `json:"database_queries"`
	CacheHits               int64 `json:"cache_hits"`
	CacheMisses             int64 `json:"cache_misses"`
}

// MetricsAlert represents a performance alert.
type MetricsAlert struct {
	ID           string                 `json:"id"`
	MetricName   string                 `json:"metric_name"`
	Service      string                 `json:"service"`
	Threshold    float64                `json:"threshold"`
	CurrentValue float64                `json:"current_value"`
	Severity     string                 `json:"severity"` // "warning", "critical"
	Message      string                 `json:"message"`
	Timestamp    time.Time              `json:"timestamp"`
	Resolved     bool                   `json:"resolved"`
	ResolvedAt   time.Time              `json:"resolved_at,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// MetricsTrend represents performance trends over time.
type MetricsTrend struct {
	MetricName string      `json:"metric_name"`
	Service    string      `json:"service"`
	Timeframe  string      `json:"timeframe"`
	DataPoints []DataPoint `json:"data_points"`
	Trend      string      `json:"trend"` // "improving", "stable", "degrading"
	AvgValue   float64     `json:"avg_value"`
	MinValue   float64     `json:"min_value"`
	MaxValue   float64     `json:"max_value"`
}

// NewMetricsService creates a new metrics service instance.
func NewMetricsService(brokerService *BrokerService, stateService *StateService, cfg *config.Config) *MetricsService {
	logger := log.WithField("service", "metrics_collector")

	return &MetricsService{
		brokerService:    brokerService,
		stateService:     stateService,
		config:           cfg,
		logger:           logger,
		systemMetrics:    make([]SystemMetrics, 0),
		serviceMetrics:   make(map[string][]ServiceMetrics),
		alertThresholds:  getDefaultMetricsThresholds(),
		maxHistorySize:   200, // Keep last 200 metric collections
		metricsInterval:  30 * time.Second,
		collectionActive: false,
	}
}

// StartCollection starts automatic metrics collection.
func (ms *MetricsService) StartCollection(ctx context.Context) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.collectionActive {
		ms.logger.Warn("Metrics collection is already active")
		return
	}

	ms.collectionActive = true
	ms.logger.WithField("interval", ms.metricsInterval).Info("Starting metrics collection")

	go ms.collectionLoop(ctx)
}

// StopCollection stops automatic metrics collection.
func (ms *MetricsService) StopCollection() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.collectionActive = false
	ms.logger.Info("Stopping metrics collection")
}

// GetSystemMetrics retrieves current comprehensive system metrics.
func (ms *MetricsService) GetSystemMetrics(ctx context.Context) (*SystemMetrics, error) {
	ms.logger.Debug("Collecting current system metrics")

	startTime := time.Now()

	// Collect performance metrics
	performance, err := ms.collectPerformanceMetrics(ctx)
	if err != nil {
		ms.logger.WithError(err).Warn("Failed to collect performance metrics")
		performance = &PerformanceMetrics{} // Use empty metrics
	}

	// Collect service metrics
	serviceMetrics, err := ms.collectAllServiceMetrics(ctx)
	if err != nil {
		ms.logger.WithError(err).Warn("Failed to collect service metrics")
		serviceMetrics = make(map[string]ServiceMetrics)
	}

	// Collect system stats
	systemStats := ms.collectSystemStats()

	// Collect app-specific metrics
	appMetrics := ms.collectAppMetrics()

	metrics := &SystemMetrics{
		Timestamp:   time.Now(),
		Performance: *performance,
		Services:    serviceMetrics,
		System:      systemStats,
		AppMetrics:  appMetrics,
	}

	// Add to history
	ms.addToHistory(*metrics)

	ms.logger.WithField("collection_time", time.Since(startTime)).Debug("System metrics collected")
	return metrics, nil
}

// GetServiceMetrics retrieves metrics for a specific service.
func (ms *MetricsService) GetServiceMetrics(ctx context.Context, serviceName string) (*ServiceMetrics, error) {
	ms.logger.WithField("service", serviceName).Debug("Getting service metrics")

	// Try to get from broker first
	if ms.brokerService != nil {
		if serviceStatus, err := ms.brokerService.GetServiceDetails(ctx, serviceName); err == nil {
			metrics := &ServiceMetrics{
				ServiceName:   serviceName,
				Workers:       serviceStatus.Workers,
				RequestRate:   serviceStatus.RequestRate,
				ErrorRate:     serviceStatus.ErrorRate,
				TotalRequests: 0, // This would come from detailed service metrics
				Uptime:        time.Since(serviceStatus.LastSeen),
			}
			return metrics, nil
		}
	}

	// Fallback to cached metrics
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if serviceMetrics, exists := ms.serviceMetrics[serviceName]; exists && len(serviceMetrics) > 0 {
		latest := serviceMetrics[len(serviceMetrics)-1]
		return &latest, nil
	}

	return nil, fmt.Errorf("no metrics available for service: %s", serviceName)
}

// GetHistoricalMetrics retrieves metrics over a time period.
func (ms *MetricsService) GetHistoricalMetrics(timeRange string) ([]*SystemMetrics, error) {
	ms.logger.WithField("time_range", timeRange).Debug("Getting historical metrics")

	ms.mu.RLock()
	defer ms.mu.RUnlock()

	// For now, return all historical data
	// In a full implementation, this would filter by time range
	var result []*SystemMetrics
	for i := range ms.systemMetrics {
		result = append(result, &ms.systemMetrics[i])
	}

	return result, nil
}

// GetPerformanceAlerts retrieves current performance alerts.
func (ms *MetricsService) GetPerformanceAlerts(ctx context.Context) ([]MetricsAlert, error) {
	ms.logger.Debug("Checking for performance alerts")

	var alerts []MetricsAlert

	// Get current metrics
	currentMetrics, err := ms.GetSystemMetrics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current metrics: %w", err)
	}

	// Check performance thresholds
	alerts = append(alerts, ms.checkPerformanceThresholds(currentMetrics)...)

	// Check service thresholds
	for serviceName, serviceMetrics := range currentMetrics.Services {
		alerts = append(alerts, ms.checkServiceThresholds(serviceName, serviceMetrics)...)
	}

	ms.logger.WithField("alert_count", len(alerts)).Debug("Performance alerts checked")
	return alerts, nil
}

// GetMetricsTrends calculates performance trends over time.
func (ms *MetricsService) GetMetricsTrends(metricName, serviceName, timeframe string) (*MetricsTrend, error) {
	ms.logger.WithFields(log.Fields{
		"metric":    metricName,
		"service":   serviceName,
		"timeframe": timeframe,
	}).Debug("Calculating metrics trends")

	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var dataPoints []DataPoint

	// Extract data points for the specific metric
	for _, metrics := range ms.systemMetrics {
		value := ms.extractMetricValue(metrics, metricName, serviceName)
		if value >= 0 { // -1 indicates metric not found
			dataPoints = append(dataPoints, DataPoint{
				Timestamp: metrics.Timestamp,
				Value:     value,
			})
		}
	}

	if len(dataPoints) == 0 {
		return nil, fmt.Errorf("no data points found for metric: %s", metricName)
	}

	// Calculate trend statistics
	trend := ms.calculateTrend(dataPoints)
	avgValue := ms.calculateAverage(dataPoints)
	minValue := ms.calculateMinimum(dataPoints)
	maxValue := ms.calculateMaximum(dataPoints)

	return &MetricsTrend{
		MetricName: metricName,
		Service:    serviceName,
		Timeframe:  timeframe,
		DataPoints: dataPoints,
		Trend:      trend,
		AvgValue:   avgValue,
		MinValue:   minValue,
		MaxValue:   maxValue,
	}, nil
}

// ExportMetrics exports metrics data in JSON format.
func (ms *MetricsService) ExportMetrics(format string, timeRange string) ([]byte, error) {
	ms.logger.WithFields(log.Fields{
		"format":     format,
		"time_range": timeRange,
	}).Debug("Exporting metrics")

	metrics, err := ms.GetHistoricalMetrics(timeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical metrics: %w", err)
	}

	switch format {
	case "json":
		return json.MarshalIndent(metrics, "", "  ")
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// collectionLoop runs the metrics collection loop.
func (ms *MetricsService) collectionLoop(ctx context.Context) {
	ticker := time.NewTicker(ms.metricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			ms.logger.Debug("Metrics collection loop stopped due to context cancellation")
			return
		case <-ticker.C:
			if !ms.collectionActive {
				return
			}

			if _, err := ms.GetSystemMetrics(ctx); err != nil {
				ms.logger.WithError(err).Error("Failed to collect metrics in background")
			}
		}
	}
}

// collectPerformanceMetrics gathers overall performance metrics.
func (ms *MetricsService) collectPerformanceMetrics(ctx context.Context) (*PerformanceMetrics, error) {
	performance := &PerformanceMetrics{
		RequestRate:     0,
		ResponseTime:    0,
		ErrorRate:       0,
		Throughput:      0,
		ActiveSessions:  0,
		P95ResponseTime: 0,
		P99ResponseTime: 0,
	}

	// Get broker metrics if available
	if ms.brokerService != nil {
		if brokerMetrics, err := ms.brokerService.GetMetrics(ctx); err == nil {
			performance.RequestRate = float64(brokerMetrics.MessagesProcessed) / 60.0 // Messages per minute to per second
			performance.ResponseTime = brokerMetrics.AvgResponseTime
			performance.Throughput = performance.RequestRate
		}
	}

	return performance, nil
}

// collectAllServiceMetrics gathers metrics for all services.
func (ms *MetricsService) collectAllServiceMetrics(ctx context.Context) (map[string]ServiceMetrics, error) {
	serviceMetrics := make(map[string]ServiceMetrics)

	if ms.brokerService != nil {
		services, err := ms.brokerService.GetServiceStatuses(ctx)
		if err != nil {
			return serviceMetrics, err
		}

		for _, service := range services {
			metrics := ServiceMetrics{
				ServiceName:   service.Name,
				Workers:       service.Workers,
				RequestRate:   service.RequestRate,
				ErrorRate:     service.ErrorRate,
				TotalRequests: 0, // This would come from detailed metrics
				Uptime:        time.Since(service.LastSeen),
			}
			serviceMetrics[service.Name] = metrics
		}
	}

	return serviceMetrics, nil
}

// collectSystemStats gathers system-level statistics.
func (ms *MetricsService) collectSystemStats() SystemStats {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return SystemStats{
		CPUUsage:      0, // Would use system CPU monitoring
		MemoryUsage:   memStats.Alloc,
		MemoryTotal:   memStats.Sys,
		MemoryPercent: float64(memStats.Alloc) / float64(memStats.Sys) * 100,
		Goroutines:    runtime.NumGoroutine(),
		Uptime:        time.Since(time.Now()), // Would track actual uptime
	}
}

// collectAppMetrics gathers app-specific metrics.
func (ms *MetricsService) collectAppMetrics() AppSpecificMetrics {
	// These would be collected from actual app counters
	return AppSpecificMetrics{
		HTTPRequests:            0,
		HTTPErrors:              0,
		AuthenticationAttempts:  0,
		AuthenticationSuccesses: 0,
		SessionsCreated:         0,
		SessionsDestroyed:       0,
		TemplateRenders:         0,
		DatabaseQueries:         0,
		CacheHits:               0,
		CacheMisses:             0,
	}
}

// checkPerformanceThresholds checks if performance metrics exceed thresholds.
func (ms *MetricsService) checkPerformanceThresholds(metrics *SystemMetrics) []MetricsAlert {
	var alerts []MetricsAlert

	// Check response time
	if threshold, exists := ms.alertThresholds["response_time_critical_ms"]; exists {
		if float64(metrics.Performance.ResponseTime.Milliseconds()) > threshold {
			alerts = append(alerts, MetricsAlert{
				ID:           fmt.Sprintf("response_time_%d", time.Now().Unix()),
				MetricName:   "response_time",
				Service:      "system",
				Threshold:    threshold,
				CurrentValue: float64(metrics.Performance.ResponseTime.Milliseconds()),
				Severity:     "critical",
				Message:      "Response time exceeds critical threshold",
				Timestamp:    time.Now(),
			})
		}
	}

	// Check error rate
	if threshold, exists := ms.alertThresholds["error_rate_critical"]; exists {
		if metrics.Performance.ErrorRate > threshold {
			alerts = append(alerts, MetricsAlert{
				ID:           fmt.Sprintf("error_rate_%d", time.Now().Unix()),
				MetricName:   "error_rate",
				Service:      "system",
				Threshold:    threshold,
				CurrentValue: metrics.Performance.ErrorRate,
				Severity:     "critical",
				Message:      "Error rate exceeds critical threshold",
				Timestamp:    time.Now(),
			})
		}
	}

	return alerts
}

// checkServiceThresholds checks if service metrics exceed thresholds.
func (ms *MetricsService) checkServiceThresholds(serviceName string, metrics ServiceMetrics) []MetricsAlert {
	var alerts []MetricsAlert

	// Check service error rate
	if threshold, exists := ms.alertThresholds["service_error_rate_critical"]; exists {
		if metrics.ErrorRate > threshold {
			alerts = append(alerts, MetricsAlert{
				ID:           fmt.Sprintf("service_error_%s_%d", serviceName, time.Now().Unix()),
				MetricName:   "error_rate",
				Service:      serviceName,
				Threshold:    threshold,
				CurrentValue: metrics.ErrorRate,
				Severity:     "critical",
				Message:      fmt.Sprintf("Service %s error rate exceeds critical threshold", serviceName),
				Timestamp:    time.Now(),
			})
		}
	}

	return alerts
}

// addToHistory adds metrics to the historical data.
func (ms *MetricsService) addToHistory(metrics SystemMetrics) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.systemMetrics = append(ms.systemMetrics, metrics)

	// Keep only the last maxHistorySize entries
	if len(ms.systemMetrics) > ms.maxHistorySize {
		ms.systemMetrics = ms.systemMetrics[1:]
	}

	// Also add to service-specific history
	for serviceName, serviceMetrics := range metrics.Services {
		if _, exists := ms.serviceMetrics[serviceName]; !exists {
			ms.serviceMetrics[serviceName] = make([]ServiceMetrics, 0)
		}

		ms.serviceMetrics[serviceName] = append(ms.serviceMetrics[serviceName], serviceMetrics)

		// Keep service history limited too
		if len(ms.serviceMetrics[serviceName]) > ms.maxHistorySize {
			ms.serviceMetrics[serviceName] = ms.serviceMetrics[serviceName][1:]
		}
	}
}

// extractMetricValue extracts a specific metric value from system metrics.
func (ms *MetricsService) extractMetricValue(metrics SystemMetrics, metricName, serviceName string) float64 {
	switch metricName {
	case "request_rate":
		if serviceName == "system" {
			return metrics.Performance.RequestRate
		}
		if service, exists := metrics.Services[serviceName]; exists {
			return service.RequestRate
		}
	case "error_rate":
		if serviceName == "system" {
			return metrics.Performance.ErrorRate
		}
		if service, exists := metrics.Services[serviceName]; exists {
			return service.ErrorRate
		}
	case "cpu_usage":
		return metrics.System.CPUUsage
	case "memory_percent":
		return metrics.System.MemoryPercent
	}
	return -1 // Metric not found
}

// calculateTrend determines if metrics are improving, stable, or degrading.
func (ms *MetricsService) calculateTrend(dataPoints []DataPoint) string {
	if len(dataPoints) < 2 {
		return "stable"
	}

	recent := dataPoints[len(dataPoints)-1].Value
	older := dataPoints[0].Value

	if recent > older*1.1 { // 10% increase
		return "degrading" // Higher values are usually worse for performance metrics
	} else if recent < older*0.9 { // 10% decrease
		return "improving"
	}

	return "stable"
}

// calculateAverage calculates the average value of data points.
func (ms *MetricsService) calculateAverage(dataPoints []DataPoint) float64 {
	if len(dataPoints) == 0 {
		return 0
	}

	sum := 0.0
	for _, point := range dataPoints {
		sum += point.Value
	}
	return sum / float64(len(dataPoints))
}

// calculateMinimum finds the minimum value in data points.
func (ms *MetricsService) calculateMinimum(dataPoints []DataPoint) float64 {
	if len(dataPoints) == 0 {
		return 0
	}

	minVal := dataPoints[0].Value
	for _, point := range dataPoints {
		if point.Value < minVal {
			minVal = point.Value
		}
	}
	return minVal
}

// calculateMaximum finds the maximum value in data points.
func (ms *MetricsService) calculateMaximum(dataPoints []DataPoint) float64 {
	if len(dataPoints) == 0 {
		return 0
	}

	maxVal := dataPoints[0].Value
	for _, point := range dataPoints {
		if point.Value > maxVal {
			maxVal = point.Value
		}
	}
	return maxVal
}

// getDefaultMetricsThresholds returns default alert thresholds for metrics.
func getDefaultMetricsThresholds() map[string]float64 {
	return map[string]float64{
		"response_time_warning_ms":    1000, // 1 second
		"response_time_critical_ms":   5000, // 5 seconds
		"error_rate_warning":          5.0,  // 5%
		"error_rate_critical":         15.0, // 15%
		"service_error_rate_critical": 20.0, // 20%
		"cpu_usage_warning":           80.0, // 80%
		"cpu_usage_critical":          95.0, // 95%
		"memory_usage_warning":        80.0, // 80%
		"memory_usage_critical":       95.0, // 95%
	}
}
