package mdp

import "time"

// Majordomo Protocol Client and Worker API.
// Implements the MDP/Worker spec at http://rfc.zeromq.org/spec:7.

const (
	// MdpcClient is the version of MDP/Client we implement - upgraded to v0.2
	MdpcClient = "MDPC02"

	// MdpwWorker is the version of MDP/Worker we implement - upgraded to v0.2
	MdpwWorker = "MDPW02"

	// Backward compatibility constants for v0.1 (if needed)
	MdpcClientV1 = "MDPC01"
	MdpwWorkerV1 = "MDPW01"

	// HeartbeatLiveness is the number of heartbeat cycles a worker is deemed to
	// be dead after, initially set to 3, 5 is reasonable.
	HeartbeatLiveness = 3

	// HeartbeatInterval is the interval at which the broker sends heartbeats to
	// workers, initially set to 2.500 ms.
	HeartbeatInterval = 2500 * time.Millisecond

	// HeartbeatExpiry is the total duration for a worker until it is deemed to
	// be dead.
	HeartbeatExpiry = HeartbeatInterval * HeartbeatLiveness
)

// MDP v0.2 Client commands (single-byte identifiers)
const (
	MdpcRequest = string(rune(0x01)) // Client request
)

// MDP v0.2 Client reply types
const (
	MdpcPartial = string(rune(0x02)) // Partial response from broker to client
	MdpcFinal   = string(rune(0x03)) // Final response from broker to client
)

// MDP v0.2 Worker commands (single-byte identifiers)
const (
	MdpwReady      = string(rune(0x01)) // Worker ready
	MdpwRequest    = string(rune(0x02)) // Request from broker to worker
	MdpwPartial    = string(rune(0x03)) // Partial reply from worker to broker
	MdpwFinal      = string(rune(0x04)) // Final reply from worker to broker
	MdpwHeartbeat  = string(rune(0x05)) // Heartbeat
	MdpwDisconnect = string(rune(0x06)) // Worker disconnect
)

// Legacy MDP v0.1 commands (for backward compatibility if needed)
const (
	MdpwReadyV1 = string(rune(iota + 1))
	MdpwRequestV1
	MdpwReplyV1
	MdpwHeartbeatV1
	MdpwDisconnectV1
)

// MMI (Majordomo Management Interface) constants
const (
	MMINamespace = "mmi."

	// Standard MMI services defined in the MDP specification
	MMIService   = "mmi.service"   // Check if a service is available
	MMIWorkers   = "mmi.workers"   // List workers for a service
	MMIHeartbeat = "mmi.heartbeat" // Echo heartbeat
	MMIBroker    = "mmi.broker"    // Get broker information
)

// MMI response codes following HTTP-style status codes
const (
	MMICodeOK             = "200" // Service available/operation successful
	MMICodeNotFound       = "404" // Service not found/not available
	MMICodeNotImplemented = "501" // MMI service not implemented
	MMICodeError          = "500" // Internal error
)

var (
	// MdpsCommands are the v0.2 commands that are understood by the broker devices.
	MdpsCommands = map[string]string{
		MdpwReady:      "READY",
		MdpwRequest:    "REQUEST",
		MdpwPartial:    "PARTIAL",
		MdpwFinal:      "FINAL",
		MdpwHeartbeat:  "HEARTBEAT",
		MdpwDisconnect: "DISCONNECT",
	}

	// MdpsCommandsV1 are the legacy v0.1 commands (for backward compatibility)
	MdpsCommandsV1 = map[string]string{
		MdpwReadyV1:      "READY",
		MdpwRequestV1:    "REQUEST",
		MdpwReplyV1:      "REPLY",
		MdpwHeartbeatV1:  "HEARTBEAT",
		MdpwDisconnectV1: "DISCONNECT",
	}

	// MMIServices lists all supported MMI services
	MMIServices = map[string]string{
		MMIService:   "Check if a service is available",
		MMIWorkers:   "List workers for a service",
		MMIHeartbeat: "Echo heartbeat",
		MMIBroker:    "Get broker information",
	}
)
