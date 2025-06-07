package main

import (
	"sync"

	cfg "github.com/geoffjay/plantd/core/config"

	log "github.com/sirupsen/logrus"
)

type Config struct {
	cfg.Config

	Env            string        `mapstructure:"env"`
	ClientEndpoint string        `mapstructure:"client-endpoint"`
	Log            cfg.LogConfig `mapstructure:"log"`
}

var lock = &sync.Mutex{}
var instance *Config

var defaults = map[string]interface{}{
	"env":              "development",
	"client-endpoint":  "tcp://localhost:9797",
	"log.formatter":    "text",
	"log.level":        "info",
	"log.loki.address": "http://localhost:3100",
	"log.loki.labels":  map[string]string{"app": "proxy", "environment": "development"},
}

// GetConfig returns the application configuration singleton.
func GetConfig() *Config {
	if instance == nil {
		lock.Lock()
		defer lock.Unlock()
		if instance == nil {
			if err := cfg.LoadConfigWithDefaults("proxy", &instance, defaults); err != nil {
				log.Fatalf("error reading config file: %s\n", err)
			}
		}
	}

	log.Tracef("config: %+v", instance)

	return instance
}
