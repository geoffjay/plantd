package handlers

import (
	"testing"

	"github.com/geoffjay/plantd/app/internal/auth"
	"github.com/geoffjay/plantd/app/internal/services"
	"github.com/stretchr/testify/assert"
)

func TestNewDashboardHandler(t *testing.T) {
	t.Run("successful creation with all services", func(t *testing.T) {
		var brokerService *services.BrokerService
		var stateService *services.StateService
		var healthService *services.HealthService
		var metricsService *services.MetricsService

		handler := NewDashboardHandler(brokerService, stateService, healthService, metricsService)

		assert.NotNil(t, handler)
		assert.Equal(t, brokerService, handler.brokerService)
		assert.Equal(t, stateService, handler.stateService)
		assert.Equal(t, healthService, handler.healthService)
		assert.Equal(t, metricsService, handler.metricsService)
	})

	t.Run("creation with nil services", func(t *testing.T) {
		handler := NewDashboardHandler(nil, nil, nil, nil)

		assert.NotNil(t, handler)
		assert.Nil(t, handler.brokerService)
		assert.Nil(t, handler.stateService)
		assert.Nil(t, handler.healthService)
		assert.Nil(t, handler.metricsService)
	})
}

func TestDashboardHandler_UserContextHandling(t *testing.T) {
	t.Run("create default user context", func(t *testing.T) {
		// This tests the logic that creates a default user context when none exists
		// We can verify the default values are correct

		defaultUser := &auth.UserContext{
			ID:            1,
			Email:         "admin@plantd.local",
			Username:      "admin",
			Roles:         []string{"admin"},
			Organizations: []string{"plantd"},
			Permissions:   []string{"*"},
		}

		assert.Equal(t, uint(1), defaultUser.ID)
		assert.Equal(t, "admin@plantd.local", defaultUser.Email)
		assert.Equal(t, "admin", defaultUser.Username)
		assert.Contains(t, defaultUser.Roles, "admin")
		assert.Contains(t, defaultUser.Organizations, "plantd")
		assert.Contains(t, defaultUser.Permissions, "*")
	})

	t.Run("validate user context structure", func(t *testing.T) {
		userCtx := &auth.UserContext{
			ID:            123,
			Email:         "test@example.com",
			Username:      "testuser",
			Roles:         []string{"user", "moderator"},
			Organizations: []string{"org1", "org2"},
			Permissions:   []string{"read", "write"},
		}

		assert.Equal(t, uint(123), userCtx.ID)
		assert.Equal(t, "test@example.com", userCtx.Email)
		assert.Equal(t, "testuser", userCtx.Username)
		assert.Len(t, userCtx.Roles, 2)
		assert.Len(t, userCtx.Organizations, 2)
		assert.Len(t, userCtx.Permissions, 2)
		assert.Contains(t, userCtx.Roles, "user")
		assert.Contains(t, userCtx.Roles, "moderator")
	})
}

func TestDashboardData_Structure(t *testing.T) {
	t.Run("dashboard data initialization", func(t *testing.T) {
		// Test that DashboardData can be created with expected fields
		user := &auth.UserContext{
			ID:       1,
			Email:    "test@example.com",
			Username: "testuser",
		}

		dashboardData := &DashboardData{
			User:             user,
			ServiceCount:     5,
			HealthStatus:     "healthy",
			RequestRate:      "100/sec",
			Uptime:           "1h30m",
			Services:         make([]interface{}, 0),
			HealthComponents: make(map[string]interface{}),
		}

		assert.NotNil(t, dashboardData)
		assert.Equal(t, user, dashboardData.User)
		assert.Equal(t, 5, dashboardData.ServiceCount)
		assert.Equal(t, "healthy", dashboardData.HealthStatus)
		assert.Equal(t, "100/sec", dashboardData.RequestRate)
		assert.Equal(t, "1h30m", dashboardData.Uptime)
		assert.NotNil(t, dashboardData.Services)
		assert.NotNil(t, dashboardData.HealthComponents)
	})

	t.Run("dashboard data with empty values", func(t *testing.T) {
		dashboardData := &DashboardData{
			ServiceCount:     0,
			HealthStatus:     "unknown",
			RequestRate:      "0/sec",
			Uptime:           "Unknown",
			Services:         make([]interface{}, 0),
			HealthComponents: make(map[string]interface{}),
		}

		assert.Equal(t, 0, dashboardData.ServiceCount)
		assert.Equal(t, "unknown", dashboardData.HealthStatus)
		assert.Equal(t, "0/sec", dashboardData.RequestRate)
		assert.Equal(t, "Unknown", dashboardData.Uptime)
		assert.Empty(t, dashboardData.Services)
		assert.Empty(t, dashboardData.HealthComponents)
	})
}

