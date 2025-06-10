// Package client provides a client library for the PlantD Identity Service.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/geoffjay/plantd/core/mdp"
	"github.com/geoffjay/plantd/identity/internal/handlers"
	"github.com/sirupsen/logrus"
)

// Client represents the identity service client.
type Client struct {
	mdpClient *mdp.Client
	logger    *logrus.Logger
	timeout   time.Duration
}

// Config holds configuration for the identity client.
type Config struct {
	BrokerEndpoint string        `json:"broker_endpoint"`
	Timeout        time.Duration `json:"timeout"`
	Logger         *logrus.Logger
}

// DefaultConfig returns a default client configuration.
func DefaultConfig() *Config {
	return &Config{
		BrokerEndpoint: "tcp://127.0.0.1:9797",
		Timeout:        30 * time.Second,
		Logger:         logrus.New(),
	}
}

// NewClient creates a new identity service client.
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if config.Logger == nil {
		config.Logger = logrus.New()
	}

	// Create MDP client
	mdpClient, err := mdp.NewClient(config.BrokerEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create MDP client: %w", err)
	}

	// Set timeout
	mdpClient.SetTimeout(config.Timeout)

	return &Client{
		mdpClient: mdpClient,
		logger:    config.Logger,
		timeout:   config.Timeout,
	}, nil
}

// Close closes the client connection.
func (c *Client) Close() error {
	return c.mdpClient.Close()
}

// sendRequest sends a request to the identity service and returns the response.
func (c *Client) sendRequest(
	_ context.Context,
	service, operation string,
	request interface{},
) ([]byte, error) {
	// Generate request ID
	requestID := fmt.Sprintf("client-%d", time.Now().UnixNano())

	// Add request header if the request supports it
	if req, ok := request.(interface{ SetRequestID(string) }); ok {
		req.SetRequestID(requestID)
	}

	// Marshal request
	requestData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"service":    service,
		"operation":  operation,
		"request_id": requestID,
	}).Debug("Sending request to identity service")

	// Send request via MDP
	if err := c.mdpClient.Send("org.plantd.Identity", service, operation, string(requestData)); err != nil {
		return nil, fmt.Errorf("failed to send MDP request: %w", err)
	}

	// Receive response
	response, err := c.mdpClient.Recv()
	if err != nil {
		return nil, fmt.Errorf("failed to receive MDP response: %w", err)
	}

	if len(response) == 0 {
		return nil, fmt.Errorf("received empty response")
	}

	c.logger.WithFields(logrus.Fields{
		"service":    service,
		"operation":  operation,
		"request_id": requestID,
	}).Debug("Received response from identity service")

	return []byte(response[0]), nil
}

