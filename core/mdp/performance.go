package mdp

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/zeromq/goczmq/v4"
)

// ConnectionPool manages a pool of ZeroMQ connections for improved performance
type ConnectionPool struct {
	mu          sync.RWMutex
	connections map[string]*PooledConnection
	maxSize     int
	idleTimeout time.Duration
	cleanup     *time.Ticker
	ctx         context.Context
	cancel      context.CancelFunc
}

// PooledConnection represents a connection in the pool
type PooledConnection struct {
	Socket    *goczmq.Sock
	Endpoint  string
	LastUsed  time.Time
	InUse     bool
	CreatedAt time.Time
	UseCount  int64
}

// ConnectionPoolConfig defines connection pool configuration
type ConnectionPoolConfig struct {
	MaxSize         int           `json:"max_size"`
	IdleTimeout     time.Duration `json:"idle_timeout"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(config ConnectionPoolConfig) *ConnectionPool {
	ctx, cancel := context.WithCancel(context.Background())

	pool := &ConnectionPool{
		connections: make(map[string]*PooledConnection),
		maxSize:     config.MaxSize,
		idleTimeout: config.IdleTimeout,
		cleanup:     time.NewTicker(config.CleanupInterval),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Start cleanup goroutine
	go pool.cleanupRoutine()

	return pool
}

// GetConnection retrieves or creates a connection from the pool
func (cp *ConnectionPool) GetConnection(endpoint string, socketType int) (*PooledConnection, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Check if connection exists and is available
	if conn, exists := cp.connections[endpoint]; exists && !conn.InUse {
		conn.InUse = true
		conn.LastUsed = time.Now()
		atomic.AddInt64(&conn.UseCount, 1)
		return conn, nil
	}

	// Check pool size limit
	if len(cp.connections) >= cp.maxSize {
		return nil, ErrConnectionPoolFull
	}

	// Create new connection
	var socket *goczmq.Sock
	var err error

	switch socketType {
	case goczmq.Dealer:
		socket, err = goczmq.NewDealer(endpoint)
	case goczmq.Router:
		socket, err = goczmq.NewRouter(endpoint)
	case goczmq.Req:
		socket, err = goczmq.NewReq(endpoint)
	case goczmq.Rep:
		socket, err = goczmq.NewRep(endpoint)
	default:
		return nil, ErrUnsupportedSocketType
	}

	if err != nil {
		return nil, err
	}

	conn := &PooledConnection{
		Socket:    socket,
		Endpoint:  endpoint,
		LastUsed:  time.Now(),
		InUse:     true,
		CreatedAt: time.Now(),
		UseCount:  1,
	}

	cp.connections[endpoint] = conn

	log.WithFields(log.Fields{
		"endpoint":  endpoint,
		"pool_size": len(cp.connections),
	}).Debug("created new pooled connection")

	return conn, nil
}

// ReleaseConnection returns a connection to the pool
func (cp *ConnectionPool) ReleaseConnection(endpoint string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if conn, exists := cp.connections[endpoint]; exists {
		conn.InUse = false
		conn.LastUsed = time.Now()
	}
}

// Close closes all connections and shuts down the pool
func (cp *ConnectionPool) Close() error {
	cp.cancel()
	cp.cleanup.Stop()

	cp.mu.Lock()
	defer cp.mu.Unlock()

	for endpoint, conn := range cp.connections {
		if conn.Socket != nil {
			conn.Socket.Destroy()
		}
		delete(cp.connections, endpoint)
	}

	log.Info("connection pool closed")
	return nil
}

// cleanupRoutine periodically removes idle connections
func (cp *ConnectionPool) cleanupRoutine() {
	for {
		select {
		case <-cp.ctx.Done():
			return
		case <-cp.cleanup.C:
			cp.cleanupIdleConnections()
		}
	}
}

// cleanupIdleConnections removes connections that have been idle too long
func (cp *ConnectionPool) cleanupIdleConnections() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	now := time.Now()
	for endpoint, conn := range cp.connections {
		if !conn.InUse && now.Sub(conn.LastUsed) > cp.idleTimeout {
			if conn.Socket != nil {
				conn.Socket.Destroy()
			}
			delete(cp.connections, endpoint)

			log.WithFields(log.Fields{
				"endpoint":  endpoint,
				"idle_time": now.Sub(conn.LastUsed),
			}).Debug("removed idle connection from pool")
		}
	}
}

// GetStats returns connection pool statistics
func (cp *ConnectionPool) GetStats() map[string]interface{} {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	stats := make(map[string]interface{})
	totalConnections := len(cp.connections)
	inUseConnections := 0
	totalUseCount := int64(0)

	for _, conn := range cp.connections {
		if conn.InUse {
			inUseConnections++
		}
		totalUseCount += conn.UseCount
	}

	stats["total_connections"] = totalConnections
	stats["in_use_connections"] = inUseConnections
	stats["idle_connections"] = totalConnections - inUseConnections
	stats["total_use_count"] = totalUseCount
	stats["max_size"] = cp.maxSize

	return stats
}

// MessageBatcher batches multiple messages for efficient transmission
type MessageBatcher struct {
	mu            sync.Mutex
	batches       map[string]*Batch
	maxBatchSize  int
	flushInterval time.Duration
	flushTimer    *time.Timer
	flushFunc     func(string, [][]string) error
	ctx           context.Context
	cancel        context.CancelFunc
}

// Batch represents a collection of messages for a specific destination
type Batch struct {
	Destination string
	Messages    [][]string
	CreatedAt   time.Time
	LastAdded   time.Time
}

// MessageBatcherConfig defines message batcher configuration
type MessageBatcherConfig struct {
	MaxBatchSize  int           `json:"max_batch_size"`
	FlushInterval time.Duration `json:"flush_interval"`
}

// NewMessageBatcher creates a new message batcher
func NewMessageBatcher(config MessageBatcherConfig, flushFunc func(string, [][]string) error) *MessageBatcher {
	ctx, cancel := context.WithCancel(context.Background())

	batcher := &MessageBatcher{
		batches:       make(map[string]*Batch),
		maxBatchSize:  config.MaxBatchSize,
		flushInterval: config.FlushInterval,
		flushFunc:     flushFunc,
		ctx:           ctx,
		cancel:        cancel,
	}

	batcher.flushTimer = time.AfterFunc(config.FlushInterval, batcher.flushAll)

	return batcher
}

// AddMessage adds a message to the appropriate batch
func (mb *MessageBatcher) AddMessage(destination string, message []string) error {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	batch, exists := mb.batches[destination]
	if !exists {
		batch = &Batch{
			Destination: destination,
			Messages:    make([][]string, 0, mb.maxBatchSize),
			CreatedAt:   time.Now(),
		}
		mb.batches[destination] = batch
	}

	batch.Messages = append(batch.Messages, message)
	batch.LastAdded = time.Now()

	// Flush if batch is full
	if len(batch.Messages) >= mb.maxBatchSize {
		return mb.flushBatch(destination)
	}

	return nil
}

// flushBatch flushes a specific batch
func (mb *MessageBatcher) flushBatch(destination string) error {
	batch, exists := mb.batches[destination]
	if !exists || len(batch.Messages) == 0 {
		return nil
	}

	messages := make([][]string, len(batch.Messages))
	copy(messages, batch.Messages)

	// Clear the batch
	delete(mb.batches, destination)

	// Release lock before calling flush function
	mb.mu.Unlock()
	err := mb.flushFunc(destination, messages)
	mb.mu.Lock()

	if err != nil {
		log.WithFields(log.Fields{
			"destination": destination,
			"batch_size":  len(messages),
			"error":       err,
		}).Error("failed to flush message batch")
		return err
	}

	log.WithFields(log.Fields{
		"destination": destination,
		"batch_size":  len(messages),
	}).Debug("flushed message batch")

	return nil
}

// flushAll flushes all batches
func (mb *MessageBatcher) flushAll() {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	for destination := range mb.batches {
		_ = mb.flushBatch(destination)
	}

	// Reset timer
	mb.flushTimer.Reset(mb.flushInterval)
}

// Flush manually flushes all pending batches
func (mb *MessageBatcher) Flush() error {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	var lastError error
	for destination := range mb.batches {
		if err := mb.flushBatch(destination); err != nil {
			lastError = err
		}
	}

	return lastError
}

// Close shuts down the message batcher
func (mb *MessageBatcher) Close() error {
	mb.cancel()

	if mb.flushTimer != nil {
		mb.flushTimer.Stop()
	}

	// Flush any remaining messages
	return mb.Flush()
}

// GetStats returns message batcher statistics
func (mb *MessageBatcher) GetStats() map[string]interface{} {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	stats := make(map[string]interface{})
	totalBatches := len(mb.batches)
	totalMessages := 0

	batchSizes := make(map[string]int)
	for destination, batch := range mb.batches {
		batchSize := len(batch.Messages)
		totalMessages += batchSize
		batchSizes[destination] = batchSize
	}

	stats["total_batches"] = totalBatches
	stats["total_messages"] = totalMessages
	stats["batch_sizes"] = batchSizes
	stats["max_batch_size"] = mb.maxBatchSize

	return stats
}

// PerformanceMetrics collects and tracks performance metrics
type PerformanceMetrics struct {
	mu               sync.RWMutex
	MessagesSent     int64
	MessagesReceived int64
	BytesSent        int64
	BytesReceived    int64
	RequestLatencies []time.Duration
	ErrorCount       int64
	StartTime        time.Time
	LastResetTime    time.Time
}

// NewPerformanceMetrics creates a new performance metrics collector
func NewPerformanceMetrics() *PerformanceMetrics {
	now := time.Now()
	return &PerformanceMetrics{
		RequestLatencies: make([]time.Duration, 0, 1000), // Keep last 1000 latencies
		StartTime:        now,
		LastResetTime:    now,
	}
}

// RecordMessageSent records a sent message
func (pm *PerformanceMetrics) RecordMessageSent(bytes int64) {
	atomic.AddInt64(&pm.MessagesSent, 1)
	atomic.AddInt64(&pm.BytesSent, bytes)
}

// RecordMessageReceived records a received message
func (pm *PerformanceMetrics) RecordMessageReceived(bytes int64) {
	atomic.AddInt64(&pm.MessagesReceived, 1)
	atomic.AddInt64(&pm.BytesReceived, bytes)
}

// RecordRequestLatency records request latency
func (pm *PerformanceMetrics) RecordRequestLatency(latency time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Keep only the last 1000 latencies to prevent memory growth
	if len(pm.RequestLatencies) >= 1000 {
		pm.RequestLatencies = pm.RequestLatencies[1:]
	}
	pm.RequestLatencies = append(pm.RequestLatencies, latency)
}

// RecordError records an error occurrence
func (pm *PerformanceMetrics) RecordError() {
	atomic.AddInt64(&pm.ErrorCount, 1)
}

// GetStats returns current performance statistics
func (pm *PerformanceMetrics) GetStats() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	now := time.Now()
	uptime := now.Sub(pm.StartTime)
	timeSinceReset := now.Sub(pm.LastResetTime)

	stats := make(map[string]interface{})
	stats["messages_sent"] = atomic.LoadInt64(&pm.MessagesSent)
	stats["messages_received"] = atomic.LoadInt64(&pm.MessagesReceived)
	stats["bytes_sent"] = atomic.LoadInt64(&pm.BytesSent)
	stats["bytes_received"] = atomic.LoadInt64(&pm.BytesReceived)
	stats["error_count"] = atomic.LoadInt64(&pm.ErrorCount)
	stats["uptime_seconds"] = uptime.Seconds()
	stats["time_since_reset_seconds"] = timeSinceReset.Seconds()

	// Calculate latency statistics
	if len(pm.RequestLatencies) > 0 {
		var total time.Duration
		minLatency := pm.RequestLatencies[0]
		maxLatency := pm.RequestLatencies[0]

		for _, latency := range pm.RequestLatencies {
			total += latency
			if latency < minLatency {
				minLatency = latency
			}
			if latency > maxLatency {
				maxLatency = latency
			}
		}

		avg := total / time.Duration(len(pm.RequestLatencies))
		stats["avg_latency_ms"] = float64(avg.Nanoseconds()) / 1e6
		stats["min_latency_ms"] = float64(minLatency.Nanoseconds()) / 1e6
		stats["max_latency_ms"] = float64(maxLatency.Nanoseconds()) / 1e6
		stats["latency_samples"] = len(pm.RequestLatencies)
	}

	// Calculate throughput
	if timeSinceReset.Seconds() > 0 {
		stats["messages_per_second"] = float64(atomic.LoadInt64(&pm.MessagesSent)) / timeSinceReset.Seconds()
		stats["bytes_per_second"] = float64(atomic.LoadInt64(&pm.BytesSent)) / timeSinceReset.Seconds()
	}

	return stats
}

// Reset resets all metrics
func (pm *PerformanceMetrics) Reset() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	atomic.StoreInt64(&pm.MessagesSent, 0)
	atomic.StoreInt64(&pm.MessagesReceived, 0)
	atomic.StoreInt64(&pm.BytesSent, 0)
	atomic.StoreInt64(&pm.BytesReceived, 0)
	atomic.StoreInt64(&pm.ErrorCount, 0)
	pm.RequestLatencies = pm.RequestLatencies[:0]
	pm.LastResetTime = time.Now()
}

// PerformanceOptimizer combines all performance optimizations
type PerformanceOptimizer struct {
	ConnectionPool *ConnectionPool
	MessageBatcher *MessageBatcher
	Metrics        *PerformanceMetrics
	config         PerformanceConfig
}

// PerformanceConfig defines overall performance configuration
type PerformanceConfig struct {
	ConnectionPool ConnectionPoolConfig `json:"connection_pool"`
	MessageBatcher MessageBatcherConfig `json:"message_batcher"`
	EnableMetrics  bool                 `json:"enable_metrics"`
}

// NewPerformanceOptimizer creates a new performance optimizer
func NewPerformanceOptimizer(config PerformanceConfig, flushFunc func(string, [][]string) error) *PerformanceOptimizer {
	optimizer := &PerformanceOptimizer{
		ConnectionPool: NewConnectionPool(config.ConnectionPool),
		MessageBatcher: NewMessageBatcher(config.MessageBatcher, flushFunc),
		config:         config,
	}

	if config.EnableMetrics {
		optimizer.Metrics = NewPerformanceMetrics()
	}

	return optimizer
}

// Close shuts down all performance optimizations
func (po *PerformanceOptimizer) Close() error {
	var lastError error

	if po.ConnectionPool != nil {
		if err := po.ConnectionPool.Close(); err != nil {
			lastError = err
		}
	}

	if po.MessageBatcher != nil {
		if err := po.MessageBatcher.Close(); err != nil {
			lastError = err
		}
	}

	return lastError
}

// GetCombinedStats returns statistics from all performance components
func (po *PerformanceOptimizer) GetCombinedStats() map[string]interface{} {
	stats := make(map[string]interface{})

	if po.ConnectionPool != nil {
		stats["connection_pool"] = po.ConnectionPool.GetStats()
	}

	if po.MessageBatcher != nil {
		stats["message_batcher"] = po.MessageBatcher.GetStats()
	}

	if po.Metrics != nil {
		stats["metrics"] = po.Metrics.GetStats()
	}

	return stats
}

// Error definitions for performance optimizations
var (
	ErrConnectionPoolFull    = NewMDPError("POOL_FULL", "connection pool is full", nil)
	ErrUnsupportedSocketType = NewMDPError("UNSUPPORTED_SOCKET", "unsupported socket type", nil)
)
