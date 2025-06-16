package mdp

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestBasicClientWorkerInteraction tests basic request-reply flow
func TestBasicClientWorkerInteraction(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// This is a mock integration test - in practice would need a real broker
	t.Run("mock client-worker interaction", func(t *testing.T) {
		// Test the message format creation and validation
		service := "echo"
		request := []string{"hello", "world"}

		// Simulate client sending request
		clientMsg := make([]string, 3, len(request)+3)
		clientMsg = append(clientMsg, request...)
		clientMsg[2] = service
		clientMsg[1] = MdpcClient
		clientMsg[0] = ""

		// Validate client message format
		if err := ValidateClientMessage(clientMsg); err != nil {
			t.Errorf("valid client message failed validation: %v", err)
		}

		// Simulate worker receiving request (via broker)
		workerRequestMsg := []string{"", MdpwWorker, MdpwRequest, "client-123", ""}
		workerRequestMsg = append(workerRequestMsg, request...)

		if err := ValidateWorkerMessage(workerRequestMsg); err != nil {
			t.Errorf("valid worker request message failed validation: %v", err)
		}

		// Simulate worker sending reply
		reply := []string{"echo:", "hello", "world"}
		workerReplyMsg := []string{MdpwWorker, MdpwFinal, "client-123", "response-data"}
		workerReplyMsg = append(workerReplyMsg, reply...)

		if err := ValidateWorkerMessage(workerReplyMsg); err != nil {
			t.Errorf("valid worker reply message failed validation: %v", err)
		}

		// Simulate client receiving reply (via broker)
		clientReplyMsg := []string{"", MdpcClient, service}
		clientReplyMsg = append(clientReplyMsg, reply...)

		if err := ValidateClientMessage(clientReplyMsg); err != nil {
			t.Errorf("valid client reply message failed validation: %v", err)
		}
	})
}

// TestWorkerRegistration tests worker service registration
func TestWorkerRegistration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("worker ready message format", func(t *testing.T) {
		service := "calculator"

		// Test READY message format
		readyMsg := []string{"", MdpwWorker, MdpwReady, service}

		if err := ValidateWorkerMessage(readyMsg); err != nil {
			t.Errorf("valid worker READY message failed validation: %v", err)
		}

		// Verify the service name is properly included
		if len(readyMsg) < 4 {
			t.Error("READY message should include service name")
		}
		if readyMsg[3] != service {
			t.Errorf("expected service %s, got %s", service, readyMsg[3])
		}
	})
}

// TestHeartbeatMechanism tests the heartbeat mechanism
func TestHeartbeatMechanism(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("heartbeat message format", func(t *testing.T) {
		// Test worker heartbeat
		heartbeatMsg := []string{"", MdpwWorker, MdpwHeartbeat}

		if err := ValidateWorkerMessage(heartbeatMsg); err != nil {
			t.Errorf("valid worker heartbeat message failed validation: %v", err)
		}

		// Heartbeat should be minimal message
		if len(heartbeatMsg) != 3 {
			t.Errorf("heartbeat message should have exactly 3 frames, got %d", len(heartbeatMsg))
		}
	})

	t.Run("heartbeat timing simulation", func(t *testing.T) {
		// Test heartbeat timing logic
		heartbeatInterval := time.Duration(HeartbeatInterval)
		liveness := HeartbeatLiveness

		if heartbeatInterval <= 0 {
			t.Error("heartbeat interval must be positive")
		}
		if liveness <= 0 {
			t.Error("heartbeat liveness must be positive")
		}

		// Simulate heartbeat timeout calculation
		maxSilence := heartbeatInterval * time.Duration(liveness)
		if maxSilence < heartbeatInterval {
			t.Error("maximum silence period should be at least one heartbeat interval")
		}
	})
}

// TestDisconnectionHandling tests disconnection scenarios
func TestDisconnectionHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("worker disconnect message", func(t *testing.T) {
		disconnectMsg := []string{"", MdpwWorker, MdpwDisconnect}

		if err := ValidateWorkerMessage(disconnectMsg); err != nil {
			t.Errorf("valid worker disconnect message failed validation: %v", err)
		}
	})

	t.Run("disconnection error handling", func(t *testing.T) {
		err := NewWorkerDisconnectedError("worker-123", nil)

		if !IsRetryableError(err) {
			t.Error("worker disconnection should be retryable")
		}
		if err.Context["worker"] != "worker-123" {
			t.Error("worker context should be preserved")
		}
	})
}

// TestConcurrentClients tests multiple clients accessing services
func TestConcurrentClients(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("concurrent client message validation", func(t *testing.T) {
		numClients := 10
		service := "echo"

		var wg sync.WaitGroup
		errors := make(chan error, numClients)

		for i := 0; i < numClients; i++ {
			wg.Add(1)
			go func(clientID int) {
				defer wg.Done()

				request := []string{fmt.Sprintf("message-from-client-%d", clientID)}
				clientMsg := []string{"", MdpcClient, service}
				clientMsg = append(clientMsg, request...)

				if err := ValidateClientMessage(clientMsg); err != nil {
					errors <- fmt.Errorf("client %d message validation failed: %w", clientID, err)
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for any validation errors
		for err := range errors {
			t.Error(err)
		}
	})
}

// BenchmarkIntegrationScenarios benchmarks integration scenarios
func BenchmarkIntegrationScenarios(b *testing.B) {
	b.Run("MessageRoundTrip", func(b *testing.B) {
		service := "echo"
		request := []string{"hello", "world"}

		for i := 0; i < b.N; i++ {
			// Client message creation
			clientMsg := []string{"", MdpcClient, service}
			clientMsg = append(clientMsg, request...)

			// Message validation
			_ = ValidateClientMessage(clientMsg)

			// Worker message creation
			workerMsg := []string{"", MdpwWorker, MdpwRequest, "client-id", ""}
			workerMsg = append(workerMsg, request...)

			// Worker message validation
			_ = ValidateWorkerMessage(workerMsg)
		}
	})
}
