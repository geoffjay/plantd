package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceConfig(t *testing.T) {
	t.Run("empty service config", func(t *testing.T) {
		config := ServiceConfig{}
		assert.Empty(t, config.ID)
	})

	t.Run("service config with ID", func(t *testing.T) {
		config := ServiceConfig{
			ID: "org.plantd.Service",
		}

		assert.Equal(t, "org.plantd.Service", config.ID)
	})

	t.Run("service config with different ID formats", func(t *testing.T) {
		testCases := []struct {
			name string
			id   string
		}{
			{"simple ID", "service"},
			{"dotted ID", "org.plantd.Service"},
			{"complex ID", "com.example.app.Service"},
			{"hyphenated ID", "plantd-service"},
			{"underscore ID", "plantd_service"},
			{"mixed format", "org.plantd.service-v1_test"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				config := ServiceConfig{
					ID: tc.id,
				}

				assert.Equal(t, tc.id, config.ID)
			})
		}
	})

	t.Run("service config with empty string ID", func(t *testing.T) {
		config := ServiceConfig{
			ID: "",
		}

		assert.Empty(t, config.ID)
	})

	t.Run("service config with whitespace ID", func(t *testing.T) {
		config := ServiceConfig{
			ID: "  org.plantd.Service  ",
		}

		// Should preserve whitespace as-is (validation would be done elsewhere)
		assert.Equal(t, "  org.plantd.Service  ", config.ID)
	})
}
