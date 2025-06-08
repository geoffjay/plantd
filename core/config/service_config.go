package config

// ServiceConfig is a struct to store service configuration.
//
// ID is the ID of the service.
//
// Example:
//
//	id: "org.plantd.Service"
type ServiceConfig struct {
	ID string `mapstructure:"id"`
}
