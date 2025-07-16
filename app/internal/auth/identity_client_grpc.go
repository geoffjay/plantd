// Package auth provides authentication and authorization functionality.
package auth

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"github.com/geoffjay/plantd/app/config"
	healthv1 "github.com/geoffjay/plantd/gen/proto/go/plantd/health/v1"
	identityv1 "github.com/geoffjay/plantd/gen/proto/go/plantd/identity/v1"
	"github.com/geoffjay/plantd/gen/proto/go/plantd/identity/v1/identityv1connect"
	"google.golang.org/protobuf/types/known/emptypb"

	log "github.com/sirupsen/logrus"
)

// GRPCIdentityClient provides real gRPC-based Identity Service integration.
type GRPCIdentityClient struct {
	client    identityv1connect.IdentityServiceClient
	config    *config.Config
	baseURL   string
	timeout   time.Duration
	logger    *log.Entry
	isHealthy bool
	lastCheck time.Time
}

// NewGRPCIdentityClient creates a new gRPC-based Identity Service client.
func NewGRPCIdentityClient(cfg *config.Config) (*GRPCIdentityClient, error) {
	fields := log.Fields{
		"service": "app",
		"context": "grpc_identity_client.new",
	}

	// Determine base URL for Traefik gateway
	baseURL := "http://localhost:8080" // Default Traefik gateway
	if cfg.Identity.Endpoint != "" {
		// Parse endpoint and convert to HTTP URL
		// For now, assume Traefik is on localhost:8080
		baseURL = "http://localhost:8080"
	}

	// Parse timeout
	timeout, err := time.ParseDuration(cfg.Identity.Timeout)
	if err != nil {
		timeout = 30 * time.Second
		log.WithError(err).Warn("Failed to parse identity timeout, using default 30s")
	}

	// Create HTTP client for Connect
	httpClient := &http.Client{
		Timeout: timeout,
	}

	// Create Connect client
	client := identityv1connect.NewIdentityServiceClient(
		httpClient,
		baseURL,
		connect.WithGRPC(), // Use gRPC protocol
	)

	ic := &GRPCIdentityClient{
		client:    client,
		config:    cfg,
		baseURL:   baseURL,
		timeout:   timeout,
		logger:    log.WithField("service", "grpc_identity_client"),
		isHealthy: false,
		lastCheck: time.Time{},
	}

	// Perform initial health check
	go ic.performHealthCheck()

	log.WithFields(fields).WithFields(log.Fields{
		"base_url": baseURL,
		"timeout":  timeout,
	}).Info("gRPC Identity Service client initialized")

	return ic, nil
}

// Close closes the Identity Service client connection.
func (ic *GRPCIdentityClient) Close() error {
	// HTTP client doesn't require explicit closing
	return nil
}

