// Package config provides configuration for the identity service.
package config

import (
	"sync"

	cfg "github.com/geoffjay/plantd/core/config"

	log "github.com/sirupsen/logrus"
)

// DatabaseConfig represents database configuration settings.
type DatabaseConfig struct {
	Driver   string `mapstructure:"driver"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Database string `mapstructure:"database"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	SSLMode  string `mapstructure:"ssl_mode"`
	DSN      string `mapstructure:"dsn"`
}

// ServerConfig represents server configuration settings.
type ServerConfig struct {
	Port         int `mapstructure:"port"`
	ReadTimeout  int `mapstructure:"read_timeout"`
	WriteTimeout int `mapstructure:"write_timeout"`
	IdleTimeout  int `mapstructure:"idle_timeout"`
}

// SecurityConfig represents security configuration settings.
type SecurityConfig struct {
	// JWT Configuration
	JWTSecret         string `mapstructure:"jwt_secret"`
	JWTRefreshSecret  string `mapstructure:"jwt_refresh_secret"`
	JWTExpiration     int    `mapstructure:"jwt_expiration"`
	RefreshExpiration int    `mapstructure:"refresh_expiration"`
	JWTIssuer         string `mapstructure:"jwt_issuer"`

	// Password Configuration
	BcryptCost          int  `mapstructure:"bcrypt_cost"`
	PasswordMinLength   int  `mapstructure:"password_min_length"`
	PasswordMaxLength   int  `mapstructure:"password_max_length"`
	RequireUppercase    bool `mapstructure:"require_uppercase"`
	RequireLowercase    bool `mapstructure:"require_lowercase"`
	RequireNumbers      bool `mapstructure:"require_numbers"`
	RequireSpecialChars bool `mapstructure:"require_special_chars"`

	// Rate Limiting Configuration
	RateLimitRPS           int `mapstructure:"rate_limit_rps"`
	RateLimitBurst         int `mapstructure:"rate_limit_burst"`
	MaxFailedAttempts      int `mapstructure:"max_failed_attempts"`
	LockoutDurationMinutes int `mapstructure:"lockout_duration_minutes"`

	// Registration Configuration
	AllowSelfRegistration         bool `mapstructure:"allow_self_registration"`
	RequireEmailVerification      bool `mapstructure:"require_email_verification"`
	EmailVerificationExpireyHours int  `mapstructure:"email_verification_expiry_hours"`
	PasswordResetExpiryHours      int  `mapstructure:"password_reset_expiry_hours"`
}

// Config represents the configuration for the identity service.
type Config struct {
	cfg.Config

	Env      string            `mapstructure:"env"`
	Database DatabaseConfig    `mapstructure:"database"`
	Server   ServerConfig      `mapstructure:"server"`
	Security SecurityConfig    `mapstructure:"security"`
	Log      cfg.LogConfig     `mapstructure:"log"`
	Service  cfg.ServiceConfig `mapstructure:"service"`
}

var lock = &sync.Mutex{}
var instance *Config

var defaults = map[string]interface{}{
	"env": "development",

	// Database defaults
	"database.driver":   "sqlite",
	"database.host":     "localhost",
	"database.port":     5432,
	"database.database": "identity",
	"database.username": "identity",
	"database.password": "",
	"database.ssl_mode": "disable",
	"database.dsn":      "identity.db",

	// Server defaults
	"server.port":          8080,
	"server.read_timeout":  30,
	"server.write_timeout": 30,
	"server.idle_timeout":  120,

	// Security defaults
	"security.jwt_secret":                      "change-me-in-production",
	"security.jwt_refresh_secret":              "change-me-in-production-too",
	"security.jwt_expiration":                  900,    // 15 minutes
	"security.refresh_expiration":              604800, // 7 days
	"security.jwt_issuer":                      "plantd-identity",
	"security.bcrypt_cost":                     12,
	"security.password_min_length":             8,
	"security.password_max_length":             128,
	"security.require_uppercase":               true,
	"security.require_lowercase":               true,
	"security.require_numbers":                 true,
	"security.require_special_chars":           true,
	"security.rate_limit_rps":                  10,
	"security.rate_limit_burst":                5,
	"security.max_failed_attempts":             5,
	"security.lockout_duration_minutes":        15,
	"security.allow_self_registration":         true,
	"security.require_email_verification":      true,
	"security.email_verification_expiry_hours": 24,
	"security.password_reset_expiry_hours":     2,

	// Logging defaults
	"log.formatter":    "text",
	"log.level":        "info",
	"log.loki.address": "http://localhost:3100",
	"log.loki.labels": map[string]string{
		"app": "identity", "environment": "development"},

	// Service defaults
	"service.id": "org.plantd.Identity",
}

// GetConfig returns the application configuration singleton.
func GetConfig() *Config {
	if instance == nil {
		lock.Lock()
		defer lock.Unlock()
		if instance == nil {
			if err := cfg.LoadConfigWithDefaults("identity", &instance,
				defaults); err != nil {
				log.Fatalf("error reading config file: %s\n", err)
			}
		}
	}

	log.Tracef("config: %+v", instance)

	return instance
}

// Validate validates the configuration settings.
func (c *Config) Validate() error {
	// TODO: Add configuration validation logic
	// - Check required fields
	// - Validate database connection settings
	// - Ensure JWT secret is set in production
	// - Validate port ranges
	return nil
}
