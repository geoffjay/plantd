package config

import (
	"runtime"
	"time"

	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/storage/redis/v3"
	log "github.com/sirupsen/logrus"
)

type sessionConfig struct {
	Expiration     string `mapstructure:"expiration"`
	KeyLookup      string `mapstructure:"key-lookup"`
	CookieSecure   bool   `mapstructure:"cookie-secure"`
	CookieHTTPOnly bool   `mapstructure:"cookie-http-only"`
	CookieSameSite string `mapstructure:"cookie-same-site"`

	// Additional fields for enhanced session management
	CookieName string `mapstructure:"cookie-name"`
	MaxAge     int    `mapstructure:"max-age"`
	Secure     bool   `mapstructure:"secure"`
	HTTPOnly   bool   `mapstructure:"http-only"`
}

func (c *sessionConfig) ToSessionConfig() session.Config {
	storage := redis.New(redis.Config{
		Host:      "127.0.0.1",
		Port:      6379,
		Username:  "",
		Password:  "",
		Database:  0,
		Reset:     false,
		TLSConfig: nil,
		PoolSize:  10 * runtime.GOMAXPROCS(0),
	})

	expiration, err := time.ParseDuration(c.Expiration)
	if err != nil {
		log.Fatal(err)
	}

	return session.Config{
		Storage:        storage,
		Expiration:     expiration,
		KeyLookup:      c.KeyLookup,
		CookieSecure:   c.CookieSecure,
		CookieHTTPOnly: c.CookieHTTPOnly,
		CookieSameSite: c.CookieSameSite,
	}
}
