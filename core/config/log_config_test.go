package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLokiConfig(t *testing.T) {
	t.Run("empty loki config", func(t *testing.T) {
		config := LokiConfig{}
		assert.Empty(t, config.Address)
		assert.Nil(t, config.Labels)
	})

	t.Run("loki config with values", func(t *testing.T) {
		config := LokiConfig{
			Address: "http://localhost:3100",
			Labels: map[string]string{
				"service": "plantd",
				"env":     "test",
			},
		}

		assert.Equal(t, "http://localhost:3100", config.Address)
		assert.Equal(t, "plantd", config.Labels["service"])
		assert.Equal(t, "test", config.Labels["env"])
		assert.Len(t, config.Labels, 2)
	})

	t.Run("loki config with empty labels map", func(t *testing.T) {
		config := LokiConfig{
			Address: "http://localhost:3100",
			Labels:  make(map[string]string),
		}

		assert.Equal(t, "http://localhost:3100", config.Address)
		assert.NotNil(t, config.Labels)
		assert.Len(t, config.Labels, 0)
	})
}

func TestLogConfigEmpty(t *testing.T) {
	config := LogConfig{}
	assert.Empty(t, config.Formatter)
	assert.Empty(t, config.Level)
	assert.Empty(t, config.Loki.Address)
	assert.Nil(t, config.Loki.Labels)
}

func TestLogConfigTextFormatter(t *testing.T) {
	config := LogConfig{
		Formatter: "text",
		Level:     "info",
		Loki: LokiConfig{
			Address: "http://localhost:3100",
			Labels: map[string]string{
				"service": "plantd",
			},
		},
	}

	assert.Equal(t, "text", config.Formatter)
	assert.Equal(t, "info", config.Level)
	assert.Equal(t, "http://localhost:3100", config.Loki.Address)
	assert.Equal(t, "plantd", config.Loki.Labels["service"])
}

func TestLogConfigJSONFormatter(t *testing.T) {
	config := LogConfig{
		Formatter: "json",
		Level:     "debug",
		Loki: LokiConfig{
			Address: "http://loki.example.com:3100",
			Labels: map[string]string{
				"app":         "plantd",
				"environment": "production",
				"version":     "1.0.0",
			},
		},
	}

	assert.Equal(t, "json", config.Formatter)
	assert.Equal(t, "debug", config.Level)
	assert.Equal(t, "http://loki.example.com:3100", config.Loki.Address)
	assert.Equal(t, "plantd", config.Loki.Labels["app"])
	assert.Equal(t, "production", config.Loki.Labels["environment"])
	assert.Equal(t, "1.0.0", config.Loki.Labels["version"])
	assert.Len(t, config.Loki.Labels, 3)
}

func TestLogConfigLogLevels(t *testing.T) {
	levels := []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}

	for _, level := range levels {
		t.Run(level, func(t *testing.T) {
			config := LogConfig{
				Formatter: "text",
				Level:     level,
			}

			assert.Equal(t, level, config.Level)
		})
	}
}

func TestLogConfigInvalidFormatter(t *testing.T) {
	config := LogConfig{
		Formatter: "invalid-formatter",
		Level:     "info",
	}

	assert.Equal(t, "invalid-formatter", config.Formatter)
	assert.Equal(t, "info", config.Level)
}

func TestLogConfigNestedLoki(t *testing.T) {
	config := LogConfig{
		Formatter: "json",
		Level:     "warn",
		Loki: LokiConfig{
			Address: "https://logs.example.com",
			Labels: map[string]string{
				"datacenter": "us-west-1",
				"cluster":    "prod-cluster",
			},
		},
	}

	// Test that nested config is properly accessible
	assert.Equal(t, "json", config.Formatter)
	assert.Equal(t, "warn", config.Level)
	assert.Equal(t, "https://logs.example.com", config.Loki.Address)
	assert.Equal(t, "us-west-1", config.Loki.Labels["datacenter"])
	assert.Equal(t, "prod-cluster", config.Loki.Labels["cluster"])
}