// parseResponse parses a JSON response and checks for errors.
func (c *Client) parseResponse(responseData []byte, response interface{}) error {
	// First parse as a generic response to check for errors
	var genericResp struct {
		Header handlers.ResponseHeader `json:"header"`
		Code   string                  `json:"code,omitempty"`
		Detail string                  `json:"detail,omitempty"`
	}

	if err := json.Unmarshal(responseData, &genericResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	// Check if the response indicates an error
	if !genericResp.Header.Success {
		errorMsg := genericResp.Header.Error
		if genericResp.Code != "" {
			errorMsg = fmt.Sprintf("%s (%s)", errorMsg, genericResp.Code)
		}
		if genericResp.Detail != "" {
			errorMsg = fmt.Sprintf("%s: %s", errorMsg, genericResp.Detail)
		}
		return fmt.Errorf("service error: %s", errorMsg)
	}

	// Parse the full response
	if err := json.Unmarshal(responseData, response); err != nil {
		return fmt.Errorf("failed to parse response data: %w", err)
	}

	return nil
}

// Authentication methods

// Login authenticates a user and returns tokens.
func (c *Client) Login(ctx context.Context, identifier, password string) (*handlers.LoginResponse, error) {
	request := &handlers.LoginRequest{
		Header: handlers.RequestHeader{
			Timestamp: time.Now().Unix(),
		},
		Identifier: identifier,
		Password:   password,
	}

	responseData, err := c.sendRequest(ctx, "auth", "login", request)
	if err != nil {
		return nil, err
	}

	var response handlers.LoginResponse
	if err := c.parseResponse(responseData, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// RefreshToken refreshes an access token using a refresh token.
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*handlers.RefreshTokenResponse, error) {
	request := &handlers.RefreshTokenRequest{
		Header: handlers.RequestHeader{
			Timestamp: time.Now().Unix(),
		},
		RefreshToken: refreshToken,
	}

	responseData, err := c.sendRequest(ctx, "auth", "refresh", request)
	if err != nil {
		return nil, err
	}

	var response handlers.RefreshTokenResponse
	if err := c.parseResponse(responseData, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// Logout invalidates an access token.
func (c *Client) Logout(ctx context.Context, accessToken string) error {
	request := &handlers.LogoutRequest{
		Header: handlers.RequestHeader{
			Timestamp: time.Now().Unix(),
		},
		AccessToken: accessToken,
	}

	responseData, err := c.sendRequest(ctx, "auth", "logout", request)
	if err != nil {
		return err
	}

	var response handlers.LogoutResponse
	return c.parseResponse(responseData, &response)
}

// ValidateToken validates an access token.
func (c *Client) ValidateToken(ctx context.Context, token string) (*handlers.ValidateTokenResponse, error) {
	request := &handlers.ValidateTokenRequest{
		Header: handlers.RequestHeader{
			Timestamp: time.Now().Unix(),
		},
		Token: token,
	}

	responseData, err := c.sendRequest(ctx, "auth", "validate", request)
	if err != nil {
		return nil, err
	}

	var response handlers.ValidateTokenResponse
	if err := c.parseResponse(responseData, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// User management methods

// CreateUser creates a new user.
func (c *Client) CreateUser(ctx context.Context, req *handlers.CreateUserRequest) (*handlers.CreateUserResponse, error) {
	if req.Header.Timestamp == 0 {
		req.Header.Timestamp = time.Now().Unix()
	}

	responseData, err := c.sendRequest(ctx, "user", "create", req)
	if err != nil {
		return nil, err
	}

	var response handlers.CreateUserResponse
	if err := c.parseResponse(responseData, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// GetUser retrieves a user by ID, email, or username.
func (c *Client) GetUser(ctx context.Context, req *handlers.GetUserRequest) (*handlers.GetUserResponse, error) {
	if req.Header.Timestamp == 0 {
		req.Header.Timestamp = time.Now().Unix()
	}

	responseData, err := c.sendRequest(ctx, "user", "get", req)
	if err != nil {
		return nil, err
	}

	var response handlers.GetUserResponse
	if err := c.parseResponse(responseData, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// UpdateUser updates an existing user.
func (c *Client) UpdateUser(ctx context.Context, req *handlers.UpdateUserRequest) (*handlers.UpdateUserResponse, error) {
	if req.Header.Timestamp == 0 {
		req.Header.Timestamp = time.Now().Unix()
	}

	responseData, err := c.sendRequest(ctx, "user", "update", req)
	if err != nil {
		return nil, err
	}

	var response handlers.UpdateUserResponse
	if err := c.parseResponse(responseData, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// DeleteUser deletes a user.
func (c *Client) DeleteUser(ctx context.Context, userID uint) error {
	request := &handlers.DeleteUserRequest{
		Header: handlers.RequestHeader{
			Timestamp: time.Now().Unix(),
		},
		UserID: userID,
	}

	responseData, err := c.sendRequest(ctx, "user", "delete", request)
	if err != nil {
		return err
	}

	var response handlers.DeleteUserResponse
	return c.parseResponse(responseData, &response)
}

// ListUsers lists users with pagination and filtering.
func (c *Client) ListUsers(ctx context.Context, req *handlers.ListUsersRequest) (*handlers.ListUsersResponse, error) {
	if req.Header.Timestamp == 0 {
		req.Header.Timestamp = time.Now().Unix()
	}

	responseData, err := c.sendRequest(ctx, "user", "list", req)
	if err != nil {
		return nil, err
	}

	var response handlers.ListUsersResponse
	if err := c.parseResponse(responseData, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// Health check methods

// HealthCheck performs a health check on the identity service.
func (c *Client) HealthCheck(ctx context.Context) (*handlers.HealthCheckResponse, error) {
	request := &handlers.HealthCheckRequest{
		Header: handlers.RequestHeader{
			Timestamp: time.Now().Unix(),
		},
	}

	responseData, err := c.sendRequest(ctx, "health", "check", request)
	if err != nil {
		return nil, err
	}

	var response handlers.HealthCheckResponse
	if err := c.parseResponse(responseData, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// Convenience methods

// LoginWithEmail authenticates a user using email and password.
func (c *Client) LoginWithEmail(ctx context.Context, email, password string) (*handlers.LoginResponse, error) {
	return c.Login(ctx, email, password)
}

// LoginWithUsername authenticates a user using username and password.
func (c *Client) LoginWithUsername(ctx context.Context, username, password string) (*handlers.LoginResponse, error) {
	return c.Login(ctx, username, password)
}

// GetUserByID retrieves a user by ID.
func (c *Client) GetUserByID(ctx context.Context, userID uint) (*handlers.GetUserResponse, error) {
	request := &handlers.GetUserRequest{
		Header: handlers.RequestHeader{
			Timestamp: time.Now().Unix(),
		},
		UserID: &userID,
	}
	return c.GetUser(ctx, request)
}

// GetUserByEmail retrieves a user by email.
func (c *Client) GetUserByEmail(ctx context.Context, email string) (*handlers.GetUserResponse, error) {
	request := &handlers.GetUserRequest{
		Header: handlers.RequestHeader{
			Timestamp: time.Now().Unix(),
		},
		Email: email,
	}
	return c.GetUser(ctx, request)
}

// GetUserByUsername retrieves a user by username.
func (c *Client) GetUserByUsername(ctx context.Context, username string) (*handlers.GetUserResponse, error) {
	request := &handlers.GetUserRequest{
		Header: handlers.RequestHeader{
			Timestamp: time.Now().Unix(),
		},
		Username: username,
	}
	return c.GetUser(ctx, request)
}
