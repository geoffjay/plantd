package mdp

import (
	"testing"
)

// TestMDPv1ProtocolCompliance tests all message formats according to MDP v0.1 specification
func TestMDPv1ProtocolCompliance(t *testing.T) {
	testCases := []struct {
		name      string
		message   []string
		validator func([]string) error
		expectErr bool
	}{
		// Client message validation tests
		{
			name:      "valid client message",
			message:   []string{"", MdpcClient, "echo", "hello", "world"},
			validator: ValidateClientMessage,
			expectErr: false,
		},
		{
			name:      "client message too short",
			message:   []string{"", MdpcClient, "echo"},
			validator: ValidateClientMessage,
			expectErr: true,
		},
		{
			name:      "client message invalid empty frame",
			message:   []string{"not-empty", MdpcClient, "echo", "hello"},
			validator: ValidateClientMessage,
			expectErr: true,
		},
		{
			name:      "client message invalid protocol",
			message:   []string{"", "BADPROTO", "echo", "hello"},
			validator: ValidateClientMessage,
			expectErr: true,
		},
		{
			name:      "client message empty service",
			message:   []string{"", MdpcClient, "", "hello"},
			validator: ValidateClientMessage,
			expectErr: true,
		},

		// Worker message validation tests
		{
			name:      "valid worker ready message",
			message:   []string{"", MdpwWorker, MdpwReady, "echo"},
			validator: ValidateWorkerMessage,
			expectErr: false,
		},
		{
			name:      "valid worker request message",
			message:   []string{"", MdpwWorker, MdpwRequest, "client-id", "", "hello"},
			validator: ValidateWorkerMessage,
			expectErr: false,
		},
		{
			name:      "valid worker reply message",
			message:   []string{"", MdpwWorker, MdpwReply, "client-id", "", "response"},
			validator: ValidateWorkerMessage,
			expectErr: false,
		},
		{
			name:      "valid worker heartbeat message",
			message:   []string{"", MdpwWorker, MdpwHeartbeat},
			validator: ValidateWorkerMessage,
			expectErr: false,
		},
		{
			name:      "valid worker disconnect message",
			message:   []string{"", MdpwWorker, MdpwDisconnect},
			validator: ValidateWorkerMessage,
			expectErr: false,
		},
		{
			name:      "worker message too short",
			message:   []string{"", MdpwWorker},
			validator: ValidateWorkerMessage,
			expectErr: true,
		},
		{
			name:      "worker message invalid empty frame",
			message:   []string{"not-empty", MdpwWorker, MdpwReady, "echo"},
			validator: ValidateWorkerMessage,
			expectErr: true,
		},
		{
			name:      "worker message invalid protocol",
			message:   []string{"", "BADPROTO", MdpwReady, "echo"},
			validator: ValidateWorkerMessage,
			expectErr: true,
		},
		{
			name:      "worker message invalid command",
			message:   []string{"", MdpwWorker, "BADCMD", "echo"},
			validator: ValidateWorkerMessage,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.validator(tc.message)
			if tc.expectErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tc.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestBrokerToClientMessageValidation tests broker-to-client message validation
func TestBrokerToClientMessageValidation(t *testing.T) {
	testCases := []struct {
		name      string
		message   []string
		expectErr bool
	}{
		{
			name:      "valid broker-to-client message",
			message:   []string{"client-id", "", MdpcClient, "echo", "response"},
			expectErr: false,
		},
		{
			name:      "broker-to-client message too short",
			message:   []string{"client-id", "", MdpcClient},
			expectErr: true,
		},
		{
			name:      "broker-to-client empty client address",
			message:   []string{"", "", MdpcClient, "echo", "response"},
			expectErr: true,
		},
		{
			name:      "broker-to-client invalid delimiter",
			message:   []string{"client-id", "not-empty", MdpcClient, "echo", "response"},
			expectErr: true,
		},
		{
			name:      "broker-to-client invalid protocol",
			message:   []string{"client-id", "", "BADPROTO", "echo", "response"},
			expectErr: true,
		},
		{
			name:      "broker-to-client empty service",
			message:   []string{"client-id", "", MdpcClient, "", "response"},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateBrokerToClientMessage(tc.message)
			if tc.expectErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tc.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestBrokerToWorkerMessageValidation tests broker-to-worker message validation
func TestBrokerToWorkerMessageValidation(t *testing.T) {
	testCases := []struct {
		name      string
		message   []string
		expectErr bool
	}{
		{
			name:      "valid broker-to-worker message",
			message:   []string{"worker-id", "", MdpwWorker, MdpwRequest, "client-id", "", "request"},
			expectErr: false,
		},
		{
			name:      "broker-to-worker message too short",
			message:   []string{"worker-id", ""},
			expectErr: true,
		},
		{
			name:      "broker-to-worker empty worker address",
			message:   []string{"", "", MdpwWorker, MdpwRequest},
			expectErr: true,
		},
		{
			name:      "broker-to-worker invalid delimiter",
			message:   []string{"worker-id", "not-empty", MdpwWorker, MdpwRequest},
			expectErr: true,
		},
		{
			name:      "broker-to-worker invalid protocol",
			message:   []string{"worker-id", "", "BADPROTO", MdpwRequest},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateBrokerToWorkerMessage(tc.message)
			if tc.expectErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tc.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestMessageFrameStructure tests the fundamental frame structure requirements
func TestMessageFrameStructure(t *testing.T) {
	t.Run("client message frame structure", func(t *testing.T) {
		// Test minimum frame count
		msg := []string{"", MdpcClient, "service", "data"}
		if err := ValidateClientMessage(msg); err != nil {
			t.Errorf("valid message failed validation: %v", err)
		}

		// Test with multiple data frames
		msg = []string{"", MdpcClient, "service", "data1", "data2", "data3"}
		if err := ValidateClientMessage(msg); err != nil {
			t.Errorf("valid multi-frame message failed validation: %v", err)
		}
	})

	t.Run("worker message frame structure", func(t *testing.T) {
		// Test minimum frame count for different commands
		commands := []string{MdpwReady, MdpwRequest, MdpwReply, MdpwHeartbeat, MdpwDisconnect}

		for _, cmd := range commands {
			msg := []string{"", MdpwWorker, cmd}
			if err := ValidateWorkerMessage(msg); err != nil {
				t.Errorf("valid %s message failed validation: %v", cmd, err)
			}
		}
	})
}

// TestProtocolConstants verifies protocol constant values
func TestProtocolConstants(t *testing.T) {
	// Test protocol identifiers
	if MdpcClient != "MDPC01" {
		t.Errorf("expected MDPC01, got %s", MdpcClient)
	}
	if MdpwWorker != "MDPW01" {
		t.Errorf("expected MDPW01, got %s", MdpwWorker)
	}

	// Test worker commands
	expectedCommands := map[string]string{
		"READY":      MdpwReady,
		"REQUEST":    MdpwRequest,
		"REPLY":      MdpwReply,
		"HEARTBEAT":  MdpwHeartbeat,
		"DISCONNECT": MdpwDisconnect,
	}

	for name, constant := range expectedCommands {
		if constant == "" {
			t.Errorf("command %s is empty", name)
		}
		if len(constant) != 1 {
			t.Errorf("command %s should be single byte, got %d bytes", name, len(constant))
		}
	}
}

// BenchmarkMessageValidation benchmarks message validation performance
func BenchmarkMessageValidation(b *testing.B) {
	clientMsg := []string{"", MdpcClient, "echo", "hello", "world"}
	workerMsg := []string{"", MdpwWorker, MdpwRequest, "client-id", "", "hello"}

	b.Run("ClientMessageValidation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = ValidateClientMessage(clientMsg)
		}
	})

	b.Run("WorkerMessageValidation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = ValidateWorkerMessage(workerMsg)
		}
	})
}