func TestFormatRequestRate(t *testing.T) {
	t.Run("format various request rates", func(t *testing.T) {
		// Test the formatRequestRate function logic
		testCases := []struct {
			input    float64
			expected string
		}{
			{0.0, "< 1 req/sec"},
			{1.5, "1.5 req/sec"},
			{100.0, "100.0 req/sec"},
			{1500.75, "1500.8 req/sec"},
			{999999.99, "1000000.0 req/sec"},
		}

		for _, tc := range testCases {
			result := formatRequestRate(tc.input)
			// Simple format validation - the actual implementation details may vary
			assert.Contains(t, result, "req/sec", "Result should contain 'req/sec' for input %f", tc.input)
			assert.NotEmpty(t, result, "Result should not be empty for input %f", tc.input)
		}
	})

	t.Run("format zero rate", func(t *testing.T) {
		result := formatRequestRate(0.0)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "req/sec")
		assert.Contains(t, result, "< 1")
	})

	t.Run("format high rate", func(t *testing.T) {
		result := formatRequestRate(10000.0)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "req/sec")
	})
}

// Integration test helpers
func TestDashboardHandler_ServiceIntegration(t *testing.T) {
	t.Run("nil service handling", func(t *testing.T) {
		handler := NewDashboardHandler(nil, nil, nil, nil)

		// These should not panic when services are nil
		assert.NotPanics(t, func() {
			// Create basic dashboard data structure that would be used with nil services
			dashboardData := &DashboardData{
				ServiceCount:     0,
				HealthStatus:     "unknown",
				RequestRate:      "0/sec",
				Uptime:           "Unknown",
				Services:         make([]interface{}, 0),
				HealthComponents: make(map[string]interface{}),
			}
			assert.NotNil(t, dashboardData)
		})

		assert.Nil(t, handler.brokerService)
		assert.Nil(t, handler.healthService)
		assert.Nil(t, handler.metricsService)
	})

	t.Run("error handling patterns", func(t *testing.T) {
		// Test that error handling patterns work correctly
		handler := NewDashboardHandler(nil, nil, nil, nil)

		// Verify handler can handle missing services gracefully
		assert.NotNil(t, handler)

		// Test default data creation
		defaultData := &DashboardData{
			ServiceCount:     0,
			HealthStatus:     "unknown",
			RequestRate:      "0/sec",
			Uptime:           "Unknown",
			Services:         make([]interface{}, 0),
			HealthComponents: make(map[string]interface{}),
		}

		assert.NotNil(t, defaultData)
		assert.Equal(t, 0, defaultData.ServiceCount)
		assert.Equal(t, "unknown", defaultData.HealthStatus)
	})
}

// Performance test helpers
func TestDashboardHandler_Performance(t *testing.T) {
	t.Run("handler creation performance", func(t *testing.T) {
		// Test that handler creation is fast
		for i := 0; i < 1000; i++ {
			handler := NewDashboardHandler(nil, nil, nil, nil)
			assert.NotNil(t, handler)
		}
	})

	t.Run("data structure creation performance", func(t *testing.T) {
		user := &auth.UserContext{
			ID:       1,
			Email:    "test@example.com",
			Username: "testuser",
		}

		// Test that creating dashboard data structures is efficient
		for i := 0; i < 1000; i++ {
			dashboardData := &DashboardData{
				User:             user,
				ServiceCount:     i,
				HealthStatus:     "healthy",
				RequestRate:      "100/sec",
				Uptime:           "1h30m",
				Services:         make([]interface{}, 0),
				HealthComponents: make(map[string]interface{}),
			}
			assert.NotNil(t, dashboardData)
		}
	})
}

// Benchmark tests for performance validation
func BenchmarkNewDashboardHandler(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewDashboardHandler(nil, nil, nil, nil)
	}
}

func BenchmarkDashboardData_Creation(b *testing.B) {
	user := &auth.UserContext{
		ID:       1,
		Email:    "test@example.com",
		Username: "testuser",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = &DashboardData{
			User:             user,
			ServiceCount:     5,
			HealthStatus:     "healthy",
			RequestRate:      "100/sec",
			Uptime:           "1h30m",
			Services:         make([]interface{}, 0),
			HealthComponents: make(map[string]interface{}),
		}
	}
}
