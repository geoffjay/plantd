package config

import (
	"time"

	"github.com/geoffjay/plantd/identity/internal/auth"
)

// ToPasswordConfig converts the security config to a password config.
func (c *Config) ToPasswordConfig() *auth.PasswordConfig {
	return &auth.PasswordConfig{
		MinLength:           c.Security.PasswordMinLength,
		MaxLength:           c.Security.PasswordMaxLength,
		RequireUppercase:    c.Security.RequireUppercase,
		RequireLowercase:    c.Security.RequireLowercase,
		RequireNumbers:      c.Security.RequireNumbers,
		RequireSpecialChars: c.Security.RequireSpecialChars,
		BcryptCost:          c.Security.BcryptCost,
	}
}

// ToJWTConfig converts the security config to a JWT config.
func (c *Config) ToJWTConfig() *auth.JWTConfig {
	return &auth.JWTConfig{
		AccessTokenSecret:  c.Security.JWTSecret,
		RefreshTokenSecret: c.Security.JWTRefreshSecret,
		AccessTokenExpiry:  time.Duration(c.Security.JWTExpiration) * time.Second,
		RefreshTokenExpiry: time.Duration(c.Security.RefreshExpiration) * time.Second,
		Issuer:             c.Security.JWTIssuer,
	}
}

// ToRateLimiterConfig converts the security config to a rate limiter config.
func (c *Config) ToRateLimiterConfig() *auth.RateLimiterConfig {
	return &auth.RateLimiterConfig{
		RequestsPerMinute: c.Security.RateLimitRPS,
		BurstSize:         c.Security.RateLimitBurst,
		BlockDuration:     5 * time.Minute, // Fixed for now
		MaxFailedAttempts: c.Security.MaxFailedAttempts,
		LockoutDuration:   time.Duration(c.Security.LockoutDurationMinutes) * time.Minute,
	}
}

// ToAuthConfig converts the security config to a full auth config.
func (c *Config) ToAuthConfig() *auth.AuthConfig {
	return &auth.AuthConfig{
		Password:       c.ToPasswordConfig(),
		JWT:            c.ToJWTConfig(),
		RateLimit:      c.ToRateLimiterConfig(),
		SessionTimeout: 24 * time.Hour, // Fixed for now
	}
}

// ToRegistrationConfig converts the security config to a registration config.
func (c *Config) ToRegistrationConfig() *auth.RegistrationConfig {
	return &auth.RegistrationConfig{
		RequireEmailVerification: c.Security.RequireEmailVerification,
		AllowSelfRegistration:    c.Security.AllowSelfRegistration,
		EmailVerificationExpiry:  time.Duration(c.Security.EmailVerificationExpireyHours) * time.Hour,
		PasswordResetExpiry:      time.Duration(c.Security.PasswordResetExpiryHours) * time.Hour,
		DefaultUserRole:          "user",
	}
}
