package config

import (
	"sync"

	cfg "github.com/geoffjay/plantd/core/config"

	log "github.com/sirupsen/logrus"
)

// TODO:
// - add a new configuration section for the database

type Config struct {
	cfg.Config

	Env            string        `mapstructure:"env"`
	ClientEndpoint string        `mapstructure:"client-endpoint"`
	Log            cfg.LogConfig `mapstructure:"log"`
	Cors           corsConfig    `mapstructure:"cors"`
	Session        sessionConfig `mapstructure:"session"`
}

var lock = &sync.Mutex{}
var instance *Config

var defaults = map[string]interface{}{
	"env":                      "development",
	"client-endpoint":          "tcp://localhost:9797",
	"log.formatter":            "text",
	"log.level":                "info",
	"log.loki.address":         "http://localhost:3100",
	"log.loki.labels":          map[string]string{"app": "app", "environment": "development"},
	"cors.allow-credentials":   true,
	"cors.allow-origins":       "*",
	"cors.allow-headers":       "Origin, Content-Type, Accept, Content-Length, Accept-Language, Accept-Encoding, Connection, Authorization, Access-Control-Allow-Origin, Access-Control-Allow-Methods, Access-Control-Allow-Headers, Access-Control-Allow-Origin",
	"cors.allow-methods":       "GET, POST, HEAD, PUT, DELETE, PATCH, OPTIONS",
	"session.expiration":       "2h",
	"session.key-lookup":       "cookie:__Host-session",
	"session.cookie-secure":    true,
	"session.cookie-http-only": true,
	"session.cookie-same-site": "Lax",
}

// GetConfig returns the application configuration singleton.
func GetConfig() *Config {
	if instance == nil {
		lock.Lock()
		defer lock.Unlock()
		if instance == nil {
			if err := cfg.LoadConfigWithDefaults("app", &instance, defaults); err != nil {
				log.Fatalf("error reading config file: %s\n", err)
			}
		}
	}

	log.Tracef("config: %+v", instance)

	return instance
}
