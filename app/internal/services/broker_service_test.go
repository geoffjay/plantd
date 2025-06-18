package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/geoffjay/plantd/app/config"
	"github.com/geoffjay/plantd/core/mdp"
	"github.com/stretchr/testify/assert"
)

func TestCircuitBreaker(t *testing.T) {
	t.Run("CircuitBreaker normal operation", func(t *testing.T) {
		cb := NewCircuitBreaker(3, 30*time.Second)

		// Normal operation should work
		err := cb.Call(func() error {
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, CircuitClosed, cb.state)
	})

	t.Run("CircuitBreaker opens after failures", func(t *testing.T) {
		cb := NewCircuitBreaker(2, 30*time.Second) // Lower limit for testing

		// First failure
		err := cb.Call(func() error {
			return assert.AnError
		})
		assert.Error(t, err)
		assert.Equal(t, CircuitClosed, cb.state)

		// Second failure should open circuit
		err = cb.Call(func() error {
			return assert.AnError
		})
		assert.Error(t, err)
		assert.Equal(t, CircuitOpen, cb.state)

		// Third call should fail immediately
		err = cb.Call(func() error {
			return nil // This won't be called
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "circuit breaker is open")
	})

	t.Run("CircuitBreaker resets after timeout", func(t *testing.T) {
		cb := NewCircuitBreaker(1, 100*time.Millisecond) // Short timeout for testing

		// Cause failure to open circuit
		err := cb.Call(func() error {
			return assert.AnError
		})
		assert.Error(t, err)
		assert.Equal(t, CircuitOpen, cb.state)

		// Wait for reset timeout
		time.Sleep(150 * time.Millisecond)

		// Next call should work (half-open state)
		err = cb.Call(func() error {
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, CircuitClosed, cb.state)
	})
}

func TestNewBrokerService(t *testing.T) {
	t.Run("successful creation with valid config", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.Services.BrokerEndpoint = "tcp://127.0.0.1:9797"
		cfg.Services.StateEndpoint = "tcp://127.0.0.1:9798"
		cfg.Services.Timeout = "30s"

		bs, err := NewBrokerService(cfg)

		assert.NoError(t, err)
		assert.NotNil(t, bs)
		assert.Equal(t, cfg, bs.config)
		assert.NotNil(t, bs.circuitBreaker)
		assert.NotNil(t, bs.logger)

		if bs.client != nil {
			err = bs.Close()
			assert.NoError(t, err)
		}
	})

	t.Run("creation with invalid broker endpoint", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.Services.BrokerEndpoint = "invalid://endpoint"
		cfg.Services.Timeout = "30s"

		bs, err := NewBrokerService(cfg)

		// Service should still be created but with broker functionality disabled
		assert.NoError(t, err)
		assert.NotNil(t, bs)
		assert.True(t, bs.disabled)
		assert.NotNil(t, bs.lastError)
	})

	t.Run("creation with empty broker endpoint", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.Services.BrokerEndpoint = ""
		cfg.Services.Timeout = "30s"

		bs, err := NewBrokerService(cfg)

		// Should use default endpoint
		assert.NoError(t, err)
		assert.NotNil(t, bs)

		if bs.client != nil {
			err = bs.Close()
			assert.NoError(t, err)
		}
	})

	t.Run("creation with invalid timeout", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.Services.BrokerEndpoint = "tcp://127.0.0.1:9797"
		cfg.Services.Timeout = "invalid"

		bs, err := NewBrokerService(cfg)

		// Should still work with default timeout
		assert.NoError(t, err)
		assert.NotNil(t, bs)

		if bs.client != nil {
			err = bs.Close()
			assert.NoError(t, err)
		}
	})
}

func TestBrokerService_IsAvailable(t *testing.T) {
	t.Run("available when not disabled and no client error", func(t *testing.T) {
		client := &mdp.Client{} // Mock client
		bs := &BrokerService{
			client:    client,
			disabled:  false,
			lastError: nil,
		}

		assert.True(t, bs.IsAvailable())
	})

	t.Run("not available when disabled", func(t *testing.T) {
		bs := &BrokerService{
			disabled:  true,
			lastError: nil,
		}

		assert.False(t, bs.IsAvailable())
	})

	t.Run("not available when has error", func(t *testing.T) {
		bs := &BrokerService{
			disabled:  false,
			lastError: assert.AnError,
		}

		assert.False(t, bs.IsAvailable())
	})
}

