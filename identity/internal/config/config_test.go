package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDatabaseConfig(t *testing.T) {
	config := DatabaseConfig{
		Driver:   "postgres",
		Host:     "localhost",
		Port:     5432,
		Database: "testdb",
		Username: "testuser",
		Password: "testpass",
		SSLMode:  "disable",
		DSN:      "postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable",
	}

	assert.Equal(t, "postgres", config.Driver)
	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 5432, config.Port)
	assert.Equal(t, "testdb", config.Database)
	assert.Equal(t, "testuser", config.Username)
	assert.Equal(t, "testpass", config.Password)
	assert.Equal(t, "disable", config.SSLMode)
	assert.NotEmpty(t, config.DSN)
}

func TestServerConfig(t *testing.T) {
	config := ServerConfig{
		Port:         8080,
		ReadTimeout:  30,
		WriteTimeout: 30,
		IdleTimeout:  120,
	}

	assert.Equal(t, 8080, config.Port)
	assert.Equal(t, 30, config.ReadTimeout)
	assert.Equal(t, 30, config.WriteTimeout)
	assert.Equal(t, 120, config.IdleTimeout)
}

func TestSecurityConfig(t *testing.T) {
	config := SecurityConfig{
		JWTSecret:         "test-secret-key",
		JWTExpiration:     3600,
		RefreshExpiration: 604800,
		BcryptCost:        12,
		RateLimitRPS:      10,
		RateLimitBurst:    20,
	}

	assert.Equal(t, "test-secret-key", config.JWTSecret)
	assert.Equal(t, 3600, config.JWTExpiration)
	assert.Equal(t, 604800, config.RefreshExpiration)
	assert.Equal(t, 12, config.BcryptCost)
	assert.Equal(t, 10, config.RateLimitRPS)
	assert.Equal(t, 20, config.RateLimitBurst)
}

func TestConfig_Validate(t *testing.T) {
	config := &Config{
		Env: "test",
		Database: DatabaseConfig{
			Driver:   "sqlite",
			Host:     "localhost",
			Port:     5432,
			Database: "test",
			Username: "test",
			Password: "test",
			SSLMode:  "disable",
			DSN:      "test.db",
		},
		Server: ServerConfig{
			Port:         8080,
			ReadTimeout:  30,
			WriteTimeout: 30,
			IdleTimeout:  120,
		},
		Security: SecurityConfig{
			JWTSecret:         "test-secret-key",
			JWTExpiration:     3600,
			RefreshExpiration: 604800,
			BcryptCost:        12,
			RateLimitRPS:      10,
			RateLimitBurst:    20,
		},
	}

	// Test validation - currently returns nil but we test the structure
	err := config.Validate()
	assert.NoError(t, err)

	// Test that all fields are accessible
	assert.Equal(t, "test", config.Env)
	assert.NotNil(t, config.Database)
	assert.NotNil(t, config.Server)
	assert.NotNil(t, config.Security)
}

func TestDefaults_Keys(t *testing.T) {
	// Test that defaults map contains expected keys
	expectedKeys := []string{
		"env",
		"database.driver",
		"server.port",
		"security.jwt_secret",
		"log.formatter",
		"service.id",
	}

	for _, key := range expectedKeys {
		t.Run("has_default_"+key, func(t *testing.T) {
			assert.Contains(t, defaults, key)
			assert.NotNil(t, defaults[key])
		})
	}
}

func TestDefaults_Values(t *testing.T) {
	// Test specific default values
	assert.Equal(t, "development", defaults["env"])
	assert.Equal(t, "sqlite", defaults["database.driver"])
	assert.Equal(t, 8080, defaults["server.port"])
	assert.Equal(t, "change-me-in-production", defaults["security.jwt_secret"])
	assert.Equal(t, "text", defaults["log.formatter"])
	assert.Equal(t, "org.plantd.Identity", defaults["service.id"])
}
