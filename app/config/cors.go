package config

import (
	"github.com/gofiber/fiber/v2/middleware/cors"
)

type corsConfig struct {
	AllowCredentials bool   `mapstructure:"allow-credentials"`
	AllowOrigins     string `mapstructure:"allow-origins"`
	AllowHeaders     string `mapstructure:"allow-headers"`
	AllowMethods     string `mapstructure:"allow-methods"`
}

func (c *corsConfig) ToCorsConfig() cors.Config {
	return cors.Config{
		AllowCredentials: c.AllowCredentials,
		AllowOrigins:     c.AllowOrigins,
		AllowHeaders:     c.AllowHeaders,
		AllowMethods:     c.AllowMethods,
	}
}
