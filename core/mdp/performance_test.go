package mdp

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromq/goczmq/v4"
)

func TestConnectionPool(t *testing.T) {
	config := ConnectionPoolConfig{
		MaxSize:         5,
		IdleTimeout:     30 * time.Second,
		CleanupInterval: 10 * time.Second,
	}

	pool := NewConnectionPool(config)
	defer pool.Close()

	t.Run("GetConnection", func(t *testing.T) {
		// Skip this test if we can't create sockets (CI environment)
		conn, err := pool.GetConnection("inproc://test", goczmq.Dealer)
		if err != nil {
			t.Skip("Cannot create ZMQ sockets in test environment")
			return
		}

		assert.NotNil(t, conn)
		assert.Equal(t, "inproc://test", conn.Endpoint)
		assert.True(t, conn.InUse)
		assert.Equal(t, int64(1), conn.UseCount)

		// Release connection
		pool.ReleaseConnection("inproc://test")
		assert.False(t, conn.InUse)
	})

	t.Run("ReuseConnection", func(t *testing.T) {
		// Skip if we can't create sockets
		conn1, err := pool.GetConnection("inproc://reuse", goczmq.Dealer)
		if err != nil {
			t.Skip("Cannot create ZMQ sockets in test environment")
			return
		}

		pool.ReleaseConnection("inproc://reuse")

		conn2, err := pool.GetConnection("inproc://reuse", goczmq.Dealer)
		require.NoError(t, err)

		// Should be the same connection
		assert.Equal(t, conn1.Socket, conn2.Socket)
		assert.Equal(t, int64(2), conn2.UseCount)
	})

	t.Run("MaxSizeLimit", func(t *testing.T) {
		// Create connections up to max size
		endpoints := make([]string, config.MaxSize+1)
		for i := 0; i < config.MaxSize+1; i++ {
			endpoints[i] = fmt.Sprintf("inproc://max-%d", i)
		}

		// Fill up the pool
		for i := 0; i < config.MaxSize; i++ {
			_, err := pool.GetConnection(endpoints[i], goczmq.Dealer)
			if err != nil {
				t.Skip("Cannot create ZMQ sockets in test environment")
				return
			}
		}

		// Next connection should fail
		_, err := pool.GetConnection(endpoints[config.MaxSize], goczmq.Dealer)
		assert.Error(t, err)
		assert.Equal(t, ErrConnectionPoolFull, err)
	})

	t.Run("GetStats", func(t *testing.T) {
		stats := pool.GetStats()

		assert.Contains(t, stats, "total_connections")
		assert.Contains(t, stats, "in_use_connections")
		assert.Contains(t, stats, "idle_connections")
		assert.Contains(t, stats, "total_use_count")
		assert.Contains(t, stats, "max_size")

		assert.Equal(t, config.MaxSize, stats["max_size"])
	})

	t.Run("IdleCleanup", func(t *testing.T) {
		// Create a pool with short idle timeout for testing
		shortConfig := ConnectionPoolConfig{
			MaxSize:         5,
			IdleTimeout:     100 * time.Millisecond,
			CleanupInterval: 50 * time.Millisecond,
		}

		shortPool := NewConnectionPool(shortConfig)
		defer shortPool.Close()

		// Create and release a connection
		_, err := shortPool.GetConnection("inproc://idle", goczmq.Dealer)
		if err != nil {
			t.Skip("Cannot create ZMQ sockets in test environment")
			return
		}

		shortPool.ReleaseConnection("inproc://idle")

		// Wait for cleanup
		time.Sleep(200 * time.Millisecond)

		// Manually trigger cleanup
		shortPool.cleanupIdleConnections()

		stats := shortPool.GetStats()
		assert.Equal(t, 0, stats["total_connections"])
	})
}

