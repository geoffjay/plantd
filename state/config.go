package main

import (
	"sync"

	cfg "github.com/geoffjay/plantd/core/config"

	log "github.com/sirupsen/logrus"
)

type databaseConfig struct {
	Adapter string `mapstructure:"adapter"`
	URI     string `mapstructure:"uri"`
}

type identityConfig struct {
	Endpoint string `mapstructure:"endpoint"`
	Timeout  string `mapstructure:"timeout"`
	Retries  int    `mapstructure:"retries"`
}

// Config represents the configuration for the state service.
type Config struct {
	cfg.Config

	Env            string            `mapstructure:"env"`
	BrokerEndpoint string            `mapstructure:"broker-endpoint"`
	StateEndpoint  string            `mapstructure:"state-endpoint"`
	Database       databaseConfig    `mapstructure:"database"`
	Identity       identityConfig    `mapstructure:"identity"`
	Log            cfg.LogConfig     `mapstructure:"log"`
	Service        cfg.ServiceConfig `mapstructure:"service"`
}

var lock = &sync.Mutex{}
var instance *Config

var defaults = map[string]interface{}{
	"env":               "development",
	"broker-endpoint":   "tcp://localhost:9797",
	"state-endpoint":    ">tcp://localhost:11001",
	"database.adapter":  "bbolt",
	"database.uri":      "plantd-state.db",
	"identity.endpoint": "tcp://127.0.0.1:9797",
	"identity.timeout":  "30s",
	"identity.retries":  3,
	"log.formatter":     "text",
	"log.level":         "info",
	"log.loki.address":  "http://localhost:3100",
	"log.loki.labels": map[string]string{
		"app": "state", "environment": "development"},
	"service.id": "org.plantd.State",
}

// GetConfig returns the application configuration singleton.
func GetConfig() *Config {
	if instance == nil {
		lock.Lock()
		defer lock.Unlock()
		if instance == nil {
			if err := cfg.LoadConfigWithDefaults("state", &instance,
				defaults); err != nil {
				log.Fatalf("error reading config file: %s\n", err)
			}
		}
	}

	log.Tracef("config: %+v", instance)

	return instance
}
