// Package services provides business logic for service integrations.
package services

// Status constants for services and components
const (
	StatusHealthy   = "healthy"
	StatusUnhealthy = "unhealthy"
	StatusUnknown   = "unknown"
	StatusDegraded  = "degraded"
	StatusStopped   = "stopped"
	StatusStable    = "stable"
	StatusError     = "error"
)

// Common error messages
const (
	ErrorUnknown = "unknown error"
)

// Test constants
const (
	TestBrokerEndpoint = "tcp://127.0.0.1:9797"
	TestTimeout        = "30s"
)
