// Package services provides business logic for service integrations.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/geoffjay/plantd/app/config"
	healthv1 "github.com/geoffjay/plantd/gen/proto/go/plantd/health/v1"
	statev1 "github.com/geoffjay/plantd/gen/proto/go/plantd/state/v1"
	"github.com/geoffjay/plantd/gen/proto/go/plantd/state/v1/statev1connect"
	"google.golang.org/protobuf/types/known/emptypb"

	log "github.com/sirupsen/logrus"
)

// GRPCStateService handles communication with the plantd state service using gRPC.
type GRPCStateService struct {
	client   statev1connect.StateServiceClient
	config   *config.Config
	baseURL  string
	timeout  time.Duration
	logger   *log.Entry
	authFunc func() string // Function to get current auth token
}

// NewGRPCStateService creates a new gRPC-based state service client.
func NewGRPCStateService(cfg *config.Config, authFunc func() string) (*GRPCStateService, error) {
	logger := log.WithField("service", "grpc_state_client")

	// Determine base URL for Traefik gateway
	baseURL := "http://localhost:8080" // Default Traefik gateway
	if cfg.Services.StateEndpoint != "" {
		// Parse endpoint and convert to HTTP URL
		// For now, assume Traefik is on localhost:8080
		baseURL = "http://localhost:8080"
	}

	// Parse timeout
	timeout, err := time.ParseDuration(cfg.Services.Timeout)
	if err != nil {
		timeout = 30 * time.Second
		logger.WithError(err).Warn("Failed to parse state service timeout, using default 30s")
	}

	// Create HTTP client for Connect
	httpClient := &http.Client{
		Timeout: timeout,
	}

	// Create Connect client
	client := statev1connect.NewStateServiceClient(
		httpClient,
		baseURL,
		connect.WithGRPC(), // Use gRPC protocol
	)

	ss := &GRPCStateService{
		client:   client,
		config:   cfg,
		baseURL:  baseURL,
		timeout:  timeout,
		logger:   logger,
		authFunc: authFunc,
	}

	logger.WithFields(log.Fields{
		"base_url": baseURL,
		"timeout":  timeout,
	}).Info("gRPC State service client initialized")

	return ss, nil
}

// Close closes the state service client connection.
func (ss *GRPCStateService) Close() error {
	// HTTP client doesn't require explicit closing
	return nil
}

// GetAllScopes retrieves a list of all state scopes.
// Note: Since gRPC State service doesn't have ListScopes, this is a placeholder
func (ss *GRPCStateService) GetAllScopes(ctx context.Context, userToken string) ([]string, error) {
	ss.logger.Debug("Getting all scopes via gRPC - using placeholder implementation")

	// For now, return common scopes since the gRPC service doesn't have ListScopes yet
	// TODO: Implement proper scope listing when available in the gRPC service
	scopes := []string{
		"org.plantd.Client",
		"system",
		"org.plantd.State",
	}

	ss.logger.WithField("scope_count", len(scopes)).Debug("Retrieved scopes via gRPC (placeholder)")
	return scopes, nil
}

// GetStateValue retrieves a state value by key from the specified scope.
func (ss *GRPCStateService) GetStateValue(ctx context.Context, userToken, scope, key string) (*StateValue, error) {
	ss.logger.WithFields(log.Fields{
		"scope": scope,
		"key":   key,
	}).Debug("Getting state value via gRPC")

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, ss.timeout)
	defer cancel()

	// Create request
	req := connect.NewRequest(&statev1.GetRequest{
		Scope: scope,
		Key:   key,
	})

	// Add authentication header
	if userToken != "" {
		req.Header().Set("Authorization", "Bearer "+userToken)
	} else if ss.authFunc != nil {
		if token := ss.authFunc(); token != "" {
			req.Header().Set("Authorization", "Bearer "+token)
		}
	}

	// Make request
	resp, err := ss.client.Get(timeoutCtx, req)
	if err != nil {
		ss.logger.WithError(err).WithFields(log.Fields{
			"scope": scope,
			"key":   key,
		}).Error("Failed to get state value")
		return nil, fmt.Errorf("failed to get state value: %w", err)
	}

	// Parse value
	var value interface{}
	if resp.Msg.Value != "" {
		if err := json.Unmarshal([]byte(resp.Msg.Value), &value); err != nil {
			// If JSON parsing fails, use as string
			value = resp.Msg.Value
		}
	}

	stateData := &StateValue{
		Scope:   scope,
		Key:     key,
		Value:   value,
		SetAt:   time.Now(),
		Version: 1,
	}

	ss.logger.WithFields(log.Fields{
		"scope": scope,
		"key":   key,
		"value": stateData.Value,
	}).Debug("Retrieved state value via gRPC")

	return stateData, nil
}

