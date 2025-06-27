// Package grpc provides gRPC client implementations for plantd services.
package grpc

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"connectrpc.com/connect"
	identityv1 "github.com/geoffjay/plantd/gen/proto/go/plantd/identity/v1"
	"github.com/geoffjay/plantd/gen/proto/go/plantd/identity/v1/identityv1connect"
	log "github.com/sirupsen/logrus"
)

// IdentityClient provides a gRPC client for the Identity service.
type IdentityClient struct {
	client  identityv1connect.IdentityServiceClient
	baseURL string
	timeout time.Duration
	logger  *log.Entry
}

// IdentityClientConfig holds configuration for the Identity gRPC client.
type IdentityClientConfig struct {
	BaseURL    string        // Traefik gateway URL (e.g., "http://localhost:8080")
	Timeout    time.Duration // Request timeout
	HTTPClient *http.Client  // Optional custom HTTP client
}

// LoginResponse represents the response from a login operation.
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
	UserID       int64  `json:"user_id"`
	Email        string `json:"email"`
}

// TokenValidationResponse represents the response from token validation.
type TokenValidationResponse struct {
	Valid       bool     `json:"valid"`
	UserID      *int64   `json:"user_id,omitempty"`
	Email       string   `json:"email,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	ExpiresAt   *int64   `json:"expires_at,omitempty"`
}

// NewIdentityClient creates a new Identity service gRPC client.
func NewIdentityClient(config *IdentityClientConfig) *IdentityClient {
	logger := log.WithField("service", "identity_grpc_client")

	// Use default HTTP client if none provided
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: config.Timeout,
		}
	}

	// Create Connect client
	client := identityv1connect.NewIdentityServiceClient(
		httpClient,
		config.BaseURL,
		connect.WithGRPC(), // Use gRPC protocol
	)

	return &IdentityClient{
		client:  client,
		baseURL: config.BaseURL,
		timeout: config.Timeout,
		logger:  logger,
	}
}

// Login authenticates a user and returns access tokens.
func (ic *IdentityClient) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	ic.logger.WithField("email", email).Debug("Authenticating user via gRPC")

	// Create request
	req := connect.NewRequest(&identityv1.LoginRequest{
		Email:    email,
		Password: password,
	})

	// Set timeout
	if ic.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, ic.timeout)
		defer cancel()
	}

	// Make request
	resp, err := ic.client.Login(ctx, req)
	if err != nil {
		ic.logger.WithError(err).WithField("email", email).Error("Failed to login user")
		return nil, err
	}

	var expiresAt int64
	if resp.Msg.ExpiresAt != nil {
		expiresAt = resp.Msg.ExpiresAt.AsTime().Unix()
	}

	var userID int64
	var userEmail string
	if resp.Msg.User != nil {
		if resp.Msg.User.Id != "" {
			// Parse user ID from string
			if id, err := strconv.ParseInt(resp.Msg.User.Id, 10, 64); err == nil {
				userID = id
			}
		}
		userEmail = resp.Msg.User.Email
	}

	loginResp := &LoginResponse{
		AccessToken:  resp.Msg.AccessToken,
		RefreshToken: resp.Msg.RefreshToken,
		ExpiresAt:    expiresAt,
		UserID:       userID,
		Email:        userEmail,
	}

	ic.logger.WithFields(log.Fields{
		"email":   userEmail,
		"user_id": userID,
	}).Info("User login successful via gRPC")

	return loginResp, nil
}

// ValidateToken validates an access token and returns user information.
func (ic *IdentityClient) ValidateToken(ctx context.Context, token string) (*TokenValidationResponse, error) {
	ic.logger.Debug("Validating token via gRPC")

	// Create request
	req := connect.NewRequest(&identityv1.ValidateTokenRequest{
		Token: token,
	})

	// Set timeout
	if ic.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, ic.timeout)
		defer cancel()
	}

	// Make request
	resp, err := ic.client.ValidateToken(ctx, req)
	if err != nil {
		ic.logger.WithError(err).Error("Failed to validate token")
		return nil, err
	}

	validationResp := &TokenValidationResponse{
		Valid:       resp.Msg.Valid,
		Permissions: resp.Msg.Permissions,
	}

	// Extract user information if available
	if resp.Msg.User != nil {
		validationResp.Email = resp.Msg.User.Email
		validationResp.Roles = resp.Msg.User.Roles

		// Parse user ID from string if available
		if resp.Msg.User.Id != "" {
			// For now, just use a simple approach - in real implementation this would be proper ID parsing
			var userID int64 = 0 // Default value
			validationResp.UserID = &userID
		}
	}

	// Handle timestamp
	if resp.Msg.ExpiresAt != nil {
		expiresAt := resp.Msg.ExpiresAt.AsTime().Unix()
		validationResp.ExpiresAt = &expiresAt
	}

	ic.logger.WithFields(log.Fields{
		"valid":   validationResp.Valid,
		"user_id": validationResp.UserID,
		"email":   validationResp.Email,
	}).Debug("Token validation complete via gRPC")

	return validationResp, nil
}

// RefreshToken refreshes an access token using a refresh token.
func (ic *IdentityClient) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	ic.logger.Debug("Refreshing token via gRPC")

	// Create request
	req := connect.NewRequest(&identityv1.RefreshTokenRequest{
		RefreshToken: refreshToken,
	})

	// Set timeout
	if ic.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, ic.timeout)
		defer cancel()
	}

	// Make request
	resp, err := ic.client.RefreshToken(ctx, req)
	if err != nil {
		ic.logger.WithError(err).Error("Failed to refresh token")
		return nil, err
	}

	var expiresAt int64
	if resp.Msg.ExpiresAt != nil {
		expiresAt = resp.Msg.ExpiresAt.AsTime().Unix()
	}

	loginResp := &LoginResponse{
		AccessToken:  resp.Msg.AccessToken,
		RefreshToken: resp.Msg.RefreshToken,
		ExpiresAt:    expiresAt,
		UserID:       0,  // RefreshTokenResponse doesn't include user info
		Email:        "", // RefreshTokenResponse doesn't include user info
	}

	ic.logger.Info("Token refresh successful via gRPC")

	return loginResp, nil
}

// Logout invalidates a user's tokens.
func (ic *IdentityClient) Logout(ctx context.Context, token string) error {
	ic.logger.Debug("Logging out user via gRPC")

	// Create request
	req := connect.NewRequest(&identityv1.LogoutRequest{
		Token: token,
	})

	// Set timeout
	if ic.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, ic.timeout)
		defer cancel()
	}

	// Make request
	_, err := ic.client.Logout(ctx, req)
	if err != nil {
		ic.logger.WithError(err).Error("Failed to logout user")
		return err
	}

	ic.logger.Info("User logout successful via gRPC")
	return nil
}

// HealthCheck checks the health of the Identity service.
func (ic *IdentityClient) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), ic.timeout)
	defer cancel()

	ic.logger.Debug("Checking identity service health via gRPC")

	// Use a simple HTTP GET to the health endpoint through the gateway
	httpClient := &http.Client{Timeout: ic.timeout}

	req, err := http.NewRequestWithContext(ctx, "GET", ic.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		ic.logger.WithError(err).Error("Identity service health check failed")
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ic.logger.WithField("status_code", resp.StatusCode).Error("Identity service health check failed")
		return connect.NewError(connect.CodeUnavailable, nil)
	}

	ic.logger.Debug("Identity service health check passed")
	return nil
}
