package auth

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiterConfig holds configuration for rate limiting
type RateLimiterConfig struct {
	// RequestsPerMinute is the number of requests allowed per minute
	RequestsPerMinute int `json:"requests_per_minute" yaml:"requests_per_minute"`
	// BurstSize is the maximum number of requests that can be made in a burst
	BurstSize int `json:"burst_size" yaml:"burst_size"`
	// BlockDuration is how long to block after exceeding limits
	BlockDuration time.Duration `json:"block_duration" yaml:"block_duration"`
	// MaxFailedAttempts is the number of failed attempts before account lockout
	MaxFailedAttempts int `json:"max_failed_attempts" yaml:"max_failed_attempts"`
	// LockoutDuration is how long to lock an account after too many failed attempts
	LockoutDuration time.Duration `json:"lockout_duration" yaml:"lockout_duration"`
}

// DefaultRateLimiterConfig returns a secure default rate limiter configuration
func DefaultRateLimiterConfig() *RateLimiterConfig {
	return &RateLimiterConfig{
		RequestsPerMinute: 10,
		BurstSize:         5,
		BlockDuration:     5 * time.Minute,
		MaxFailedAttempts: 5,
		LockoutDuration:   15 * time.Minute,
	}
}

// ClientLimiter tracks rate limiting for a specific client (IP address)
type ClientLimiter struct {
	limiter      *rate.Limiter
	lastSeen     time.Time
	blockedUntil time.Time
}

// AccountLockout tracks failed login attempts for user accounts
type AccountLockout struct {
	FailedAttempts int       `json:"failed_attempts"`
	LastAttempt    time.Time `json:"last_attempt"`
	LockedUntil    time.Time `json:"locked_until"`
}

// RateLimiter provides rate limiting functionality for authentication
type RateLimiter struct {
	config        *RateLimiterConfig
	clients       map[string]*ClientLimiter
	accountLocks  map[string]*AccountLockout
	mu            sync.RWMutex
	cleanupTicker *time.Ticker
	stopCleanup   chan bool
}

// NewRateLimiter creates a new rate limiter with the given configuration
func NewRateLimiter(config *RateLimiterConfig) *RateLimiter {
	if config == nil {
		config = DefaultRateLimiterConfig()
	}

	rl := &RateLimiter{
		config:       config,
		clients:      make(map[string]*ClientLimiter),
		accountLocks: make(map[string]*AccountLockout),
		stopCleanup:  make(chan bool),
	}

	// Start cleanup goroutine to remove old entries
	rl.cleanupTicker = time.NewTicker(5 * time.Minute)
	go rl.cleanupLoop()

	return rl
}

// AllowRequest checks if a request from the given IP address should be allowed
func (rl *RateLimiter) AllowRequest(ipAddress string) (bool, error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Get or create client limiter
	client, exists := rl.clients[ipAddress]
	if !exists {
		// Create new rate limiter for this client
		limit := rate.Every(time.Minute / time.Duration(rl.config.RequestsPerMinute))
		client = &ClientLimiter{
			limiter:  rate.NewLimiter(limit, rl.config.BurstSize),
			lastSeen: now,
		}
		rl.clients[ipAddress] = client
	}

	client.lastSeen = now

	// Check if client is currently blocked
	if now.Before(client.blockedUntil) {
		return false, fmt.Errorf("client blocked until %v", client.blockedUntil)
	}

	// Check rate limit
	if !client.limiter.Allow() {
		// Block the client for the configured duration
		client.blockedUntil = now.Add(rl.config.BlockDuration)
		return false, fmt.Errorf("rate limit exceeded, blocked for %v", rl.config.BlockDuration)
	}

	return true, nil
}

// RecordFailedLogin records a failed login attempt for account lockout tracking
func (rl *RateLimiter) RecordFailedLogin(identifier string) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	lockout, exists := rl.accountLocks[identifier]
	if !exists {
		lockout = &AccountLockout{
			FailedAttempts: 0,
			LastAttempt:    now,
		}
		rl.accountLocks[identifier] = lockout
	}

	// Reset failed attempts if enough time has passed
	if now.Sub(lockout.LastAttempt) > rl.config.LockoutDuration {
		lockout.FailedAttempts = 0
	}

	lockout.FailedAttempts++
	lockout.LastAttempt = now

	// Check if account should be locked
	if lockout.FailedAttempts >= rl.config.MaxFailedAttempts {
		lockout.LockedUntil = now.Add(rl.config.LockoutDuration)
	}

	return nil
}

// IsAccountLocked checks if an account is currently locked due to failed attempts
func (rl *RateLimiter) IsAccountLocked(identifier string) (bool, time.Time, error) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	lockout, exists := rl.accountLocks[identifier]
	if !exists {
		return false, time.Time{}, nil
	}

	now := time.Now()

	// Check if lockout has expired
	if now.After(lockout.LockedUntil) {
		return false, time.Time{}, nil
	}

	return lockout.FailedAttempts >= rl.config.MaxFailedAttempts, lockout.LockedUntil, nil
}

// RecordSuccessfulLogin clears failed login attempts for an account
func (rl *RateLimiter) RecordSuccessfulLogin(identifier string) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Clear failed attempts on successful login
	delete(rl.accountLocks, identifier)
	return nil
}

// UnlockAccount manually unlocks an account (admin function)
func (rl *RateLimiter) UnlockAccount(identifier string) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	delete(rl.accountLocks, identifier)
	return nil
}

// GetAccountLockoutInfo returns information about an account's lockout status
func (rl *RateLimiter) GetAccountLockoutInfo(identifier string) (*AccountLockout, bool) {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	lockout, exists := rl.accountLocks[identifier]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	return &AccountLockout{
		FailedAttempts: lockout.FailedAttempts,
		LastAttempt:    lockout.LastAttempt,
		LockedUntil:    lockout.LockedUntil,
	}, true
}

// GetStats returns statistics about the rate limiter
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	blockedClients := 0
	lockedAccounts := 0
	now := time.Now()

	for _, client := range rl.clients {
		if now.Before(client.blockedUntil) {
			blockedClients++
		}
	}

	for _, lockout := range rl.accountLocks {
		if lockout.FailedAttempts >= rl.config.MaxFailedAttempts && now.Before(lockout.LockedUntil) {
			lockedAccounts++
		}
	}

	return map[string]interface{}{
		"total_clients":   len(rl.clients),
		"blocked_clients": blockedClients,
		"total_accounts":  len(rl.accountLocks),
		"locked_accounts": lockedAccounts,
		"config":          rl.config,
	}
}

// cleanupLoop periodically removes old entries from the rate limiter
func (rl *RateLimiter) cleanupLoop() {
	for {
		select {
		case <-rl.cleanupTicker.C:
			rl.cleanup()
		case <-rl.stopCleanup:
			rl.cleanupTicker.Stop()
			return
		}
	}
}

// cleanup removes old entries from clients and account locks
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cleanupThreshold := 30 * time.Minute

	// Cleanup old client entries
	for ip, client := range rl.clients {
		if now.Sub(client.lastSeen) > cleanupThreshold && now.After(client.blockedUntil) {
			delete(rl.clients, ip)
		}
	}

	// Cleanup old account lockout entries
	for identifier, lockout := range rl.accountLocks {
		if now.Sub(lockout.LastAttempt) > rl.config.LockoutDuration*2 && now.After(lockout.LockedUntil) {
			delete(rl.accountLocks, identifier)
		}
	}
}

// Stop stops the rate limiter cleanup goroutine
func (rl *RateLimiter) Stop() {
	close(rl.stopCleanup)
}
