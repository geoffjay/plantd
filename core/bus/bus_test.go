package bus

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBus(t *testing.T) {
	config := Config{
		Name:     "test-bus",
		Unit:     "test-unit",
		Backend:  "inproc://backend",
		Frontend: "inproc://frontend",
		Capture:  "inproc://capture",
	}

	bus := NewBus(config)

	assert.NotNil(t, bus)
	assert.Equal(t, "test-bus", bus.name)
	assert.Equal(t, "test-unit", bus.unit)
	assert.Equal(t, "inproc://backend", bus.backend)
	assert.Equal(t, "inproc://frontend", bus.frontend)
	assert.Equal(t, "inproc://capture", bus.capture)
}

func TestBusConfig(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		config := Config{}
		bus := NewBus(config)

		assert.NotNil(t, bus)
		assert.Empty(t, bus.name)
		assert.Empty(t, bus.unit)
		assert.Empty(t, bus.backend)
		assert.Empty(t, bus.frontend)
		assert.Empty(t, bus.capture)
	})

	t.Run("partial config", func(t *testing.T) {
		config := Config{
			Name:    "partial-bus",
			Backend: "tcp://localhost:5555",
		}
		bus := NewBus(config)

		assert.NotNil(t, bus)
		assert.Equal(t, "partial-bus", bus.name)
		assert.Equal(t, "tcp://localhost:5555", bus.backend)
		assert.Empty(t, bus.unit)
		assert.Empty(t, bus.frontend)
		assert.Empty(t, bus.capture)
	})
}

func TestBusStart(t *testing.T) {
	t.Run("context cancellation", func(t *testing.T) {
		config := Config{
			Name:     "test-bus",
			Backend:  "inproc://test-backend",
			Frontend: "inproc://test-frontend",
			Capture:  "inproc://test-capture",
		}
		bus := NewBus(config)

		ctx, cancel := context.WithCancel(context.Background())
		var wg sync.WaitGroup
		wg.Add(1)

		// Channel to capture error from goroutine
		errChan := make(chan error, 1)

		// Start the bus in a goroutine
		go func() {
			defer wg.Done()
			err := bus.Start(ctx, &wg)
			errChan <- err
		}()

		// Give it a moment to start
		time.Sleep(100 * time.Millisecond)

		// Cancel context
		cancel()

		// Wait for completion
		wg.Wait()

		// Check error
		select {
		case err := <-errChan:
			assert.NoError(t, err)
		default:
			// No error received, which is also fine
		}
	})

	t.Run("context with timeout", func(t *testing.T) {
		config := Config{
			Name:     "timeout-bus",
			Backend:  "inproc://timeout-backend",
			Frontend: "inproc://timeout-frontend",
			Capture:  "inproc://timeout-capture",
		}
		bus := NewBus(config)

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)

		start := time.Now()
		err := bus.Start(ctx, &wg)
		duration := time.Since(start)

		// Should exit cleanly due to timeout
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, duration, 200*time.Millisecond)
	})
}

func TestBusRun(t *testing.T) {
	t.Run("deprecated run method", func(t *testing.T) {
		config := Config{
			Name:     "deprecated-bus",
			Backend:  "inproc://deprecated-backend",
			Frontend: "inproc://deprecated-frontend",
			Capture:  "inproc://deprecated-capture",
		}
		bus := NewBus(config)

		done := make(chan bool, 1)

		go bus.Run(done)
		time.Sleep(100 * time.Millisecond)
		done <- true
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Bus did not exit within timeout")
		}
	})
}

func TestBusIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("bus lifecycle", func(t *testing.T) {
		config := Config{
			Name:     "integration-bus",
			Backend:  "inproc://integration-backend",
			Frontend: "inproc://integration-frontend",
			Capture:  "inproc://integration-capture",
		}
		bus := NewBus(config)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)

		// Test that bus can be started and stopped cleanly
		go func() {
			err := bus.Start(ctx, &wg)
			require.NoError(t, err)
		}()

		// Let it run for a bit
		time.Sleep(500 * time.Millisecond)

		// Cancel and wait
		cancel()
		wg.Wait()
	})
}