func TestBrokerService_GetStatus(t *testing.T) {
	t.Run("status when available", func(t *testing.T) {
		cfg := &config.Config{}
		cfg.Services.BrokerEndpoint = "tcp://127.0.0.1:9797"

		client := &mdp.Client{} // Mock client
		bs := &BrokerService{
			client:    client,
			config:    cfg,
			disabled:  false,
			lastError: nil,
		}

		status := bs.GetStatus()

		assert.Equal(t, "healthy", status["status"])
		assert.Equal(t, cfg.Services.BrokerEndpoint, status["endpoint"])
		assert.Nil(t, status["last_error"])
	})

	t.Run("status when disabled", func(t *testing.T) {
		bs := &BrokerService{
			disabled:  true,
			lastError: assert.AnError,
		}

		status := bs.GetStatus()

		assert.Equal(t, "disabled", status["status"])
		assert.Equal(t, assert.AnError.Error(), status["last_error"])
	})
}

func TestBrokerService_CheckConnectivity(t *testing.T) {
	t.Run("connectivity check when disabled", func(t *testing.T) {
		bs := &BrokerService{
			disabled:  true,
			lastError: fmt.Errorf("test error"),
		}

		ctx := context.Background()
		err := bs.CheckConnectivity(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "broker service is disabled")
	})

	t.Run("connectivity check with circuit breaker failure", func(t *testing.T) {
		cb := &CircuitBreaker{
			state: CircuitOpen,
		}

		bs := &BrokerService{
			disabled:       false,
			circuitBreaker: cb,
		}

		ctx := context.Background()
		err := bs.CheckConnectivity(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "circuit breaker is open")
	})
}

func TestBrokerService_GetServiceStatuses(t *testing.T) {
	t.Run("get service statuses when disabled", func(t *testing.T) {
		bs := &BrokerService{
			disabled: true,
		}

		ctx := context.Background()
		statuses, err := bs.GetServiceStatuses(ctx)

		assert.NoError(t, err) // Should not error, just return empty
		assert.Empty(t, statuses)
	})
}

func TestBrokerService_Close(t *testing.T) {
	t.Run("close with nil client", func(t *testing.T) {
		bs := &BrokerService{
			client: nil,
		}

		err := bs.Close()
		assert.NoError(t, err)
	})
}

func TestBrokerService_parseSimpleMetrics(t *testing.T) {
	bs := &BrokerService{}

	t.Run("parse valid metrics", func(t *testing.T) {
		response := []string{
			"messages_processed:1000",
			"avg_response_time:50ms",
			"error_count:5",
			"active_connections:10",
		}

		metrics := bs.parseSimpleMetrics(response)

		assert.NotNil(t, metrics)
		assert.Equal(t, int64(1000), metrics.MessagesProcessed)
		assert.Equal(t, 50*time.Millisecond, metrics.AvgResponseTime)
		assert.Equal(t, int64(5), metrics.ErrorCount)
		assert.Equal(t, 10, metrics.ActiveConnections)
	})

	t.Run("parse metrics with invalid data", func(t *testing.T) {
		response := []string{
			"messages_processed:invalid",
			"avg_response_time:invalid",
			"error_count:invalid",
			"active_connections:invalid",
		}

		metrics := bs.parseSimpleMetrics(response)

		// Should return zero values for invalid data
		assert.NotNil(t, metrics)
		assert.Equal(t, int64(0), metrics.MessagesProcessed)
		assert.Equal(t, time.Duration(0), metrics.AvgResponseTime)
		assert.Equal(t, int64(0), metrics.ErrorCount)
		assert.Equal(t, 0, metrics.ActiveConnections)
	})

	t.Run("parse empty metrics", func(t *testing.T) {
		response := []string{}

		metrics := bs.parseSimpleMetrics(response)

		assert.NotNil(t, metrics)
		assert.Equal(t, int64(0), metrics.MessagesProcessed)
		assert.Equal(t, time.Duration(0), metrics.AvgResponseTime)
		assert.Equal(t, int64(0), metrics.ErrorCount)
		assert.Equal(t, 0, metrics.ActiveConnections)
	})
}

// Benchmark tests for performance validation
func BenchmarkCircuitBreaker_Success(b *testing.B) {
	cb := NewCircuitBreaker(3, 30*time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cb.Call(func() error {
			return nil
		})
	}
}

func BenchmarkCircuitBreaker_Failure(b *testing.B) {
	cb := NewCircuitBreaker(1000, 30*time.Second) // High limit to avoid opening

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cb.Call(func() error {
			return assert.AnError
		})
	}
}
