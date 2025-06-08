package bus

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewSource(t *testing.T) {
	endpoint := "inproc://test-source"
	envelope := "test-envelope"

	source := NewSource(endpoint, envelope)

	assert.NotNil(t, source)
	assert.Equal(t, endpoint, source.endpoint)
	assert.Equal(t, envelope, source.envelope)
	assert.False(t, source.running)
	assert.NotNil(t, source.queue)
}

func TestSourceDefaultFieldsBasic(t *testing.T) {
	source := NewSource("inproc://test", "envelope")
	fields := source.defaultFields(nil)
	assert.Equal(t, "inproc://test", fields["endpoint"])
	assert.Equal(t, "envelope", fields["envelope"])
	assert.NotContains(t, fields, "err")
}

func TestSourceDefaultFieldsWithError(t *testing.T) {
	source := NewSource("inproc://test", "envelope")
	testErr := assert.AnError
	fields := source.defaultFields(testErr)
	assert.Equal(t, "inproc://test", fields["endpoint"])
	assert.Equal(t, "envelope", fields["envelope"])
	assert.Equal(t, testErr, fields["err"])
}

func TestSourceRunning(t *testing.T) {
	source := NewSource("inproc://test", "envelope")

	assert.False(t, source.Running())

	source.running = true
	assert.True(t, source.Running())

	source.Stop()
	assert.False(t, source.Running())
}

func TestSourceStop(t *testing.T) {
	source := NewSource("inproc://test", "envelope")

	source.running = true
	assert.True(t, source.Running())

	source.Stop()
	assert.False(t, source.Running())

	assert.Panics(t, func() {
		source.QueueMessage([]byte("test"))
	})
}

func TestSourceQueueMessage(t *testing.T) {
	source := NewSource("inproc://test", "envelope")

	message := []byte("test message")

	// Should be able to queue message
	go func() {
		source.QueueMessage(message)
	}()

	// Should receive the message
	select {
	case received := <-source.queue:
		assert.Equal(t, message, received)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestSourceShutdown(t *testing.T) {
	source := NewSource("inproc://test", "envelope")
	source.running = true

	// Should queue shutdown command when running
	go source.Shutdown()

	select {
	case received := <-source.queue:
		assert.Equal(t, shutdownCommand, received)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for shutdown command")
	}
}

func TestSourceShutdownWhenNotRunning(t *testing.T) {
	source := NewSource("inproc://test", "envelope")
	source.running = false

	// Should not queue shutdown command when not running
	go source.Shutdown()

	select {
	case <-source.queue:
		t.Fatal("should not receive shutdown command when not running")
	case <-time.After(100 * time.Millisecond):
		// Expected - no message should be queued
	}
}

func TestSourceRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("context cancellation", func(t *testing.T) {
		source := NewSource("inproc://test-source-run", "test")

		ctx, cancel := context.WithCancel(context.Background())
		var wg sync.WaitGroup
		wg.Add(1)

		// Start source
		go source.Run(ctx, &wg)

		// Give it time to start
		time.Sleep(100 * time.Millisecond)
		assert.True(t, source.Running())

		// Cancel context
		cancel()

		// Wait for completion
		wg.Wait()
		assert.False(t, source.Running())
	})

	t.Run("shutdown command", func(t *testing.T) {
		source := NewSource("inproc://test-source-shutdown", "test")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)

		// Start source
		go source.Run(ctx, &wg)

		// Give it time to start
		time.Sleep(100 * time.Millisecond)
		assert.True(t, source.Running())

		// Send shutdown command
		source.Shutdown()

		// Wait for completion
		wg.Wait()
		assert.False(t, source.Running())
	})
}

func TestShutdownCommand(t *testing.T) {
	expected := []byte{0x0D, 0x0E, 0x0A, 0x0D}
	assert.Equal(t, expected, shutdownCommand)
}
