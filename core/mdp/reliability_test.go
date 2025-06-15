package mdp

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

// TestErrorClassification tests error classification functions
func TestErrorClassification(t *testing.T) {
	testCases := []struct {
		name            string
		err             error
		expectRetryable bool
		expectPermanent bool
	}{
		{
			name:            "timeout error is retryable",
			err:             NewTimeoutError("operation timed out", nil),
			expectRetryable: true,
			expectPermanent: false,
		},
		{
			name:            "broker unavailable is retryable",
			err:             NewBrokerUnavailableError("broker down", nil),
			expectRetryable: true,
			expectPermanent: false,
		},
		{
			name:            "connection failed is retryable",
			err:             NewConnectionFailedError("tcp://localhost:5555", nil),
			expectRetryable: true,
			expectPermanent: false,
		},
		{
			name:            "protocol violation is permanent",
			err:             NewProtocolViolationError("invalid frame count", nil),
			expectRetryable: false,
			expectPermanent: true,
		},
		{
			name:            "invalid message is permanent",
			err:             NewInvalidMessageError("malformed message", nil),
			expectRetryable: false,
			expectPermanent: true,
		},
		{
			name:            "standard timeout error is retryable",
			err:             ErrTimeout,
			expectRetryable: true,
			expectPermanent: false,
		},
		{
			name:            "standard protocol violation is permanent",
			err:             ErrProtocolViolation,
			expectRetryable: false,
			expectPermanent: true,
		},
		{
			name:            "nil error is neither",
			err:             nil,
			expectRetryable: false,
			expectPermanent: false,
		},
		{
			name:            "unknown error is neither",
			err:             errors.New("unknown error"),
			expectRetryable: false,
			expectPermanent: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isRetryable := IsRetryableError(tc.err)
			isPermanent := IsPermanentError(tc.err)

			if isRetryable != tc.expectRetryable {
				t.Errorf("expected retryable=%v, got %v", tc.expectRetryable, isRetryable)
			}
			if isPermanent != tc.expectPermanent {
				t.Errorf("expected permanent=%v, got %v", tc.expectPermanent, isPermanent)
			}

			// Errors should not be both retryable and permanent
			if isRetryable && isPermanent {
				t.Error("error cannot be both retryable and permanent")
			}
		})
	}
}

// TestMDPErrorStructure tests the MDPError struct functionality
func TestMDPErrorStructure(t *testing.T) {
	t.Run("basic error creation", func(t *testing.T) {
		err := NewMDPError("TEST001", "test error", nil)
		if err.Code != "TEST001" {
			t.Errorf("expected code TEST001, got %s", err.Code)
		}
		if err.Message != "test error" {
			t.Errorf("expected message 'test error', got %s", err.Message)
		}
		if err.Cause != nil {
			t.Errorf("expected nil cause, got %v", err.Cause)
		}
	})

	t.Run("error with cause", func(t *testing.T) {
		cause := errors.New("underlying error")
		err := NewMDPError("TEST002", "wrapper error", cause)

		if err.Cause != cause {
			t.Errorf("expected cause to be set")
		}
		if !errors.Is(err, cause) {
			t.Errorf("error should wrap the cause")
		}
		if errors.Unwrap(err) != cause {
			t.Errorf("unwrap should return the cause")
		}
	})

	t.Run("error context", func(t *testing.T) {
		err := NewMDPError("TEST003", "context test", nil)
		err = err.WithContext("service", "echo").WithContext("worker", "worker-1")

		if err.Context["service"] != "echo" {
			t.Errorf("expected service=echo, got %v", err.Context["service"])
		}
		if err.Context["worker"] != "worker-1" {
			t.Errorf("expected worker=worker-1, got %v", err.Context["worker"])
		}
	})

	t.Run("error string formatting", func(t *testing.T) {
		// Without cause
		err1 := NewMDPError("TEST004", "simple error", nil)
		expected1 := "MDP TEST004: simple error"
		if err1.Error() != expected1 {
			t.Errorf("expected %s, got %s", expected1, err1.Error())
		}

		// With cause
		cause := errors.New("root cause")
		err2 := NewMDPError("TEST005", "wrapped error", cause)
		expected2 := "MDP TEST005: wrapped error: root cause"
		if err2.Error() != expected2 {
			t.Errorf("expected %s, got %s", expected2, err2.Error())
		}
	})
}

// TestErrorFactoryFunctions tests the convenience error factory functions
func TestErrorFactoryFunctions(t *testing.T) {
	t.Run("service not found error", func(t *testing.T) {
		err := NewServiceNotFoundError("nonexistent", nil)
		if err.Code != ErrCodeServiceNotFound {
			t.Errorf("expected code %s, got %s", ErrCodeServiceNotFound, err.Code)
		}
		if err.Context["service"] != "nonexistent" {
			t.Errorf("expected service context to be set")
		}
	})

	t.Run("worker disconnected error", func(t *testing.T) {
		err := NewWorkerDisconnectedError("worker-123", nil)
		if err.Code != ErrCodeWorkerDisconnected {
			t.Errorf("expected code %s, got %s", ErrCodeWorkerDisconnected, err.Code)
		}
		if err.Context["worker"] != "worker-123" {
			t.Errorf("expected worker context to be set")
		}
	})

	t.Run("connection failed error", func(t *testing.T) {
		endpoint := "tcp://localhost:5555"
		err := NewConnectionFailedError(endpoint, nil)
		if err.Code != ErrCodeConnectionFailed {
			t.Errorf("expected code %s, got %s", ErrCodeConnectionFailed, err.Code)
		}
		if err.Context["endpoint"] != endpoint {
			t.Errorf("expected endpoint context to be set")
		}
	})
}