// SetStateValue sets a state value with authentication.
func (ss *GRPCStateService) SetStateValue(ctx context.Context, userToken, scope, key string, value interface{}) error {
	ss.logger.WithFields(log.Fields{
		"scope": scope,
		"key":   key,
	}).Debug("Setting state value via gRPC")

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, ss.timeout)
	defer cancel()

	// Serialize the value
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %w", err)
	}

	// Create request
	req := connect.NewRequest(&statev1.SetRequest{
		Scope: scope,
		Key:   key,
		Value: string(valueBytes),
	})

	// Add authentication header
	if userToken != "" {
		req.Header().Set("Authorization", "Bearer "+userToken)
	} else if ss.authFunc != nil {
		if token := ss.authFunc(); token != "" {
			req.Header().Set("Authorization", "Bearer "+token)
		}
	}

	// Make request
	_, err = ss.client.Set(timeoutCtx, req)
	if err != nil {
		ss.logger.WithError(err).WithFields(log.Fields{
			"scope": scope,
			"key":   key,
		}).Error("Failed to set state value")
		return fmt.Errorf("failed to set state value: %w", err)
	}

	ss.logger.WithFields(log.Fields{
		"scope": scope,
		"key":   key,
	}).Info("State value set successfully via gRPC")

	return nil
}

// DeleteStateValue deletes a state value with authentication.
func (ss *GRPCStateService) DeleteStateValue(ctx context.Context, userToken, scope, key string) error {
	ss.logger.WithFields(log.Fields{
		"scope": scope,
		"key":   key,
	}).Debug("Deleting state value via gRPC")

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, ss.timeout)
	defer cancel()

	// Create request
	req := connect.NewRequest(&statev1.DeleteRequest{
		Scope: scope,
		Key:   key,
	})

	// Add authentication header
	if userToken != "" {
		req.Header().Set("Authorization", "Bearer "+userToken)
	} else if ss.authFunc != nil {
		if token := ss.authFunc(); token != "" {
			req.Header().Set("Authorization", "Bearer "+token)
		}
	}

	// Make request
	_, err := ss.client.Delete(timeoutCtx, req)
	if err != nil {
		ss.logger.WithError(err).WithFields(log.Fields{
			"scope": scope,
			"key":   key,
		}).Error("Failed to delete state value")
		return fmt.Errorf("failed to delete state value: %w", err)
	}

	ss.logger.WithFields(log.Fields{
		"scope": scope,
		"key":   key,
	}).Info("State value deleted successfully via gRPC")

	return nil
}

// CheckConnectivity verifies connectivity to the state service.
func (ss *GRPCStateService) CheckConnectivity(ctx context.Context) error {
	ss.logger.Debug("Checking state service connectivity via gRPC")

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Use health check endpoint
	req := connect.NewRequest(&emptypb.Empty{})

	// Add authentication header if available
	if ss.authFunc != nil {
		if token := ss.authFunc(); token != "" {
			req.Header().Set("Authorization", "Bearer "+token)
		}
	}

	// Make request
	resp, err := ss.client.Health(timeoutCtx, req)
	if err != nil {
		ss.logger.WithError(err).Error("State service connectivity check failed")
		return fmt.Errorf("state service connectivity check failed: %w", err)
	}

	if resp.Msg.Status != healthv1.HealthCheckResponse_SERVING_STATUS_SERVING {
		return fmt.Errorf("state service reported unhealthy status")
	}

	ss.logger.Debug("State service connectivity check successful")
	return nil
}

// HealthCheck performs a comprehensive health check of the state service.
func (ss *GRPCStateService) HealthCheck(ctx context.Context) (map[string]interface{}, error) {
	healthStatus := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"checks":    map[string]interface{}{},
	}

	checks := healthStatus["checks"].(map[string]interface{})

	// Check connectivity
	if err := ss.CheckConnectivity(ctx); err != nil {
		checks["connectivity"] = map[string]interface{}{
			"status": "failed",
			"error":  err.Error(),
		}
		healthStatus["status"] = "unhealthy"
	} else {
		checks["connectivity"] = map[string]interface{}{
			"status": "passed",
		}
	}

	// Try to get scope list
	_, err := ss.GetAllScopes(ctx, "")
	if err != nil {
		checks["data_access"] = map[string]interface{}{
			"status": "failed",
			"error":  err.Error(),
		}
		healthStatus["status"] = "degraded"
	} else {
		checks["data_access"] = map[string]interface{}{
			"status": "passed",
		}
	}

	return healthStatus, nil
}
