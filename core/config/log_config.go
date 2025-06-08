package config

// LokiConfig is a struct to store Loki configuration.
//
// Address is the address of the Loki server.
// Labels is a map of labels to be added to the log lines.
//
// Example:
//
//	address: "http://localhost:3100"
//	labels:
//	  app: "app"
//	  environment: "development"
type LokiConfig struct {
	Address string            `mapstructure:"address"`
	Labels  map[string]string `mapstructure:"labels"`
}

// LogConfig is a struct to store log configuration.
//
// Formatter is the log formatter to be used.
// Level is the log level to be used.
// Loki is the Loki configuration.
//
// Example:
//
//	formatter: "json"
//	level: "info"
//	loki: { see LokiConfig }
type LogConfig struct {
	Formatter string     `mapstructure:"formatter"`
	Level     string     `mapstructure:"level"`
	Loki      LokiConfig `mapstructure:"loki"`
}