// TestErrorComparison tests error comparison using errors.Is
func TestErrorComparison(t *testing.T) {
	originalErr := NewTimeoutError("timeout occurred", nil)
	wrappedErr := NewProtocolViolationError("protocol error", originalErr)

	t.Run("same error type comparison", func(t *testing.T) {
		err1 := NewTimeoutError("timeout 1", nil)
		err2 := NewTimeoutError("timeout 2", nil)

		if !errors.Is(err1, err2) {
			t.Error("errors with same code should be considered equal")
		}
	})

	t.Run("different error type comparison", func(t *testing.T) {
		err1 := NewTimeoutError("timeout", nil)
		err2 := NewProtocolViolationError("protocol", nil)

		if errors.Is(err1, err2) {
			t.Error("errors with different codes should not be equal")
		}
	})

	t.Run("wrapped error comparison", func(t *testing.T) {
		if !errors.Is(wrappedErr, originalErr) {
			t.Error("wrapped error should match original error")
		}
	})

	t.Run("standard error comparison", func(t *testing.T) {
		mdpTimeout := NewTimeoutError("timeout", ErrTimeout)
		if !errors.Is(mdpTimeout, ErrTimeout) {
			t.Error("MDP error should match standard error")
		}
	})
}

// TestHeartbeatReliability tests heartbeat mechanism reliability
func TestHeartbeatReliability(t *testing.T) {
	// This would be an integration test in practice
	t.Run("heartbeat constants", func(t *testing.T) {
		if HeartbeatInterval <= 0 {
			t.Error("heartbeat interval must be positive")
		}
		if HeartbeatLiveness <= 0 {
			t.Error("heartbeat liveness must be positive")
		}
	})

	t.Run("heartbeat timing calculations", func(t *testing.T) {
		// Test that heartbeat calculations are reasonable
		interval := time.Duration(HeartbeatInterval)
		if interval < 100*time.Millisecond {
			t.Error("heartbeat interval seems too small")
		}
		if interval > 30*time.Second {
			t.Error("heartbeat interval seems too large")
		}
	})
}

// TestTimeoutHandling tests timeout handling reliability
func TestTimeoutHandling(t *testing.T) {
	t.Run("timeout error creation", func(t *testing.T) {
		timeout := 5 * time.Second
		err := NewTimeoutError(fmt.Sprintf("operation timed out after %v", timeout), nil)

		if !IsRetryableError(err) {
			t.Error("timeout error should be retryable")
		}
		if IsPermanentError(err) {
			t.Error("timeout error should not be permanent")
		}
	})

	t.Run("timeout error context", func(t *testing.T) {
		err := NewTimeoutError("timeout", nil).
			WithContext("operation", "request").
			WithContext("timeout", "5s")

		if err.Context["operation"] != "request" {
			t.Error("timeout context not preserved")
		}
		if err.Context["timeout"] != "5s" {
			t.Error("timeout value context not preserved")
		}
	})
}

// TestConnectionReliability tests connection reliability features
func TestConnectionReliability(t *testing.T) {
	t.Run("connection error types", func(t *testing.T) {
		connectionErrors := []error{
			NewConnectionFailedError("tcp://broker:5555", nil),
			ErrConnectionFailed,
			ErrBrokerUnavailable,
			ErrSocketError,
		}

		for _, err := range connectionErrors {
			if !IsRetryableError(err) {
				t.Errorf("connection error should be retryable: %v", err)
			}
			if IsPermanentError(err) {
				t.Errorf("connection error should not be permanent: %v", err)
			}
		}
	})

	t.Run("connection context preservation", func(t *testing.T) {
		endpoint := "tcp://localhost:5555"
		err := NewConnectionFailedError(endpoint, nil)

		if err.Context["endpoint"] != endpoint {
			t.Error("connection endpoint context not preserved")
		}
	})
}

// TestMessageReliability tests message handling reliability
func TestMessageReliability(t *testing.T) {
	t.Run("message validation errors are permanent", func(t *testing.T) {
		validationErrors := []error{
			NewInvalidMessageError("too few frames", nil),
			NewProtocolViolationError("wrong protocol ID", nil),
			ErrInvalidMessage,
			ErrProtocolViolation,
		}

		for _, err := range validationErrors {
			if IsRetryableError(err) {
				t.Errorf("validation error should not be retryable: %v", err)
			}
			if !IsPermanentError(err) {
				t.Errorf("validation error should be permanent: %v", err)
			}
		}
	})

	t.Run("large message handling", func(t *testing.T) {
		err := NewMDPError(ErrCodeMessageTooLarge, "message exceeds size limit", ErrMessageTooLarge)

		// Large message errors could be considered permanent or temporary
		// depending on implementation - this tests that the error is properly structured
		if err.Code != ErrCodeMessageTooLarge {
			t.Error("message too large error code incorrect")
		}
	})
}

// BenchmarkErrorHandling benchmarks error handling performance
func BenchmarkErrorHandling(b *testing.B) {
	cause := errors.New("underlying error")

	b.Run("ErrorCreation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = NewMDPError("TEST", "benchmark error", cause)
		}
	})

	b.Run("ErrorClassification", func(b *testing.B) {
		err := NewTimeoutError("timeout", nil)
		for i := 0; i < b.N; i++ {
			_ = IsRetryableError(err)
			_ = IsPermanentError(err)
		}
	})

	b.Run("ErrorComparison", func(b *testing.B) {
		err1 := NewTimeoutError("timeout 1", nil)
		err2 := NewTimeoutError("timeout 2", nil)
		for i := 0; i < b.N; i++ {
			_ = errors.Is(err1, err2)
		}
	})

	b.Run("ContextAddition", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := NewMDPError("TEST", "test", nil)
			_ = err.WithContext("key1", "value1").WithContext("key2", "value2")
		}
	})
}
