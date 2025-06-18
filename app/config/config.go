// Package config provides application configuration functionality.
package config

import (
	"sync"

	cfg "github.com/geoffjay/plantd/core/config"

	log "github.com/sirupsen/logrus"
)

// TODO:
// - add a new configuration section for the database

// Config represents the application configuration structure.
type Config struct {
	cfg.Config

	Env            string        `mapstructure:"env"`
	ClientEndpoint string        `mapstructure:"client-endpoint"`
	Log            cfg.LogConfig `mapstructure:"log"`
	Cors           corsConfig    `mapstructure:"cors"`
	Session        sessionConfig `mapstructure:"session"`

	// Identity Service integration
	Identity struct {
		Endpoint string `yaml:"endpoint" env:"PLANTD_APP_IDENTITY_ENDPOINT"`
		Timeout  string `yaml:"timeout" env:"PLANTD_APP_IDENTITY_TIMEOUT"`
		ClientID string `yaml:"client_id" env:"PLANTD_APP_IDENTITY_CLIENT_ID"`
	} `yaml:"identity" mapstructure:"identity"`

	// Services endpoints
	Services struct {
		BrokerEndpoint string `yaml:"broker_endpoint" env:"PLANTD_APP_BROKER_ENDPOINT"`
		StateEndpoint  string `yaml:"state_endpoint" env:"PLANTD_APP_STATE_ENDPOINT"`
		Timeout        string `yaml:"timeout" env:"PLANTD_APP_SERVICES_TIMEOUT"`
	} `yaml:"services" mapstructure:"services"`

	// Enhanced session configuration
	EnhancedSession struct {
		SecretKey  string `yaml:"secret_key" env:"PLANTD_APP_SESSION_SECRET"`
		CookieName string `yaml:"cookie_name" env:"PLANTD_APP_SESSION_COOKIE"`
		MaxAge     int    `yaml:"max_age" env:"PLANTD_APP_SESSION_MAX_AGE"`
		Secure     bool   `yaml:"secure" env:"PLANTD_APP_SESSION_SECURE"`
		HTTPOnly   bool   `yaml:"http_only" env:"PLANTD_APP_SESSION_HTTP_ONLY"`
	} `yaml:"enhanced_session" mapstructure:"enhanced_session"`

	// Feature flags
	Features struct {
		EnableMetrics bool `yaml:"enable_metrics" env:"PLANTD_APP_ENABLE_METRICS"`
		EnableConfig  bool `yaml:"enable_config" env:"PLANTD_APP_ENABLE_CONFIG"`
		EnableHealth  bool `yaml:"enable_health" env:"PLANTD_APP_ENABLE_HEALTH"`
	} `yaml:"features" mapstructure:"features"`
}

var lock = &sync.Mutex{}
var instance *Config

var defaults = map[string]interface{}{
	"env":                    "development",
	"client-endpoint":        "tcp://localhost:9797",
	"log.formatter":          "text",
	"log.level":              "info",
	"log.loki.address":       "http://localhost:3100",
	"log.loki.labels":        map[string]string{"app": "app", "environment": "development"},
	"cors.allow-credentials": true,
	"cors.allow-origins":     "https://localhost:8443,http://localhost:8443,https://127.0.0.1:8443,http://127.0.0.1:8443",
	"cors.allow-headers": "Origin, Content-Type, Accept, Content-Length, Accept-Language, " +
		"Accept-Encoding, Connection, Authorization, Access-Control-Allow-Origin, " +
		"Access-Control-Allow-Methods, Access-Control-Allow-Headers, Access-Control-Allow-Origin",
	"cors.allow-methods":       "GET, POST, HEAD, PUT, DELETE, PATCH, OPTIONS",
	"session.expiration":       "2h",
	"session.key-lookup":       "cookie:__Host-session",
	"session.cookie-secure":    true,
	"session.cookie-http-only": true,
	"session.cookie-same-site": "Lax",

	// Identity Service defaults
	"identity.endpoint":  "tcp://127.0.0.1:7200",
	"identity.timeout":   "30s",
	"identity.client_id": "plantd-app",

	// Services defaults
	"services.broker_endpoint": "tcp://127.0.0.1:9797",
	"services.state_endpoint":  "tcp://127.0.0.1:7300",
	"services.timeout":         "30s",

	// Enhanced session defaults
	"enhanced_session.secret_key":  "", // Must be set via environment variable
	"enhanced_session.cookie_name": "__Host-plantd-session",
	"enhanced_session.max_age":     7200, // 2 hours
	"enhanced_session.secure":      true,
	"enhanced_session.http_only":   true,

	// Feature flags defaults
	"features.enable_metrics": true,
	"features.enable_config":  true,
	"features.enable_health":  true,
}

// GetConfig returns the application configuration singleton.
func GetConfig() *Config {
	if instance == nil {
		lock.Lock()
		defer lock.Unlock()
		if instance == nil {
			if err := cfg.LoadConfigWithDefaults("app", &instance, defaults); err != nil {
				log.Fatalf("error reading config file: %s\n", err)
			}
		}
	}

	log.Tracef("config: %+v", instance)

	return instance
}
