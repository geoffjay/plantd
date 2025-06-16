package mdp

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

// Config holds all configurable parameters for the MDP implementation
type Config struct {
	// Connection settings
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval" default:"2500ms"`
	HeartbeatLiveness int           `yaml:"heartbeat_liveness" default:"3"`
	ReconnectInterval time.Duration `yaml:"reconnect_interval" default:"2500ms"`
	RequestTimeout    time.Duration `yaml:"request_timeout" default:"5000ms"`

	// Retry and reliability settings
	MaxRetries       int           `yaml:"max_retries" default:"3"`
	RetryBackoffMin  time.Duration `yaml:"retry_backoff_min" default:"100ms"`
	RetryBackoffMax  time.Duration `yaml:"retry_backoff_max" default:"5000ms"`
	RetryBackoffMult float64       `yaml:"retry_backoff_multiplier" default:"2.0"`

	// Socket settings
	SocketHWM        int           `yaml:"socket_hwm" default:"1000"`
	SocketLinger     time.Duration `yaml:"socket_linger" default:"1000ms"`
	SocketRcvTimeout time.Duration `yaml:"socket_rcv_timeout" default:"1000ms"`
	SocketSndTimeout time.Duration `yaml:"socket_snd_timeout" default:"1000ms"`

	// Message settings
	MaxMessageSize    int  `yaml:"max_message_size" default:"1048576"` // 1MB
	EnableCompression bool `yaml:"enable_compression" default:"false"`

	// Logging and monitoring
	LogLevel        string        `yaml:"log_level" default:"info"`
	EnableMetrics   bool          `yaml:"enable_metrics" default:"true"`
	MetricsInterval time.Duration `yaml:"metrics_interval" default:"30s"`

	// MMI settings
	EnableMMI   bool     `yaml:"enable_mmi" default:"true"`
	MMIServices []string `yaml:"mmi_services" default:""`

	// Security settings
	EnableAuth       bool   `yaml:"enable_auth" default:"false"`
	EnableEncryption bool   `yaml:"enable_encryption" default:"false"`
	CertPath         string `yaml:"cert_path" default:""`
	KeyPath          string `yaml:"key_path" default:""`

	// Worker pool settings
	WorkerPoolSize    int           `yaml:"worker_pool_size" default:"10"`
	WorkerIdleTimeout time.Duration `yaml:"worker_idle_timeout" default:"60000ms"`

	// Broker settings
	PersistRequests bool     `yaml:"persist_requests" default:"false"`
	PersistPath     string   `yaml:"persist_path" default:"./mdp_persist"`
	ClusterMode     bool     `yaml:"cluster_mode" default:"false"`
	ClusterPeers    []string `yaml:"cluster_peers" default:""`
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	return &Config{
		HeartbeatInterval: 2500 * time.Millisecond,
		HeartbeatLiveness: 3,
		ReconnectInterval: 2500 * time.Millisecond,
		RequestTimeout:    5000 * time.Millisecond,
		MaxRetries:        3,
		RetryBackoffMin:   100 * time.Millisecond,
		RetryBackoffMax:   5000 * time.Millisecond,
		RetryBackoffMult:  2.0,
		SocketHWM:         1000,
		SocketLinger:      1000 * time.Millisecond,
		SocketRcvTimeout:  1000 * time.Millisecond,
		SocketSndTimeout:  1000 * time.Millisecond,
		MaxMessageSize:    1048576, // 1MB
		EnableCompression: false,
		LogLevel:          "info",
		EnableMetrics:     true,
		MetricsInterval:   30 * time.Second,
		EnableMMI:         true,
		MMIServices:       []string{MMIService, MMIWorkers, MMIHeartbeat, MMIBroker},
		EnableAuth:        false,
		EnableEncryption:  false,
		CertPath:          "",
		KeyPath:           "",
		WorkerPoolSize:    10,
		WorkerIdleTimeout: 60000 * time.Millisecond,
		PersistRequests:   false,
		PersistPath:       "./mdp_persist",
		ClusterMode:       false,
		ClusterPeers:      []string{},
	}
}

