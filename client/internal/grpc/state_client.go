// Package grpc provides gRPC client implementations for plantd services.
package grpc

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"connectrpc.com/connect"
	statev1 "github.com/geoffjay/plantd/gen/proto/go/plantd/state/v1"
	"github.com/geoffjay/plantd/gen/proto/go/plantd/state/v1/statev1connect"
	log "github.com/sirupsen/logrus"
)

// StateClient provides a gRPC client for the State service.
type StateClient struct {
	client   statev1connect.StateServiceClient
	baseURL  string
	timeout  time.Duration
	logger   *log.Entry
	authFunc func() string // Function to get current auth token
}

// StateClientConfig holds configuration for the State gRPC client.
type StateClientConfig struct {
	BaseURL    string        // Traefik gateway URL (e.g., "http://localhost:8080")
	Timeout    time.Duration // Request timeout
	AuthFunc   func() string // Function to retrieve auth token
	HTTPClient *http.Client  // Optional custom HTTP client
}

// NewStateClient creates a new State service gRPC client.
func NewStateClient(config *StateClientConfig) *StateClient {
	logger := log.WithField("service", "state_grpc_client")

	// Use default HTTP client if none provided
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: config.Timeout,
		}
	}

	// Create Connect client
	client := statev1connect.NewStateServiceClient(
		httpClient,
		config.BaseURL,
		connect.WithGRPC(), // Use gRPC protocol
	)

	return &StateClient{
		client:   client,
		baseURL:  config.BaseURL,
		timeout:  config.Timeout,
		logger:   logger,
		authFunc: config.AuthFunc,
	}
}

// Get retrieves a value by key from the specified scope.
func (sc *StateClient) Get(ctx context.Context, scope, key string) (string, error) {
	sc.logger.WithFields(log.Fields{
		"scope": scope,
		"key":   key,
	}).Debug("Getting state value via gRPC")

	// Create request
	req := connect.NewRequest(&statev1.GetRequest{
		Scope: scope,
		Key:   key,
	})

	// Add authentication header if available
	if sc.authFunc != nil {
		if token := sc.authFunc(); token != "" {
			req.Header().Set("Authorization", "Bearer "+token)
		}
	}

	// Set timeout
	if sc.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, sc.timeout)
		defer cancel()
	}

	// Make request
	resp, err := sc.client.Get(ctx, req)
	if err != nil {
		sc.logger.WithError(err).WithFields(log.Fields{
			"scope": scope,
			"key":   key,
		}).Error("Failed to get state value")
		return "", err
	}

	value := resp.Msg.Value

	sc.logger.WithFields(log.Fields{
		"scope": scope,
		"key":   key,
		"found": value != "",
	}).Debug("Retrieved state value via gRPC")

	return value, nil
}

// Set stores a value by key in the specified scope.
func (sc *StateClient) Set(ctx context.Context, scope, key, value string) error {
	sc.logger.WithFields(log.Fields{
		"scope": scope,
		"key":   key,
	}).Debug("Setting state value via gRPC")

	// Create request
	req := connect.NewRequest(&statev1.SetRequest{
		Scope: scope,
		Key:   key,
		Value: value,
	})

	// Add authentication header if available
	if sc.authFunc != nil {
		if token := sc.authFunc(); token != "" {
			req.Header().Set("Authorization", "Bearer "+token)
		}
	}

	// Set timeout
	if sc.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, sc.timeout)
		defer cancel()
	}

	// Make request
	_, err := sc.client.Set(ctx, req)
	if err != nil {
		sc.logger.WithError(err).WithFields(log.Fields{
			"scope": scope,
			"key":   key,
		}).Error("Failed to set state value")
		return err
	}

	sc.logger.WithFields(log.Fields{
		"scope": scope,
		"key":   key,
	}).Info("Set state value via gRPC")

	return nil
}

