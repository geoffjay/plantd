// Package models provides data transfer objects for the App Service.
package models

// APIResponse represents a standard API response format.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

// ServiceStatus represents the status of a plantd service.
type ServiceStatus struct {
	Name        string  `json:"name"`
	Status      string  `json:"status"` // "healthy", "unhealthy", "unknown"
	Workers     int     `json:"workers"`
	LastSeen    string  `json:"last_seen"`
	Version     string  `json:"version"`
	Endpoint    string  `json:"endpoint"`
	Heartbeat   int     `json:"heartbeat_ms"`
	RequestRate float64 `json:"request_rate"`
	ErrorRate   float64 `json:"error_rate"`
}
