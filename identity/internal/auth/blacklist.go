package auth

import (
	"sync"
	"time"
)

// BlacklistedToken represents a blacklisted token with its expiry time.
type BlacklistedToken struct {
	TokenID   string    `json:"token_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// InMemoryBlacklist implements TokenBlacklistService using in-memory storage.
type InMemoryBlacklist struct {
	mu     sync.RWMutex
	tokens map[string]time.Time
}

// NewInMemoryBlacklist creates a new in-memory blacklist service.
func NewInMemoryBlacklist() *InMemoryBlacklist {
	return &InMemoryBlacklist{
		tokens: make(map[string]time.Time),
	}
}

// BlacklistToken adds a token to the blacklist with its expiry time.
func (b *InMemoryBlacklist) BlacklistToken(tokenID string, expiry time.Time) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.tokens[tokenID] = expiry
	return nil
}

// IsTokenBlacklisted checks if a token is in the blacklist.
func (b *InMemoryBlacklist) IsTokenBlacklisted(tokenID string) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	expiry, exists := b.tokens[tokenID]
	if !exists {
		return false, nil
	}

	// If token has expired, remove it and return false
	if time.Now().After(expiry) {
		delete(b.tokens, tokenID)
		return false, nil
	}

	return true, nil
}

// CleanupExpiredTokens removes expired tokens from the blacklist.
func (b *InMemoryBlacklist) CleanupExpiredTokens() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	for tokenID, expiry := range b.tokens {
		if now.After(expiry) {
			delete(b.tokens, tokenID)
		}
	}

	return nil
}

// GetBlacklistSize returns the current size of the blacklist (for monitoring).
func (b *InMemoryBlacklist) GetBlacklistSize() int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return len(b.tokens)
}

// Clear removes all tokens from the blacklist (useful for testing).
func (b *InMemoryBlacklist) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.tokens = make(map[string]time.Time)
}
