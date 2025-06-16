package mdp

import (
	"testing"
)

// TestMDPv2ProtocolCompliance tests all message formats according to MDP v0.2 specification
func TestMDPv2ProtocolCompliance(t *testing.T) {
	testCases := []struct {
		name      string
		message   []string
		validator func([]string) error
		expectErr bool
	}{
		// Client response message validation tests (what clients receive)
		{
			name:      "valid client partial response",
			message:   []string{MdpcClient, MdpcPartial, "echo", "hello", "world"},
			validator: ValidateClientMessage,
			expectErr: false,
		},
		{
			name:      "valid client final response",
			message:   []string{MdpcClient, MdpcFinal, "echo", "hello", "world"},
			validator: ValidateClientMessage,
			expectErr: false,
		},
		{
			name:      "client message too short",
			message:   []string{MdpcClient, MdpcFinal},
			validator: ValidateClientMessage,
			expectErr: true,
		},
		{
			name:      "client message invalid protocol",
			message:   []string{"BADPROTO", MdpcFinal, "echo", "hello"},
			validator: ValidateClientMessage,
			expectErr: true,
		},
		{
			name:      "client message invalid command",
			message:   []string{MdpcClient, "BADCMD", "echo", "hello"},
			validator: ValidateClientMessage,
			expectErr: true,
		},
		{
			name:      "client message empty service",
			message:   []string{MdpcClient, MdpcFinal, "", "hello"},
			validator: ValidateClientMessage,
			expectErr: true,
		},

		// Worker message validation tests (what workers receive)
		{
			name:      "valid worker ready message",
			message:   []string{MdpwWorker, MdpwReady, "echo"},
			validator: ValidateWorkerMessage,
			expectErr: false,
		},
		{
			name:      "valid worker request message",
			message:   []string{MdpwWorker, MdpwRequest, "client-id", "hello"},
			validator: ValidateWorkerMessage,
			expectErr: false,
		},
		{
			name:      "valid worker heartbeat message",
			message:   []string{MdpwWorker, MdpwHeartbeat},
			validator: ValidateWorkerMessage,
			expectErr: false,
		},
		{
			name:      "valid worker disconnect message",
			message:   []string{MdpwWorker, MdpwDisconnect},
			validator: ValidateWorkerMessage,
			expectErr: false,
		},
		{
			name:      "worker message too short",
			message:   []string{MdpwWorker},
			validator: ValidateWorkerMessage,
			expectErr: true,
		},
		{
			name:      "worker message invalid protocol",
			message:   []string{"BADPROTO", MdpwReady, "echo"},
			validator: ValidateWorkerMessage,
			expectErr: true,
		},
		{
			name:      "worker message invalid command",
			message:   []string{MdpwWorker, "BADCMD", "echo"},
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

// TestClientRequestMessageValidation tests client request message validation
func TestClientRequestMessageValidation(t *testing.T) {
	testCases := []struct {
		name      string
		message   []string
		expectErr bool
	}{
		{
			name:      "valid client request message",
			message:   []string{MdpcClient, MdpcRequest, "echo", "hello", "world"},
			expectErr: false,
		},
		{
			name:      "client request too short",
			message:   []string{MdpcClient, MdpcRequest},
			expectErr: true,
		},
		{
			name:      "client request invalid protocol",
			message:   []string{"BADPROTO", MdpcRequest, "echo", "hello"},
			expectErr: true,
		},
		{
			name:      "client request invalid command",
			message:   []string{MdpcClient, "BADCMD", "echo", "hello"},
			expectErr: true,
		},
		{
			name:      "client request empty service",
			message:   []string{MdpcClient, MdpcRequest, "", "hello"},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateClientRequestMessage(tc.message)
			if tc.expectErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tc.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestWorkerReplyMessageValidation tests worker reply message validation
func TestWorkerReplyMessageValidation(t *testing.T) {
	testCases := []struct {
		name      string
		message   []string
		expectErr bool
	}{
		{
			name:      "valid worker partial reply",
			message:   []string{"worker-id", MdpwWorker, MdpwPartial, "client-id", "partial-data"},
			expectErr: false,
		},
		{
			name:      "valid worker final reply",
			message:   []string{"worker-id", MdpwWorker, MdpwFinal, "client-id", "final-data"},
			expectErr: false,
		},
		{
			name:      "worker reply too short",
			message:   []string{"worker-id", MdpwWorker, MdpwFinal},
			expectErr: true,
		},
		{
			name:      "worker reply empty worker address",
			message:   []string{"", MdpwWorker, MdpwFinal, "client-id", "data"},
			expectErr: true,
		},
		{
			name:      "worker reply invalid protocol",
			message:   []string{"worker-id", "BADPROTO", MdpwFinal, "client-id", "data"},
			expectErr: true,
		},
		{
			name:      "worker reply invalid command",
			message:   []string{"worker-id", MdpwWorker, "BADCMD", "client-id", "data"},
			expectErr: true,
		},
		{
			name:      "worker reply empty client address",
			message:   []string{"worker-id", MdpwWorker, MdpwFinal, "", "data"},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateWorkerReplyMessage(tc.message)
			if tc.expectErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tc.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestBrokerToClientMessageValidation tests broker-to-client message validation (MDP v0.2)
func TestBrokerToClientMessageValidation(t *testing.T) {
	testCases := []struct {
		name      string
		message   []string
		expectErr bool
	}{
		{
			name:      "valid broker-to-client partial response",
			message:   []string{"client-id", MdpcClient, MdpcPartial, "echo", "partial-response"},
			expectErr: false,
		},
		{
			name:      "valid broker-to-client final response",
			message:   []string{"client-id", MdpcClient, MdpcFinal, "echo", "final-response"},
			expectErr: false,
		},
		{
			name:      "broker-to-client message too short",
			message:   []string{"client-id", MdpcClient, MdpcFinal},
			expectErr: true,
		},
		{
			name:      "broker-to-client empty client address",
			message:   []string{"", MdpcClient, MdpcFinal, "echo", "response"},
			expectErr: true,
		},
		{
			name:      "broker-to-client invalid protocol",
			message:   []string{"client-id", "BADPROTO", MdpcFinal, "echo", "response"},
			expectErr: true,
		},
		{
			name:      "broker-to-client invalid command",
			message:   []string{"client-id", MdpcClient, "BADCMD", "echo", "response"},
			expectErr: true,
		},
		{
			name:      "broker-to-client empty service",
			message:   []string{"client-id", MdpcClient, MdpcFinal, "", "response"},
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

// TestBrokerToWorkerMessageValidation tests broker-to-worker message validation (MDP v0.2)
func TestBrokerToWorkerMessageValidation(t *testing.T) {
	testCases := []struct {
		name      string
		message   []string
		expectErr bool
	}{
		{
			name:      "valid broker-to-worker request",
			message:   []string{"worker-id", MdpwWorker, MdpwRequest, "client-id", "request-data"},
			expectErr: false,
		},
		{
			name:      "valid broker-to-worker heartbeat",
			message:   []string{"worker-id", MdpwWorker, MdpwHeartbeat},
			expectErr: false,
		},
		{
			name:      "valid broker-to-worker disconnect",
			message:   []string{"worker-id", MdpwWorker, MdpwDisconnect},
			expectErr: false,
		},
		{
			name:      "broker-to-worker message too short",
			message:   []string{"worker-id", MdpwWorker},
			expectErr: true,
		},
		{
			name:      "broker-to-worker empty worker address",
			message:   []string{"", MdpwWorker, MdpwRequest, "client-id"},
			expectErr: true,
		},
		{
			name:      "broker-to-worker invalid protocol",
			message:   []string{"worker-id", "BADPROTO", MdpwRequest, "client-id"},
			expectErr: true,
		},
		{
			name:      "broker-to-worker invalid command",
			message:   []string{"worker-id", MdpwWorker, "BADCMD"},
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

// TestMessageFrameStructure tests the overall structure requirements for MDP v0.2
func TestMessageFrameStructure(t *testing.T) {
	tests := []struct {
		name        string
		description string
		frames      []string
		valid       bool
	}{
		{
			name:        "client_request_minimal",
			description: "Minimal valid client request",
			frames:      []string{MdpcClient, MdpcRequest, "test-service"},
			valid:       true,
		},
		{
			name:        "client_request_with_data",
			description: "Client request with data payload",
			frames:      []string{MdpcClient, MdpcRequest, "test-service", "data1", "data2"},
			valid:       true,
		},
		{
			name:        "worker_ready_minimal",
			description: "Minimal worker ready message",
			frames:      []string{MdpwWorker, MdpwReady},
			valid:       true,
		},
		{
			name:        "worker_reply_partial",
			description: "Worker partial reply",
			frames:      []string{"worker-123", MdpwWorker, MdpwPartial, "client-456", "partial-data"},
			valid:       true,
		},
		{
			name:        "worker_reply_final",
			description: "Worker final reply",
			frames:      []string{"worker-123", MdpwWorker, MdpwFinal, "client-456", "final-data"},
			valid:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing: %s", tt.description)
			t.Logf("Frames: %v", tt.frames)

			// Test the appropriate validator based on message type
			var err error
			if len(tt.frames) > 0 {
				switch tt.frames[0] {
				case MdpcClient:
					if len(tt.frames) > 1 && tt.frames[1] == MdpcRequest {
						err = ValidateClientRequestMessage(tt.frames)
					} else {
						err = ValidateClientMessage(tt.frames)
					}
				case MdpwWorker:
					err = ValidateWorkerMessage(tt.frames)
				default:
					// Check if it's a broker message (starts with address)
					if len(tt.frames) > 1 && tt.frames[1] == MdpcClient {
						err = ValidateBrokerToClientMessage(tt.frames)
					} else if len(tt.frames) > 1 && tt.frames[1] == MdpwWorker {
						err = ValidateBrokerToWorkerMessage(tt.frames)
					} else if len(tt.frames) > 2 && tt.frames[1] == MdpwWorker {
						err = ValidateWorkerReplyMessage(tt.frames)
					}
				}
			}

			if tt.valid && err != nil {
				t.Errorf("Expected valid message but got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Errorf("Expected invalid message but got no error")
			}
		})
	}
}

// TestProtocolConstants verifies the MDP v0.2 protocol constants
func TestProtocolConstants(t *testing.T) {
	// Test protocol version strings
	if MdpcClient != "MDPC02" {
		t.Errorf("Expected MdpcClient to be 'MDPC02', got '%s'", MdpcClient)
	}
	if MdpwWorker != "MDPW02" {
		t.Errorf("Expected MdpwWorker to be 'MDPW02', got '%s'", MdpwWorker)
	}

	// Test command constants are single bytes
	commands := []string{MdpcRequest, MdpcPartial, MdpcFinal, MdpwReady, MdpwRequest, MdpwPartial, MdpwFinal, MdpwHeartbeat, MdpwDisconnect}
	for i, cmd := range commands {
		if len(cmd) != 1 {
			t.Errorf("Command %d should be single byte, got length %d: %q", i, len(cmd), cmd)
		}
	}

	// Test MMI constants
	if MMINamespace != "mmi." {
		t.Errorf("Expected MMI namespace to be 'mmi.', got '%s'", MMINamespace)
	}
}

// BenchmarkMessageValidation benchmarks the message validation functions
func BenchmarkMessageValidation(b *testing.B) {
	clientMsg := []string{MdpcClient, MdpcFinal, "echo", "hello", "world"}
	workerMsg := []string{MdpwWorker, MdpwReady, "echo"}
	requestMsg := []string{MdpcClient, MdpcRequest, "echo", "data"}

	b.Run("ClientMessage", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = ValidateClientMessage(clientMsg)
		}
	})

	b.Run("WorkerMessage", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = ValidateWorkerMessage(workerMsg)
		}
	})

	b.Run("ClientRequestMessage", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = ValidateClientRequestMessage(requestMsg)
		}
	})
}