func TestMessageBatcher(t *testing.T) {
	config := MessageBatcherConfig{
		MaxBatchSize:  3,
		FlushInterval: 100 * time.Millisecond,
	}

	t.Run("AddMessage", func(t *testing.T) {
		var flushedBatches []struct {
			destination string
			messages    [][]string
		}

		flushFunc := func(destination string, messages [][]string) error {
			flushedBatches = append(flushedBatches, struct {
				destination string
				messages    [][]string
			}{destination, messages})
			return nil
		}

		batcher := NewMessageBatcher(config, flushFunc)
		defer batcher.Close()

		err := batcher.AddMessage("dest1", []string{"msg1", "data1"})
		assert.NoError(t, err)

		err = batcher.AddMessage("dest1", []string{"msg2", "data2"})
		assert.NoError(t, err)

		stats := batcher.GetStats()
		assert.Equal(t, 1, stats["total_batches"])
		assert.Equal(t, 2, stats["total_messages"])
	})

	t.Run("BatchSizeFlush", func(t *testing.T) {
		var flushedBatches []struct {
			destination string
			messages    [][]string
		}
		var flushMutex sync.Mutex

		flushFunc := func(destination string, messages [][]string) error {
			flushMutex.Lock()
			defer flushMutex.Unlock()
			flushedBatches = append(flushedBatches, struct {
				destination string
				messages    [][]string
			}{destination, messages})
			return nil
		}

		batcher := NewMessageBatcher(config, flushFunc)
		defer batcher.Close()

		// Add messages to trigger batch size flush
		for i := 0; i < config.MaxBatchSize; i++ {
			err := batcher.AddMessage("dest2", []string{fmt.Sprintf("msg%d", i)})
			assert.NoError(t, err)
		}

		// Should have flushed automatically
		flushMutex.Lock()
		assert.Len(t, flushedBatches, 1)
		assert.Equal(t, "dest2", flushedBatches[0].destination)
		assert.Len(t, flushedBatches[0].messages, config.MaxBatchSize)
		flushMutex.Unlock()
	})

	t.Run("TimeBasedFlush", func(t *testing.T) {
		var flushedBatches []struct {
			destination string
			messages    [][]string
		}
		var flushMutex sync.Mutex

		flushFunc := func(destination string, messages [][]string) error {
			flushMutex.Lock()
			defer flushMutex.Unlock()
			flushedBatches = append(flushedBatches, struct {
				destination string
				messages    [][]string
			}{destination, messages})
			return nil
		}

		batcher := NewMessageBatcher(config, flushFunc)
		defer batcher.Close()

		// Add a single message
		err := batcher.AddMessage("dest3", []string{"single", "message"})
		assert.NoError(t, err)

		// Wait for time-based flush
		time.Sleep(150 * time.Millisecond)

		flushMutex.Lock()
		assert.Len(t, flushedBatches, 1)
		assert.Equal(t, "dest3", flushedBatches[0].destination)
		assert.Len(t, flushedBatches[0].messages, 1)
		flushMutex.Unlock()
	})

	t.Run("ManualFlush", func(t *testing.T) {
		var flushedBatches []struct {
			destination string
			messages    [][]string
		}
		var flushMutex sync.Mutex

		flushFunc := func(destination string, messages [][]string) error {
			flushMutex.Lock()
			defer flushMutex.Unlock()
			flushedBatches = append(flushedBatches, struct {
				destination string
				messages    [][]string
			}{destination, messages})
			return nil
		}

		batcher := NewMessageBatcher(config, flushFunc)
		defer batcher.Close()

		// Add messages to multiple destinations
		err := batcher.AddMessage("dest4", []string{"msg1"})
		assert.NoError(t, err)
		err = batcher.AddMessage("dest5", []string{"msg2"})
		assert.NoError(t, err)

		// Manual flush
		err = batcher.Flush()
		assert.NoError(t, err)

		flushMutex.Lock()
		assert.Len(t, flushedBatches, 2)
		flushMutex.Unlock()
	})

	t.Run("GetStats", func(t *testing.T) {
		flushFunc := func(destination string, messages [][]string) error {
			return nil
		}

		batcher := NewMessageBatcher(config, flushFunc)
		defer batcher.Close()

		// Add some messages
		batcher.AddMessage("stats1", []string{"msg1"})
		batcher.AddMessage("stats2", []string{"msg2"})

		stats := batcher.GetStats()
		assert.Contains(t, stats, "total_batches")
		assert.Contains(t, stats, "total_messages")
		assert.Contains(t, stats, "batch_sizes")
		assert.Contains(t, stats, "max_batch_size")

		assert.Equal(t, config.MaxBatchSize, stats["max_batch_size"])
	})
}