// ValidateToken validates a JWT token with the Identity Service.
func (ic *GRPCIdentityClient) ValidateToken(token string) (*UserContext, error) {
	fields := log.Fields{
		"service": "app",
		"context": "grpc_identity_client.validate_token",
	}

	if !ic.isAvailable() {
		return nil, fmt.Errorf("identity service unavailable")
	}

	ctx, cancel := context.WithTimeout(context.Background(), ic.timeout)
	defer cancel()

	// Create request
	req := connect.NewRequest(&identityv1.ValidateTokenRequest{
		Token: token,
	})

	// Make request
	resp, err := ic.client.ValidateToken(ctx, req)
	if err != nil {
		ic.logger.WithFields(fields).WithError(err).Error("Token validation failed")
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	// Parse response
	if !resp.Msg.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	userContext := &UserContext{
		ID:            parseUserID(resp.Msg.User.Id),
		Email:         resp.Msg.User.Email,
		Username:      resp.Msg.User.Name, // Use name as username
		Roles:         resp.Msg.User.Roles,
		Organizations: []string{resp.Msg.User.OrganizationId}, // Add organization from user
		Permissions:   resp.Msg.Permissions,
	}

	ic.logger.WithFields(fields).Debug("Token validation successful")
	return userContext, nil
}

// Login authenticates a user with email and password.
func (ic *GRPCIdentityClient) Login(email, password string) (*TokenPair, *UserContext, error) {
	fields := log.Fields{
		"service": "app",
		"context": "grpc_identity_client.login",
		"email":   email,
	}

	if !ic.isAvailable() {
		return nil, nil, fmt.Errorf("identity service unavailable")
	}

	ctx, cancel := context.WithTimeout(context.Background(), ic.timeout)
	defer cancel()

	// Create request
	req := connect.NewRequest(&identityv1.LoginRequest{
		Email:    email,
		Password: password,
	})

	// Make request
	resp, err := ic.client.Login(ctx, req)
	if err != nil {
		ic.logger.WithFields(fields).WithError(err).Warn("Login failed")
		return nil, nil, fmt.Errorf("login failed: %w", err)
	}

	// Parse tokens
	var expiresAt time.Time
	if resp.Msg.ExpiresAt != nil {
		expiresAt = resp.Msg.ExpiresAt.AsTime()
	}

	tokenPair := &TokenPair{
		AccessToken:  resp.Msg.AccessToken,
		RefreshToken: resp.Msg.RefreshToken,
		ExpiresAt:    expiresAt,
		TokenType:    "Bearer",
	}

	// Parse user information
	var userContext *UserContext
	if resp.Msg.User != nil {
		userContext = &UserContext{
			ID:            parseUserID(resp.Msg.User.Id),
			Email:         resp.Msg.User.Email,
			Username:      resp.Msg.User.Email, // Use email as username
			Roles:         []string{},          // TODO: Get roles from user response
			Organizations: []string{},          // TODO: Add organization support
			Permissions:   []string{},          // TODO: Get permissions from user response
		}
	}

	ic.logger.WithFields(fields).WithField("user_id", userContext.ID).Info("User login successful")

	return tokenPair, userContext, nil
}

// RefreshToken refreshes an access token using a refresh token.
func (ic *GRPCIdentityClient) RefreshToken(refreshToken string) (*TokenPair, error) {
	fields := log.Fields{
		"service": "app",
		"context": "grpc_identity_client.refresh_token",
	}

	if !ic.isAvailable() {
		return nil, fmt.Errorf("identity service unavailable")
	}

	ctx, cancel := context.WithTimeout(context.Background(), ic.timeout)
	defer cancel()

	// Create request
	req := connect.NewRequest(&identityv1.RefreshTokenRequest{
		RefreshToken: refreshToken,
	})

	// Make request
	resp, err := ic.client.RefreshToken(ctx, req)
	if err != nil {
		ic.logger.WithFields(fields).WithError(err).Error("Token refresh failed")
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}

	// Parse response
	var expiresAt time.Time
	if resp.Msg.ExpiresAt != nil {
		expiresAt = resp.Msg.ExpiresAt.AsTime()
	}

	tokenPair := &TokenPair{
		AccessToken:  resp.Msg.AccessToken,
		RefreshToken: refreshToken, // Keep original refresh token
		ExpiresAt:    expiresAt,
		TokenType:    "Bearer",
	}

	ic.logger.WithFields(fields).Debug("Token refresh successful")

	return tokenPair, nil
}

// Logout invalidates a user's session.
func (ic *GRPCIdentityClient) Logout(accessToken string) error {
	fields := log.Fields{
		"service": "app",
		"context": "grpc_identity_client.logout",
	}

	if !ic.isAvailable() {
		ic.logger.WithFields(fields).Debug("Identity service unavailable for logout")
		return nil // Don't fail logout if service is unavailable
	}

	ctx, cancel := context.WithTimeout(context.Background(), ic.timeout)
	defer cancel()

	// Create request
	req := connect.NewRequest(&identityv1.LogoutRequest{
		Token: accessToken,
	})

	// Make request
	_, err := ic.client.Logout(ctx, req)
	if err != nil {
		ic.logger.WithFields(fields).WithError(err).Warn("Logout request failed")
		return fmt.Errorf("logout failed: %w", err)
	}

	ic.logger.WithFields(fields).Debug("User logout successful")

	return nil
}

// HealthCheck performs a health check against the Identity Service.
func (ic *GRPCIdentityClient) HealthCheck() error {
	fields := log.Fields{
		"service": "app",
		"context": "grpc_identity_client.health_check",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create request (Health method takes empty request)
	req := connect.NewRequest(&emptypb.Empty{})

	// Make request
	resp, err := ic.client.Health(ctx, req)
	if err != nil {
		ic.logger.WithFields(fields).WithError(err).Debug("Identity service health check failed")
		ic.markUnhealthy()
		return fmt.Errorf("health check failed: %w", err)
	}

	if resp.Msg.Status != healthv1.HealthCheckResponse_SERVING_STATUS_SERVING {
		ic.logger.WithFields(fields).Debug("Identity service is not serving")
		ic.markUnhealthy()
		return fmt.Errorf("service is not serving")
	}

	ic.markHealthy()
	ic.logger.WithFields(fields).Debug("Identity service health check successful")

	return nil
}

// IsHealthy returns whether the Identity Service is currently healthy.
func (ic *GRPCIdentityClient) IsHealthy() bool {
	// Perform health check if we haven't checked recently
	if time.Since(ic.lastCheck) > 30*time.Second {
		go ic.performHealthCheck()
	}

	return ic.isHealthy
}

// isAvailable checks if the Identity Service is available for requests.
func (ic *GRPCIdentityClient) isAvailable() bool {
	return ic.IsHealthy()
}

// markHealthy marks the Identity Service as healthy.
func (ic *GRPCIdentityClient) markHealthy() {
	ic.isHealthy = true
	ic.lastCheck = time.Now()
}

// markUnhealthy marks the Identity Service as unhealthy.
func (ic *GRPCIdentityClient) markUnhealthy() {
	ic.isHealthy = false
	ic.lastCheck = time.Now()
}

// performHealthCheck performs a background health check.
func (ic *GRPCIdentityClient) performHealthCheck() {
	if err := ic.HealthCheck(); err != nil {
		ic.logger.WithError(err).Debug("Background health check failed")
	}
}

// parseUserID parses a user ID string to uint.
func parseUserID(userID string) uint {
	if id, err := strconv.ParseUint(userID, 10, 32); err == nil {
		return uint(id)
	}
	return 0
}
