package main

import (
	"os"
	"reflect"
	"testing"

	cfg "github.com/geoffjay/plantd/core/config"
)

func buildConfig(t *testing.T, path string) *Config {
	var config *Config

	_ = os.Unsetenv("PLANTD_BROKER_ENDPOINT")

	_ = os.Setenv("PLANTD_BROKER_CONFIG", path)
	err := cfg.LoadConfig("broker", &config)
	if err != nil {
		t.Fatalf("Cannot create configuration: %v", err)
	}

	return config
}

// nolint: funlen
func TestLoadConfig(t *testing.T) {
	emptyConfig := &Config{
		Env:               "",
		Endpoint:          "",
		HeartbeatLiveness: 0,
		HeartbeatInterval: 0,
		Log: cfg.LogConfig{
			Formatter: "",
			Level:     "",
		},
	}
	validConfig := &Config{
		Env:               "testing",
		Endpoint:          "tcp://*:7200",
		ClientEndpoint:    "tcp://localhost:7200",
		HeartbeatLiveness: 3,
		HeartbeatInterval: 2500000,
		Log: cfg.LogConfig{
			Formatter: "text",
			Level:     "debug",
		},
	}
	cases := []struct {
		fixture string
		want    *Config
		name    string
	}{
		{
			fixture: "fixtures/config/empty.yaml",
			want:    emptyConfig,
			name:    "EmptyFixture",
		},
		{
			fixture: "fixtures/config/valid.yaml",
			want:    validConfig,
			name:    "ValidYAMLFixture",
		},
		{
			fixture: "fixtures/config/valid.json",
			want:    validConfig,
			name:    "ValidJSONFixture",
		},
		{
			fixture: "fixtures/config/valid.toml",
			want:    validConfig,
			name:    "ValidTOMLFixture",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := buildConfig(t, tc.fixture)

			// split up data to make it easier to see where an error occurs
			got := map[string]interface{}{
				"env":      config.Env,
				"endpoint": config.Endpoint,
				"log":      config.Log,
			}

			wanted := map[string]interface{}{
				"env":      tc.want.Env,
				"endpoint": tc.want.Endpoint,
				"log":      tc.want.Log,
			}

			for key, value := range wanted {
				if !reflect.DeepEqual(value, got[key]) {
					t.Errorf("Expected: %v, got: %v", wanted[key], got[key])
				}
			}
		})
	}
}
