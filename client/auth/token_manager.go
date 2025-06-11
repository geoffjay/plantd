// Package auth provides authentication token management for the plantd client.
package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Common errors
var (
	ErrNotAuthenticated = errors.New("authentication required")
	ErrTokenExpired     = errors.New("token expired")
	ErrInvalidProfile   = errors.New("invalid profile")
)

// TokenProfile represents stored authentication information for a profile.
type TokenProfile struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
	UserEmail    string `json:"user_email"`
	Endpoint     string `json:"identity_endpoint"`
}

// TokenManager manages authentication tokens and profiles.
type TokenManager struct {
	configPath string
	profiles   map[string]*TokenProfile
}

// TokenStorage represents the JSON structure for stored tokens.
type TokenStorage struct {
	Profiles map[string]*TokenProfile `json:"profiles"`
}

// NewTokenManager creates a new token manager instance.
func NewTokenManager() *TokenManager {
	configDir := getConfigDir()
	configPath := filepath.Join(configDir, "tokens.json")

	return &TokenManager{
		configPath: configPath,
		profiles:   make(map[string]*TokenProfile),
	}
}

// GetValidToken returns a valid access token for the specified profile.
// If the token is expired, it returns ErrTokenExpired.
// If no authentication exists, it returns ErrNotAuthenticated.
func (tm *TokenManager) GetValidToken(profile string) (string, error) {
	tokenProfile, err := tm.GetProfile(profile)
	if err != nil {
		return "", ErrNotAuthenticated
	}

	// Check if token is expired
	if time.Now().Unix() >= tokenProfile.ExpiresAt {
		return "", ErrTokenExpired
	}

	return tokenProfile.AccessToken, nil
}

// GetProfile returns the token profile for the specified profile name.
func (tm *TokenManager) GetProfile(profile string) (*TokenProfile, error) {
	if err := tm.loadTokens(); err != nil {
		return nil, err
	}

	tokenProfile, exists := tm.profiles[profile]
	if !exists {
		return nil, ErrInvalidProfile
	}

	return tokenProfile, nil
}

// StoreTokens stores authentication tokens for the specified profile.
func (tm *TokenManager) StoreTokens(profile string, tokenProfile *TokenProfile) error {
	if err := tm.loadTokens(); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load existing tokens: %w", err)
	}

	tm.profiles[profile] = tokenProfile
	return tm.saveTokens()
}

// ClearTokens removes authentication tokens for the specified profile.
func (tm *TokenManager) ClearTokens(profile string) error {
	if err := tm.loadTokens(); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to load existing tokens: %w", err)
	}

	delete(tm.profiles, profile)
	return tm.saveTokens()
}

// RefreshToken attempts to refresh the access token for the specified profile.
// This method does not actually perform the refresh - it should be called
// after a successful refresh to update the stored tokens.
func (tm *TokenManager) RefreshToken(profile string) (string, error) {
	tokenProfile, err := tm.GetProfile(profile)
	if err != nil {
		return "", err
	}

	// Return the refresh token for the caller to use
	return tokenProfile.RefreshToken, nil
}

// ListProfiles returns a list of available authentication profiles.
func (tm *TokenManager) ListProfiles() ([]string, error) {
	if err := tm.loadTokens(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	profiles := make([]string, 0, len(tm.profiles))
	for profile := range tm.profiles {
		profiles = append(profiles, profile)
	}

	return profiles, nil
}

// loadTokens loads tokens from the encrypted file.
func (tm *TokenManager) loadTokens() error {
	// Ensure config directory exists
	if err := os.MkdirAll(filepath.Dir(tm.configPath), 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(tm.configPath); os.IsNotExist(err) {
		return err
	}

	// Read encrypted file
	encryptedData, err := os.ReadFile(tm.configPath)
	if err != nil {
		return fmt.Errorf("failed to read token file: %w", err)
	}

	// For now, we'll store tokens as plain JSON with secure file permissions
	// TODO: Implement proper encryption in future iterations
	var storage TokenStorage
	if err := json.Unmarshal(encryptedData, &storage); err != nil {
		return fmt.Errorf("failed to parse token data: %w", err)
	}

	if storage.Profiles == nil {
		storage.Profiles = make(map[string]*TokenProfile)
	}

	tm.profiles = storage.Profiles
	return nil
}

// saveTokens saves tokens to the encrypted file.
func (tm *TokenManager) saveTokens() error {
	// Ensure config directory exists
	configDir := filepath.Dir(tm.configPath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Prepare storage structure
	storage := TokenStorage{
		Profiles: tm.profiles,
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(storage, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token data: %w", err)
	}

	// For now, write as plain JSON with secure file permissions
	// TODO: Implement proper encryption in future iterations
	if err := os.WriteFile(tm.configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// getConfigDir returns the configuration directory path.
func getConfigDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return ".config/plantd"
	}
	return filepath.Join(homeDir, ".config", "plantd")
}

// Utility methods for TokenProfile

// ExpiresAtFormatted returns a formatted expiration time string.
func (tp *TokenProfile) ExpiresAtFormatted() string {
	return FormatUnixTimestamp(tp.ExpiresAt)
}

// TokenStatus returns a human-readable status of the token.
func (tp *TokenProfile) TokenStatus() string {
	now := time.Now().Unix()
	if now >= tp.ExpiresAt {
		return "Expired"
	}

	timeLeft := tp.ExpiresAt - now
	if timeLeft < 300 { // Less than 5 minutes
		return "Expires Soon"
	}

	return "Valid"
}

// IsExpired returns true if the token is expired.
func (tp *TokenProfile) IsExpired() bool {
	return time.Now().Unix() >= tp.ExpiresAt
}

// FormatUnixTimestamp formats a Unix timestamp to a human-readable string.
func FormatUnixTimestamp(timestamp int64) string {
	return time.Unix(timestamp, 0).Format("2006-01-02 15:04:05 MST")
}

// Future encryption implementation placeholder
// These functions are prepared for future implementation of token encryption

func encrypt(data []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

func decrypt(encryptedData []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(encryptedData) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := encryptedData[:nonceSize], encryptedData[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