func TestPerformanceMetrics(t *testing.T) {
	metrics := NewPerformanceMetrics()

	t.Run("RecordMessageSent", func(t *testing.T) {
		metrics.RecordMessageSent(100)
		metrics.RecordMessageSent(200)

		stats := metrics.GetStats()
		assert.Equal(t, int64(2), stats["messages_sent"])
		assert.Equal(t, int64(300), stats["bytes_sent"])
	})

	t.Run("RecordMessageReceived", func(t *testing.T) {
		metrics.RecordMessageReceived(150)
		metrics.RecordMessageReceived(250)

		stats := metrics.GetStats()
		assert.Equal(t, int64(2), stats["messages_received"])
		assert.Equal(t, int64(400), stats["bytes_received"])
	})

	t.Run("RecordRequestLatency", func(t *testing.T) {
		metrics.RecordRequestLatency(10 * time.Millisecond)
		metrics.RecordRequestLatency(20 * time.Millisecond)
		metrics.RecordRequestLatency(30 * time.Millisecond)

		stats := metrics.GetStats()
		assert.Contains(t, stats, "avg_latency_ms")
		assert.Contains(t, stats, "min_latency_ms")
		assert.Contains(t, stats, "max_latency_ms")
		assert.Contains(t, stats, "latency_samples")

		assert.Equal(t, 3, stats["latency_samples"])
		assert.Equal(t, float64(10), stats["min_latency_ms"])
		assert.Equal(t, float64(30), stats["max_latency_ms"])
		assert.Equal(t, float64(20), stats["avg_latency_ms"])
	})

	t.Run("RecordError", func(t *testing.T) {
		metrics.RecordError()
		metrics.RecordError()

		stats := metrics.GetStats()
		assert.Equal(t, int64(2), stats["error_count"])
	})

	t.Run("LatencyBufferLimit", func(t *testing.T) {
		// Add more than 1000 latencies to test buffer limit
		for i := 0; i < 1200; i++ {
			metrics.RecordRequestLatency(time.Duration(i) * time.Millisecond)
		}

		stats := metrics.GetStats()
		assert.Equal(t, 1000, stats["latency_samples"]) // Should be capped at 1000
	})

	t.Run("Reset", func(t *testing.T) {
		metrics.Reset()

		stats := metrics.GetStats()
		assert.Equal(t, int64(0), stats["messages_sent"])
		assert.Equal(t, int64(0), stats["messages_received"])
		assert.Equal(t, int64(0), stats["bytes_sent"])
		assert.Equal(t, int64(0), stats["bytes_received"])
		assert.Equal(t, int64(0), stats["error_count"])
	})

	t.Run("Throughput", func(t *testing.T) {
		// Reset and add some data
		metrics.Reset()
		time.Sleep(10 * time.Millisecond) // Ensure some time passes

		metrics.RecordMessageSent(1000)
		metrics.RecordMessageSent(2000)

		stats := metrics.GetStats()
		assert.Contains(t, stats, "messages_per_second")
		assert.Contains(t, stats, "bytes_per_second")

		// Should have positive throughput values
		assert.Greater(t, stats["messages_per_second"].(float64), 0.0)
		assert.Greater(t, stats["bytes_per_second"].(float64), 0.0)
	})
}

