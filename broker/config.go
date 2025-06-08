package main

import (
	"sync"

	cfg "github.com/geoffjay/plantd/core/config"

	log "github.com/sirupsen/logrus"
)

type busConfig struct {
	Name     string `mapstructure:"name"`
	Frontend string `mapstructure:"frontend"`
	Backend  string `mapstructure:"backend"`
	Capture  string `mapstructure:"capture"`
}

// Config represents the configuration for the broker service.
type Config struct {
	cfg.Config

	Env               string            `mapstructure:"env"`
	Endpoint          string            `mapstructure:"endpoint"`
	ClientEndpoint    string            `mapstructure:"client-endpoint"`
	HeartbeatLiveness int               `mapstructure:"heartbeat-liveness"`
	HeartbeatInterval int               `mapstructure:"heartbeat-interval"`
	Buses             []busConfig       `mapstructure:"buses"`
	Log               cfg.LogConfig     `mapstructure:"log"`
	Service           cfg.ServiceConfig `mapstructure:"service"`
}

var lock = &sync.Mutex{}
var instance *Config

var defaults = map[string]interface{}{
	"env":                "development",
	"endpoint":           "tcp://*:9797",
	"client-endpoint":    "tcp://localhost:9797",
	"heartbeat-liveness": 3,
	"heartbeat-interval": 2500000,
	"buses": []map[string]string{
		{
			"name":     "state",
			"frontend": "@tcp://127.0.0.1:11000",
			"backend":  "@tcp://127.0.0.1:11001",
			"capture":  "inproc://broker.state.pipe",
		},
		{
			"name":     "event",
			"frontend": "@tcp://127.0.0.1:12000",
			"backend":  "@tcp://127.0.0.1:12001",
			"capture":  "inproc://broker.event.pipe",
		},
		{
			"name":     "metric",
			"frontend": "@tcp://127.0.0.1:13000",
			"backend":  "@tcp://127.0.0.1:13001",
			"capture":  "inproc://broker.metric.pipe",
		},
	},
	"log.formatter":    "text",
	"log.level":        "info",
	"log.loki.address": "http://localhost:3100",
	"log.loki.labels": map[string]string{
		"app": "broker", "environment": "development"},
	"service.id": "org.plantd.Broker",
}

// GetConfig returns the application configuration singleton.
func GetConfig() *Config {
	if instance == nil {
		lock.Lock()
		defer lock.Unlock()
		if instance == nil {
			if err := cfg.LoadConfigWithDefaults("broker", &instance,
				defaults); err != nil {
				log.Fatalf("error reading config file: %s\n", err)
			}
		}
	}

	log.Tracef("config: %+v", instance)

	return instance
}
