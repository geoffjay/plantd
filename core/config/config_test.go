package config

import (
	"os"
	"reflect"
	"testing"
)

type object struct {
	IsTest bool   `mapstructure:"is-test"`
	Test   string `mapstructure:"test"`
}

type testConfig struct {
	Env    string `mapstructure:"env"`
	Object object `mapstructure:"object"`
}

func buildConfig(t *testing.T, path string) *testConfig {
	_ = os.Setenv("PLANTD_TEST_CONFIG", path)
	var config testConfig
	err := LoadConfig("test", &config)
	if err != nil {
		t.Fatalf("Cannot create configuration: %v", err)
	}

	return &config
}

// nolint: funlen
func TestLoadConfig(t *testing.T) {
	emptyConfig := &testConfig{
		Env: "",
		Object: object{
			IsTest: false,
			Test:   "",
		},
	}
	validConfig := &testConfig{
		Env: "testing",
		Object: object{
			IsTest: true,
			Test:   "config",
		},
	}
	cases := []struct {
		fixture string
		want    *testConfig
		name    string
	}{
		{
			fixture: "../fixtures/config/empty.yaml",
			want:    emptyConfig,
			name:    "EmptyFixture",
		},
		{
			fixture: "../fixtures/config/valid.yaml",
			want:    validConfig,
			name:    "ValidYAMLFixture",
		},
		{
			fixture: "../fixtures/config/valid.json",
			want:    validConfig,
			name:    "ValidJSONFixture",
		},
		{
			fixture: "../fixtures/config/valid.toml",
			want:    validConfig,
			name:    "ValidTOMLFixture",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := buildConfig(t, tc.fixture)

			// split up data to make it easier to see where an error occurs
			got := map[string]interface{}{
				"env":    config.Env,
				"object": config.Object,
			}

			wanted := map[string]interface{}{
				"env":    tc.want.Env,
				"object": tc.want.Object,
			}

			for key, value := range wanted {
				if !reflect.DeepEqual(value, got[key]) {
					t.Errorf("Expected: %v, got: %v", wanted[key], got[key])
				}
			}
		})
	}
}
