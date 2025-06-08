package bus

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock callback for testing.
type mockSinkCallback struct {
	mock.Mock
}

func (m *mockSinkCallback) Handle(data []byte) error {
	args := m.Called(data)
	return args.Error(0)
}

func TestNewSink(t *testing.T) {
	endpoint := "inproc://test-sink"
	filter := "test-filter"

	sink := NewSink(endpoint, filter)

	assert.NotNil(t, sink)
	assert.Equal(t, endpoint, sink.endpoint)
	assert.Equal(t, filter, sink.filter)
	assert.False(t, sink.running)
	assert.Nil(t, sink.handler)
}

func TestSinkDefaultFieldsBasic(t *testing.T) {
	sink := NewSink("inproc://test", "filter")
	fields := sink.defaultFields(nil)
	assert.Equal(t, "inproc://test", fields["endpoint"])
	assert.Equal(t, "filter", fields["filter"])
	assert.NotContains(t, fields, "err")
}

func TestSinkDefaultFieldsWithError(t *testing.T) {
	sink := NewSink("inproc://test", "filter")
	testErr := assert.AnError
	fields := sink.defaultFields(testErr)
	assert.Equal(t, "inproc://test", fields["endpoint"])
	assert.Equal(t, "filter", fields["filter"])
	assert.Equal(t, testErr, fields["err"])
}

func TestSinkSetHandler(t *testing.T) {
	sink := NewSink("inproc://test", "filter")
	callback := &mockSinkCallback{}
	handler := &SinkHandler{Callback: callback}

	// Initially no handler
	assert.Nil(t, sink.handler)

	// Set handler
	sink.SetHandler(handler)
	assert.NotNil(t, sink.handler)
	assert.Equal(t, handler, sink.handler)
	assert.Equal(t, callback, sink.handler.Callback)
}

func TestSinkRunning(t *testing.T) {
	sink := NewSink("inproc://test", "filter")

	assert.False(t, sink.Running())

	// Simulate running state
	sink.running = true
	assert.True(t, sink.Running())

	sink.Stop()
	assert.False(t, sink.Running())
}

func TestSinkStop(t *testing.T) {
	sink := NewSink("inproc://test", "filter")

	sink.running = true
	assert.True(t, sink.Running())

	// Stop should set running to false
	sink.Stop()
	assert.False(t, sink.Running())
}

func TestSinkHandler(t *testing.T) {
	callback := &mockSinkCallback{}
	handler := &SinkHandler{Callback: callback}

	assert.NotNil(t, handler)
	assert.Equal(t, callback, handler.Callback)
}

func TestSinkRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("context cancellation", func(t *testing.T) {
		sink := NewSink("inproc://test-sink-run", "")
		callback := &mockSinkCallback{}
		handler := &SinkHandler{Callback: callback}
		sink.SetHandler(handler)

		ctx, cancel := context.WithCancel(context.Background())
		var wg sync.WaitGroup
		wg.Add(1)

		// Start sink
		go sink.Run(ctx, &wg)

		// Give it time to start
		time.Sleep(100 * time.Millisecond)

		// Cancel context
		cancel()

		// Wait for completion
		wg.Wait()

		// Check final state
		assert.False(t, sink.Running())
	})

	t.Run("stop method", func(t *testing.T) {
		sink := NewSink("inproc://test-sink-stop", "")
		callback := &mockSinkCallback{}
		handler := &SinkHandler{Callback: callback}
		sink.SetHandler(handler)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var wg sync.WaitGroup
		wg.Add(1)

		// Start sink
		go sink.Run(ctx, &wg)

		// Give it time to start
		time.Sleep(100 * time.Millisecond)
		assert.True(t, sink.Running())

		// Stop sink
		sink.Stop()

		// Cancel context to clean up
		cancel()

		// Wait for completion
		wg.Wait()
		assert.False(t, sink.Running())
	})
}

func TestSinkCallbackInterface(t *testing.T) {
	// Test that our mock implements the interface correctly
	var callback SinkCallback = &mockSinkCallback{}
	assert.NotNil(t, callback)

	// Test callback functionality
	mockCallback := &mockSinkCallback{}
	testData := []byte("test data")
	mockCallback.On("Handle", testData).Return(nil)

	err := mockCallback.Handle(testData)
	assert.NoError(t, err)
	mockCallback.AssertExpectations(t)
}

func TestSinkCallbackError(t *testing.T) {
	mockCallback := &mockSinkCallback{}
	testData := []byte("test data")
	expectedErr := assert.AnError

	mockCallback.On("Handle", testData).Return(expectedErr)

	err := mockCallback.Handle(testData)
	assert.Equal(t, expectedErr, err)
	mockCallback.AssertExpectations(t)
}
