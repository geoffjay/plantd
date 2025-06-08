package log

import (
	"testing"

	"github.com/geoffjay/plantd/core/config"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func setupTest() (log.Level, log.Formatter) {
	return log.GetLevel(), log.StandardLogger().Formatter
}

func teardownTest(originalLevel log.Level, originalFormatter log.Formatter) {
	log.SetLevel(originalLevel)
	log.SetFormatter(originalFormatter)
	log.StandardLogger().ReplaceHooks(make(log.LevelHooks))
}

func TestInitializeTextFormatter(t *testing.T) {
	originalLevel, originalFormatter := setupTest()
	defer teardownTest(originalLevel, originalFormatter)

	logConfig := config.LogConfig{
		Level:     "info",
		Formatter: "text",
		Loki: config.LokiConfig{
			Address: "http://localhost:3100",
			Labels:  map[string]string{"service": "test"},
		},
	}

	Initialize(logConfig)

	assert.Equal(t, log.InfoLevel, log.GetLevel())
	assert.IsType(t, &log.TextFormatter{}, log.StandardLogger().Formatter)

	// Check TextFormatter configuration
	textFormatter := log.StandardLogger().Formatter.(*log.TextFormatter)
	assert.True(t, textFormatter.FullTimestamp)
	assert.Equal(t, "2006-01-02 15:04:05", textFormatter.TimestampFormat)
}

func TestInitializeJSONFormatter(t *testing.T) {
	originalLevel, originalFormatter := setupTest()
	defer teardownTest(originalLevel, originalFormatter)

	logConfig := config.LogConfig{
		Level:     "debug",
		Formatter: "json",
		Loki: config.LokiConfig{
			Address: "http://localhost:3100",
			Labels:  map[string]string{"service": "test"},
		},
	}

	Initialize(logConfig)

	assert.Equal(t, log.DebugLevel, log.GetLevel())
	assert.IsType(t, &log.JSONFormatter{}, log.StandardLogger().Formatter)

	// Check JSONFormatter configuration
	jsonFormatter := log.StandardLogger().Formatter.(*log.JSONFormatter)
	assert.Equal(t, "2006-01-02 15:04:05", jsonFormatter.TimestampFormat)
}

func TestInitializeInvalidLevel(t *testing.T) {
	originalLevel, originalFormatter := setupTest()
	defer teardownTest(originalLevel, originalFormatter)

	logConfig := config.LogConfig{
		Level:     "invalid-level",
		Formatter: "text",
		Loki: config.LokiConfig{
			Address: "http://localhost:3100",
			Labels:  map[string]string{"service": "test"},
		},
	}

	Initialize(logConfig)

	// Level should remain unchanged when invalid
	assert.Equal(t, originalLevel, log.GetLevel())
}

func TestInitializeLogLevels(t *testing.T) {
	originalLevel, originalFormatter := setupTest()
	defer teardownTest(originalLevel, originalFormatter)

	testCases := []struct {
		level    string
		expected log.Level
	}{
		{"trace", log.TraceLevel},
		{"debug", log.DebugLevel},
		{"info", log.InfoLevel},
		{"warn", log.WarnLevel},
		{"error", log.ErrorLevel},
		{"fatal", log.FatalLevel},
		{"panic", log.PanicLevel},
	}

	for _, tc := range testCases {
		t.Run(tc.level, func(t *testing.T) {
			logConfig := config.LogConfig{
				Level:     tc.level,
				Formatter: "text",
				Loki: config.LokiConfig{
					Address: "http://localhost:3100",
					Labels:  map[string]string{"service": "test"},
				},
			}

			Initialize(logConfig)
			assert.Equal(t, tc.expected, log.GetLevel())
		})
	}
}

func TestInitializeEmptyFormatter(t *testing.T) {
	originalLevel, originalFormatter := setupTest()
	defer teardownTest(originalLevel, originalFormatter)

	logConfig := config.LogConfig{
		Level:     "info",
		Formatter: "", // Empty should default to text
		Loki: config.LokiConfig{
			Address: "http://localhost:3100",
			Labels:  map[string]string{"service": "test"},
		},
	}

	Initialize(logConfig)
	assert.IsType(t, &log.TextFormatter{}, log.StandardLogger().Formatter)
}

func TestInitializeLokiConfiguration(t *testing.T) {
	originalLevel, originalFormatter := setupTest()
	defer teardownTest(originalLevel, originalFormatter)

	logConfig := config.LogConfig{
		Level:     "info",
		Formatter: "json",
		Loki: config.LokiConfig{
			Address: "http://localhost:3100",
			Labels: map[string]string{
				"service": "plantd-test",
				"env":     "testing",
			},
		},
	}

	// Clear existing hooks first
	log.StandardLogger().ReplaceHooks(make(log.LevelHooks))

	Initialize(logConfig)

	// Check that hooks were added (we can't easily test the Loki hook directly)
	hooks := log.StandardLogger().Hooks
	assert.NotEmpty(t, hooks)

	// Check that hooks are registered for the expected levels
	expectedLevels := []log.Level{
		log.InfoLevel,
		log.WarnLevel,
		log.ErrorLevel,
		log.FatalLevel,
	}

	for _, level := range expectedLevels {
		assert.NotEmpty(t, hooks[level], "Expected hook for level %s", level)
	}
}

func TestInitializeMinimalConfig(t *testing.T) {
	originalLevel, originalFormatter := setupTest()
	defer teardownTest(originalLevel, originalFormatter)

	logConfig := config.LogConfig{}

	// Should not panic with empty config
	assert.NotPanics(t, func() {
		Initialize(logConfig)
	})
}
