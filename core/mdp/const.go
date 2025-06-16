package mdp

import "time"

// Majordomo Protocol Client and Worker API.
// Implements the MDP/Worker spec at http://rfc.zeromq.org/spec:7.

const (
	// MdpcClient is the version of MDP/Client we implement - upgraded to v0.2
	MdpcClient = "MDPC02"

	// MdpwWorker is the version of MDP/Worker we implement - upgraded to v0.2
	MdpwWorker = "MDPW02"

	// MdpcClientV1 and MdpwWorkerV1 are the legacy v0.1 commands (for backward compatibility)
	MdpcClientV1 = "MDPC01"
	// MdpwWorkerV1 is the version of MDP/Worker v0.1 for backward compatibility
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

// MDP v0.2 Client commands (human-readable string identifiers)
const (
	MdpcRequest = "REQUEST" // Client request
)

// MDP v0.2 Client reply types
const (
	MdpcPartial = "PARTIAL" // Partial response from broker to client
	MdpcFinal   = "FINAL"   // Final response from broker to client
)

// MDP v0.2 Worker commands (human-readable string identifiers)
const (
	MdpwReady      = "READY"      // Worker ready
	MdpwRequest    = "REQUEST"    // Request from broker to worker
	MdpwPartial    = "PARTIAL"    // Partial reply from worker to broker
	MdpwFinal      = "FINAL"      // Final reply from worker to broker
	MdpwHeartbeat  = "HEARTBEAT"  // Heartbeat
	MdpwDisconnect = "DISCONNECT" // Worker disconnect
)

// Legacy MDP v0.1 commands (single-byte for backward compatibility)
const (
	MdpwReadyV1      = string(rune(0x01)) // Legacy READY
	MdpwRequestV1    = string(rune(0x02)) // Legacy REQUEST
	MdpwReplyV1      = string(rune(0x03)) // Legacy REPLY
	MdpwHeartbeatV1  = string(rune(0x04)) // Legacy HEARTBEAT
	MdpwDisconnectV1 = string(rune(0x05)) // Legacy DISCONNECT
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

// Node and request status constants
const (
	// StatusActive indicates a node or request is active
	StatusActive = "active"
	// StatusInactive indicates a node or request is inactive
	StatusInactive = "inactive"
	// StatusFailed indicates a node or request has failed
	StatusFailed = "failed"
	// StatusPending indicates a request is pending
	StatusPending = "pending"
	// StatusProcessing indicates a request is being processed
	StatusProcessing = "processing"
)

// Configuration string constants
const (
	// BoolTrue represents the string "true" used in configuration
	BoolTrue = "true"
	// ServiceEcho represents the echo service name
	ServiceEcho = "echo"
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