func TestPerformanceOptimizer(t *testing.T) {
	flushFunc := func(destination string, messages [][]string) error {
		return nil
	}

	config := PerformanceConfig{
		ConnectionPool: ConnectionPoolConfig{
			MaxSize:         10,
			IdleTimeout:     30 * time.Second,
			CleanupInterval: 10 * time.Second,
		},
		MessageBatcher: MessageBatcherConfig{
			MaxBatchSize:  5,
			FlushInterval: 100 * time.Millisecond,
		},
		EnableMetrics: true,
	}

	optimizer := NewPerformanceOptimizer(config, flushFunc)
	defer optimizer.Close()

	t.Run("Initialization", func(t *testing.T) {
		assert.NotNil(t, optimizer.ConnectionPool)
		assert.NotNil(t, optimizer.MessageBatcher)
		assert.NotNil(t, optimizer.Metrics)
	})

	t.Run("GetCombinedStats", func(t *testing.T) {
		stats := optimizer.GetCombinedStats()

		assert.Contains(t, stats, "connection_pool")
		assert.Contains(t, stats, "message_batcher")
		assert.Contains(t, stats, "metrics")

		// Verify nested stats structure
		poolStats := stats["connection_pool"].(map[string]interface{})
		assert.Contains(t, poolStats, "total_connections")
		assert.Contains(t, poolStats, "max_size")

		batcherStats := stats["message_batcher"].(map[string]interface{})
		assert.Contains(t, batcherStats, "total_batches")
		assert.Contains(t, batcherStats, "max_batch_size")

		metricsStats := stats["metrics"].(map[string]interface{})
		assert.Contains(t, metricsStats, "messages_sent")
		assert.Contains(t, metricsStats, "uptime_seconds")
	})

	t.Run("DisabledMetrics", func(t *testing.T) {
		configNoMetrics := config
		configNoMetrics.EnableMetrics = false

		optimizerNoMetrics := NewPerformanceOptimizer(configNoMetrics, flushFunc)
		defer optimizerNoMetrics.Close()

		assert.Nil(t, optimizerNoMetrics.Metrics)

		stats := optimizerNoMetrics.GetCombinedStats()
		assert.NotContains(t, stats, "metrics")
	})
}

func BenchmarkPerformanceComponents(b *testing.B) {
	b.Run("ConnectionPool", func(b *testing.B) {
		config := ConnectionPoolConfig{
			MaxSize:         100,
			IdleTimeout:     30 * time.Second,
			CleanupInterval: 10 * time.Second,
		}

		pool := NewConnectionPool(config)
		defer pool.Close()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				endpoint := fmt.Sprintf("inproc://bench-%d", i%10)
				_, err := pool.GetConnection(endpoint, goczmq.Dealer)
				if err == nil {
					pool.ReleaseConnection(endpoint)
				}
				i++
			}
		})
	})

	b.Run("MessageBatcher", func(b *testing.B) {
		flushFunc := func(destination string, messages [][]string) error {
			return nil
		}

		config := MessageBatcherConfig{
			MaxBatchSize:  100,
			FlushInterval: 1 * time.Second,
		}

		batcher := NewMessageBatcher(config, flushFunc)
		defer batcher.Close()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				destination := fmt.Sprintf("dest-%d", i%5)
				message := []string{fmt.Sprintf("msg-%d", i), "data"}
				_ = batcher.AddMessage(destination, message)
				i++
			}
		})
	})

	b.Run("PerformanceMetrics", func(b *testing.B) {
		metrics := NewPerformanceMetrics()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				metrics.RecordMessageSent(100)
				metrics.RecordMessageReceived(150)
				metrics.RecordRequestLatency(10 * time.Millisecond)
			}
		})
	})

	b.Run("CombinedOptimizer", func(b *testing.B) {
		flushFunc := func(destination string, messages [][]string) error {
			return nil
		}

		config := PerformanceConfig{
			ConnectionPool: ConnectionPoolConfig{
				MaxSize:         50,
				IdleTimeout:     30 * time.Second,
				CleanupInterval: 10 * time.Second,
			},
			MessageBatcher: MessageBatcherConfig{
				MaxBatchSize:  50,
				FlushInterval: 1 * time.Second,
			},
			EnableMetrics: true,
		}

		optimizer := NewPerformanceOptimizer(config, flushFunc)
		defer optimizer.Close()

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				// Simulate typical operations
				if optimizer.Metrics != nil {
					optimizer.Metrics.RecordMessageSent(100)
				}

				destination := fmt.Sprintf("dest-%d", i%3)
				message := []string{fmt.Sprintf("msg-%d", i)}
				_ = optimizer.MessageBatcher.AddMessage(destination, message)

				i++
			}
		})
	})
}