// Delete removes a value by key from the specified scope.
func (sc *StateClient) Delete(ctx context.Context, scope, key string) error {
	sc.logger.WithFields(log.Fields{
		"scope": scope,
		"key":   key,
	}).Debug("Deleting state value via gRPC")

	// Create request
	req := connect.NewRequest(&statev1.DeleteRequest{
		Scope: scope,
		Key:   key,
	})

	// Add authentication header if available
	if sc.authFunc != nil {
		if token := sc.authFunc(); token != "" {
			req.Header().Set("Authorization", "Bearer "+token)
		}
	}

	// Set timeout
	if sc.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, sc.timeout)
		defer cancel()
	}

	// Make request
	resp, err := sc.client.Delete(ctx, req)
	if err != nil {
		sc.logger.WithError(err).WithFields(log.Fields{
			"scope": scope,
			"key":   key,
		}).Error("Failed to delete state value")
		return err
	}

	sc.logger.WithFields(log.Fields{
		"scope":   scope,
		"key":     key,
		"existed": resp.Msg.Existed,
	}).Info("Deleted state value via gRPC")

	return nil
}

// List retrieves all keys in the specified scope.
func (sc *StateClient) List(ctx context.Context, scope string) ([]string, error) {
	sc.logger.WithField("scope", scope).Debug("Listing state keys via gRPC")

	// Create request
	req := connect.NewRequest(&statev1.ListRequest{
		Scope: scope,
	})

	// Add authentication header if available
	if sc.authFunc != nil {
		if token := sc.authFunc(); token != "" {
			req.Header().Set("Authorization", "Bearer "+token)
		}
	}

	// Set timeout
	if sc.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, sc.timeout)
		defer cancel()
	}

	// Make request
	stream, err := sc.client.List(ctx, req)
	if err != nil {
		sc.logger.WithError(err).WithField("scope", scope).Error("Failed to list state keys")
		return nil, err
	}

	// Collect all keys from the stream
	var keys []string
	for stream.Receive() {
		msg := stream.Msg()
		if msg.Key != "" {
			keys = append(keys, msg.Key)
		}
	}

	if err := stream.Err(); err != nil {
		sc.logger.WithError(err).WithField("scope", scope).Error("Error reading state keys stream")
		return nil, err
	}

	sc.logger.WithFields(log.Fields{
		"scope":     scope,
		"key_count": len(keys),
	}).Debug("Listed state keys via gRPC")

	return keys, nil
}

// ListScopes retrieves all available scopes.
// Note: This uses the MDP compatibility endpoint since gRPC doesn't have a dedicated ListScopes method yet.
func (sc *StateClient) ListScopes(ctx context.Context) ([]string, error) {
	sc.logger.Debug("Listing state scopes via MDP compatibility endpoint")

	// Use MDP compatibility endpoint for listing scopes
	httpClient := &http.Client{Timeout: sc.timeout}

	req, err := http.NewRequestWithContext(ctx, "POST", sc.baseURL+"/mdp/list-scopes", nil)
	if err != nil {
		return nil, err
	}

	// Add authentication header if available
	if sc.authFunc != nil {
		if token := sc.authFunc(); token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		sc.logger.WithError(err).Error("Failed to list state scopes")
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		sc.logger.WithField("status_code", resp.StatusCode).Error("List scopes request failed")
		return nil, connect.NewError(connect.CodeInternal, nil)
	}

	// Parse JSON response
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		sc.logger.WithError(err).Error("Failed to parse list scopes response")
		return nil, err
	}

	// Extract scopes from response
	var scopes []string
	if data, ok := response["data"].([]interface{}); ok {
		for _, item := range data {
			if scope, ok := item.(string); ok {
				scopes = append(scopes, scope)
			}
		}
	}

	sc.logger.WithField("scope_count", len(scopes)).Debug("Listed state scopes via MDP compatibility")

	return scopes, nil
}

// Health checks the health of the State service.
func (sc *StateClient) Health(ctx context.Context) error {
	sc.logger.Debug("Checking state service health via gRPC")

	// Use a simple HTTP GET to the health endpoint through the gateway
	httpClient := &http.Client{Timeout: sc.timeout}

	req, err := http.NewRequestWithContext(ctx, "GET", sc.baseURL+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		sc.logger.WithError(err).Error("State service health check failed")
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		sc.logger.WithField("status_code", resp.StatusCode).Error("State service health check failed")
		return connect.NewError(connect.CodeUnavailable, nil)
	}

	sc.logger.Debug("State service health check passed")
	return nil
}
