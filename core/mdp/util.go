package mdp

import (
	"fmt"
)

func popWorker(workers []*brokerWorker) (worker *brokerWorker, workers2 []*brokerWorker) {
	worker = workers[0]
	workers2 = workers[1:]
	return
}

func delWorker(workers []*brokerWorker, worker *brokerWorker) []*brokerWorker {
	for i := 0; i < len(workers); i++ {
		if workers[i] == worker {
			workers = append(workers[:i], workers[i+1:]...)
			i--
		}
	}
	return workers
}

func stringArrayToByte2D(in []string) (out [][]byte) {
	for _, str := range in {
		out = append(out, []byte(str))
	}
	return
}

func byte2DToStringArray(in [][]byte) (out []string) {
	for _, bytes := range in {
		out = append(out, string(bytes))
	}
	return
}

// ValidateClientMessage validates that a message received by a client conforms to MDP v0.2 specification
func ValidateClientMessage(frames []string) error {
	if len(frames) < 3 {
		return fmt.Errorf("client message must have at least 3 frames, got %d", len(frames))
	}
	if frames[0] != MdpcClient {
		return fmt.Errorf("frame 0 must be %s, got %s", MdpcClient, frames[0])
	}

	// Frame 1 should be command (PARTIAL or FINAL)
	command := frames[1]
	switch command {
	case MdpcPartial, MdpcFinal:
		// Valid response commands
	default:
		return fmt.Errorf("frame 1 must be a valid client response command (%s or %s), got %s", MdpcPartial, MdpcFinal, command)
	}

	// Frame 2 is service name - allow any non-empty string
	if frames[2] == "" {
		return fmt.Errorf("frame 2 (service) cannot be empty")
	}
	return nil
}

// ValidateWorkerMessage validates that a message received by a worker conforms to MDP v0.2 specification
func ValidateWorkerMessage(frames []string) error {
	if len(frames) < 2 {
		return fmt.Errorf("worker message must have at least 2 frames, got %d", len(frames))
	}
	if frames[0] != MdpwWorker {
		return fmt.Errorf("frame 0 must be %s, got %s", MdpwWorker, frames[0])
	}

	// Frame 1 is command - validate it's a known command
	command := frames[1]
	switch command {
	case MdpwReady, MdpwRequest, MdpwPartial, MdpwFinal, MdpwHeartbeat, MdpwDisconnect:
		// Valid commands
	default:
		return fmt.Errorf("frame 1 must be a valid worker command, got %s", command)
	}
	return nil
}

// ValidateBrokerToClientMessage validates messages sent from broker to client (MDP v0.2)
func ValidateBrokerToClientMessage(frames []string) error {
	if len(frames) < 4 {
		return fmt.Errorf("broker-to-client message must have at least 4 frames, got %d", len(frames))
	}
	if frames[0] == "" {
		return fmt.Errorf("frame 0 (client address) cannot be empty")
	}
	if frames[1] != MdpcClient {
		return fmt.Errorf("frame 1 must be %s, got %s", MdpcClient, frames[1])
	}

	// Frame 2 should be response command
	command := frames[2]
	switch command {
	case MdpcPartial, MdpcFinal:
		// Valid response commands
	default:
		return fmt.Errorf("frame 2 must be a valid client response command (%s or %s), got %s", MdpcPartial, MdpcFinal, command)
	}

	if frames[3] == "" {
		return fmt.Errorf("frame 3 (service) cannot be empty")
	}
	return nil
}

// ValidateBrokerToWorkerMessage validates messages sent from broker to worker (MDP v0.2)
func ValidateBrokerToWorkerMessage(frames []string) error {
	if len(frames) < 3 {
		return fmt.Errorf("broker-to-worker message must have at least 3 frames, got %d", len(frames))
	}
	if frames[0] == "" {
		return fmt.Errorf("frame 0 (worker address) cannot be empty")
	}
	if frames[1] != MdpwWorker {
		return fmt.Errorf("frame 1 must be %s, got %s", MdpwWorker, frames[1])
	}

	// Frame 2 is command - validate it's a known command
	command := frames[2]
	switch command {
	case MdpwReady, MdpwRequest, MdpwPartial, MdpwFinal, MdpwHeartbeat, MdpwDisconnect:
		// Valid commands
	default:
		return fmt.Errorf("frame 2 must be a valid worker command, got %s", command)
	}
	return nil
}

// ValidateClientRequestMessage validates a client request message for MDP v0.2
func ValidateClientRequestMessage(frames []string) error {
	if len(frames) < 3 {
		return fmt.Errorf("client request message must have at least 3 frames, got %d", len(frames))
	}
	if frames[0] != MdpcClient {
		return fmt.Errorf("frame 0 must be %s, got %s", MdpcClient, frames[0])
	}
	if frames[1] != MdpcRequest {
		return fmt.Errorf("frame 1 must be %s, got %s", MdpcRequest, frames[1])
	}
	if frames[2] == "" {
		return fmt.Errorf("frame 2 (service) cannot be empty")
	}
	return nil
}

// ValidateWorkerReplyMessage validates a worker reply message for MDP v0.2
func ValidateWorkerReplyMessage(frames []string) error {
	if len(frames) < 4 {
		return fmt.Errorf("worker reply message must have at least 4 frames, got %d", len(frames))
	}
	if frames[0] == "" {
		return fmt.Errorf("frame 0 (worker address) cannot be empty")
	}
	if frames[1] != MdpwWorker {
		return fmt.Errorf("frame 1 must be %s, got %s", MdpwWorker, frames[1])
	}

	// Frame 2 should be PARTIAL or FINAL
	command := frames[2]
	switch command {
	case MdpwPartial, MdpwFinal:
		// Valid reply commands
	default:
		return fmt.Errorf("frame 2 must be a valid worker reply command (%s or %s), got %s", MdpwPartial, MdpwFinal, command)
	}

	if frames[3] == "" {
		return fmt.Errorf("frame 3 (client address) cannot be empty")
	}
	return nil
}
