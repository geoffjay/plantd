package mdp

import "time"

// Majordomo Protocol Client and Worker API.
// Implements the MDP/Worker spec at http://rfc.zeromq.org/spec:7.

const (
	// MdpcClient is the version of MDP/Client we implement.
	MdpcClient = "MDPC01"

	// MdpwWorker is the version of MDP/Worker we implement.
	MdpwWorker = "MDPW01"

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

// MDP/Server commands, as strings.
const (
	MdpwReady = string(rune(iota + 1))
	MdpwRequest
	MdpwReply
	MdpwHeartbeat
	MdpwDisconnect
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
	// MdpsCommands are the commands that are understood by the broker devices.
	MdpsCommands = map[string]string{
		MdpwReady:      "READY",
		MdpwRequest:    "REQUEST",
		MdpwReply:      "REPLY",
		MdpwHeartbeat:  "HEARTBEAT",
		MdpwDisconnect: "DISCONNECT",
	}

	// MMIServices lists all supported MMI services
	MMIServices = map[string]string{
		MMIService:   "Check if a service is available",
		MMIWorkers:   "List workers for a service",
		MMIHeartbeat: "Echo heartbeat",
		MMIBroker:    "Get broker information",
	}
)