// LoadConfig loads configuration from a YAML file with environment variable overrides
func LoadConfig(filename string) (*Config, error) {
	config := DefaultConfig()

	// Load from file if it exists
	if filename != "" {
		if _, err := os.Stat(filename); err == nil {
			data, err := os.ReadFile(filename)
			if err != nil {
				return nil, fmt.Errorf("failed to read config file %s: %w", filename, err)
			}

			if err := yaml.Unmarshal(data, config); err != nil {
				return nil, fmt.Errorf("failed to parse config file %s: %w", filename, err)
			}
		}
	}

	// Override with environment variables
	config.applyEnvironmentOverrides()

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// applyEnvironmentOverrides applies environment variable overrides
func (c *Config) applyEnvironmentOverrides() { //nolint:cyclop
	// Connection settings
	if val := os.Getenv("MDP_HEARTBEAT_INTERVAL"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			c.HeartbeatInterval = duration
		}
	}
	if val := os.Getenv("MDP_HEARTBEAT_LIVENESS"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			c.HeartbeatLiveness = i
		}
	}
	if val := os.Getenv("MDP_RECONNECT_INTERVAL"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			c.ReconnectInterval = duration
		}
	}
	if val := os.Getenv("MDP_REQUEST_TIMEOUT"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			c.RequestTimeout = duration
		}
	}

	// Retry settings
	if val := os.Getenv("MDP_MAX_RETRIES"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			c.MaxRetries = i
		}
	}

	// Socket settings
	if val := os.Getenv("MDP_SOCKET_HWM"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			c.SocketHWM = i
		}
	}

	// Message settings
	if val := os.Getenv("MDP_MAX_MESSAGE_SIZE"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			c.MaxMessageSize = i
		}
	}
	if val := os.Getenv("MDP_ENABLE_COMPRESSION"); val != "" {
		c.EnableCompression = strings.ToLower(val) == BoolTrue
	}

	// Logging and monitoring
	if val := os.Getenv("MDP_LOG_LEVEL"); val != "" {
		c.LogLevel = val
	}
	if val := os.Getenv("MDP_ENABLE_METRICS"); val != "" {
		c.EnableMetrics = strings.ToLower(val) == BoolTrue
	}

	// MMI settings
	if val := os.Getenv("MDP_ENABLE_MMI"); val != "" {
		c.EnableMMI = strings.ToLower(val) == BoolTrue
	}

	// Security settings
	if val := os.Getenv("MDP_ENABLE_AUTH"); val != "" {
		c.EnableAuth = strings.ToLower(val) == BoolTrue
	}
	if val := os.Getenv("MDP_ENABLE_ENCRYPTION"); val != "" {
		c.EnableEncryption = strings.ToLower(val) == BoolTrue
	}
	if val := os.Getenv("MDP_CERT_PATH"); val != "" {
		c.CertPath = val
	}
	if val := os.Getenv("MDP_KEY_PATH"); val != "" {
		c.KeyPath = val
	}

	// Worker pool settings
	if val := os.Getenv("MDP_WORKER_POOL_SIZE"); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			c.WorkerPoolSize = i
		}
	}

	// Broker settings
	if val := os.Getenv("MDP_PERSIST_REQUESTS"); val != "" {
		c.PersistRequests = strings.ToLower(val) == BoolTrue
	}
	if val := os.Getenv("MDP_PERSIST_PATH"); val != "" {
		c.PersistPath = val
	}
	if val := os.Getenv("MDP_CLUSTER_MODE"); val != "" {
		c.ClusterMode = strings.ToLower(val) == BoolTrue
	}
}

// Validate validates the configuration parameters
func (c *Config) Validate() error { //nolint:cyclop
	// Validate timing parameters
	if c.HeartbeatInterval <= 0 {
		return fmt.Errorf("heartbeat_interval must be positive")
	}
	if c.HeartbeatLiveness <= 0 {
		return fmt.Errorf("heartbeat_liveness must be positive")
	}
	if c.ReconnectInterval <= 0 {
		return fmt.Errorf("reconnect_interval must be positive")
	}
	if c.RequestTimeout <= 0 {
		return fmt.Errorf("request_timeout must be positive")
	}

	// Validate retry parameters
	if c.MaxRetries < 0 {
		return fmt.Errorf("max_retries cannot be negative")
	}
	if c.RetryBackoffMin <= 0 {
		return fmt.Errorf("retry_backoff_min must be positive")
	}
	if c.RetryBackoffMax < c.RetryBackoffMin {
		return fmt.Errorf("retry_backoff_max must be >= retry_backoff_min")
	}
	if c.RetryBackoffMult <= 1.0 {
		return fmt.Errorf("retry_backoff_multiplier must be > 1.0")
	}

	// Validate socket parameters
	if c.SocketHWM <= 0 {
		return fmt.Errorf("socket_hwm must be positive")
	}

	// Validate message parameters
	if c.MaxMessageSize <= 0 {
		return fmt.Errorf("max_message_size must be positive")
	}
	if c.MaxMessageSize > 100*1024*1024 { // 100MB limit
		return fmt.Errorf("max_message_size too large (max 100MB)")
	}

	// Validate log level
	validLogLevels := []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}
	valid := false
	for _, level := range validLogLevels {
		if strings.ToLower(c.LogLevel) == level {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid log_level: %s (valid: %s)", c.LogLevel, strings.Join(validLogLevels, ", "))
	}

	// Validate worker pool settings
	if c.WorkerPoolSize <= 0 {
		return fmt.Errorf("worker_pool_size must be positive")
	}
	if c.WorkerIdleTimeout <= 0 {
		return fmt.Errorf("worker_idle_timeout must be positive")
	}

	// Validate security settings
	if c.EnableEncryption && (c.CertPath == "" || c.KeyPath == "") {
		return fmt.Errorf("cert_path and key_path required when encryption is enabled")
	}

	// Validate persistence settings
	if c.PersistRequests && c.PersistPath == "" {
		return fmt.Errorf("persist_path required when request persistence is enabled")
	}

	return nil
}

// Save saves the configuration to a YAML file
func (c *Config) Save(filename string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write config file %s: %w", filename, err)
	}

	return nil
}

// String returns a string representation of the configuration
func (c *Config) String() string {
	data, _ := yaml.Marshal(c)
	return string(data)
}
