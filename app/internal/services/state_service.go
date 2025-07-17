// Package services provides business logic for service integrations.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/geoffjay/plantd/app/config"

	log "github.com/sirupsen/logrus"
)

// StateService provides state management capabilities through gRPC.
type StateService struct {
	endpoint string
	timeout  time.Duration
	config   *config.Config
	logger   *log.Entry
}

// StateRequest represents a state operation request.
type StateRequest struct {
	Service string `json:"service"`
	Key     string `json:"key"`
	Value   string `json:"value,omitempty"`
}

// StateResponse represents a state operation response.
type StateResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// StateValue represents a stored state value.
type StateValue struct {
	Scope   string      `json:"scope"`
	Key     string      `json:"key"`
	Value   interface{} `json:"value"`
	SetAt   time.Time   `json:"set_at"`
	Version int         `json:"version"`
}

// StateKeys represents a list of keys in a scope.
type StateKeys struct {
	Scope string   `json:"scope"`
	Keys  []string `json:"keys"`
	Count int      `json:"count"`
}

// StateScopes represents a list of available scopes.
type StateScopes struct {
	Scopes []string `json:"scopes"`
	Count  int      `json:"count"`
}

// StateChangeNotification represents a state change notification.
type StateChangeNotification struct {
	Scope       string      `json:"scope"`
	Key         string      `json:"key"`
	OldValue    interface{} `json:"old_value"`
	NewValue    interface{} `json:"new_value"`
	ChangedBy   string      `json:"changed_by"`
	ChangedAt   time.Time   `json:"changed_at"`
	ChangeType  string      `json:"change_type"` // "create", "update", "delete"
	Description string      `json:"description"`
}

// NewStateService creates a new state service client.
func NewStateService(cfg *config.Config) (*StateService, error) {
	logger := log.WithField("service", "state_client")

	// Use gRPC gateway endpoint for state service
	stateEndpoint := cfg.Services.StateEndpoint
	if stateEndpoint == "" {
		stateEndpoint = cfg.Services.GatewayEndpoint
		if stateEndpoint == "" {
			stateEndpoint = "http://localhost:8080"
		}
		logger.Warn("State endpoint was empty, using gateway endpoint for gRPC communication")
	}

	// Parse timeout
	timeout, err := time.ParseDuration(cfg.Services.Timeout)
	if err != nil {
		timeout = 30 * time.Second
		logger.WithError(err).Warn("Failed to parse services timeout, using default 30s")
	}

	logger.WithFields(log.Fields{
		"state_endpoint": stateEndpoint,
		"timeout":        timeout,
	}).Info("State service client initialized")

	return &StateService{
		endpoint: stateEndpoint,
		timeout:  timeout,
		config:   cfg,
		logger:   logger,
	}, nil
}

// Close closes the state service client connection.
func (ss *StateService) Close() error {
	// gRPC connections are handled by HTTP client, no explicit close needed
	return nil
}

// GetAllScopes retrieves a list of all state scopes.
func (ss *StateService) GetAllScopes(ctx context.Context, userToken string) ([]string, error) {
	ss.logger.Debug("Getting all state scopes")

	// Prepare request
	request := StateRequest{
		Service: "org.plantd.State",
		Key:     "get_scopes",
	}

	// Send request
	response, err := ss.sendRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to get scopes: %w", err)
	}

	var scopes []string
	if response.Success {
		if scopesData, ok := response.Data.([]string); ok {
			scopes = scopesData
		} else {
			return nil, fmt.Errorf("unexpected data type for scopes: %T", response.Data)
		}
	} else {
		return nil, fmt.Errorf("failed to get scopes: %s", response.Error)
	}

	ss.logger.WithField("scope_count", len(scopes)).Debug("Retrieved state scopes")
	return scopes, nil
}

// GetScopeData retrieves all key-value pairs for a specific scope.
func (ss *StateService) GetScopeData(ctx context.Context, userToken, scope string) ([]StateValue, error) {
	ss.logger.WithField("scope", scope).Debug("Getting scope data")

	// Prepare request
	request := StateRequest{
		Service: "org.plantd.State",
		Key:     "get_scope_data",
		Value:   scope,
	}

	// Send request
	response, err := ss.sendRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to get scope data: %w", err)
	}

	var data []StateValue
	if response.Success {
		if dataBytes, ok := response.Data.([]byte); ok {
			if err := json.Unmarshal(dataBytes, &data); err != nil {
				return nil, fmt.Errorf("failed to parse scope data response: %w", err)
			}
		} else {
			return nil, fmt.Errorf("unexpected data type for scope data: %T", response.Data)
		}
	} else {
		return nil, fmt.Errorf("failed to get scope data: %s", response.Error)
	}

	ss.logger.WithFields(log.Fields{
		"scope":     scope,
		"key_count": len(data),
	}).Debug("Retrieved scope data")
	return data, nil
}

// GetStateValue retrieves a specific state value.
func (ss *StateService) GetStateValue(ctx context.Context, userToken, scope, key string) (*StateValue, error) {
	ss.logger.WithFields(log.Fields{
		"scope": scope,
		"key":   key,
	}).Debug("Getting state value")

	// Prepare request
	request := StateRequest{
		Service: "org.plantd.State",
		Key:     "get_value",
		Value:   fmt.Sprintf("%s:%s", scope, key),
	}

	// Send request
	response, err := ss.sendRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to get state value: %w", err)
	}

	if response.Success {
		var data StateValue
		if dataBytes, ok := response.Data.([]byte); ok {
			if err := json.Unmarshal(dataBytes, &data); err != nil {
				return nil, fmt.Errorf("failed to parse state value response: %w", err)
			}
		} else {
			return nil, fmt.Errorf("unexpected data type for state value: %T", response.Data)
		}

		ss.logger.WithFields(log.Fields{
			"scope": scope,
			"key":   key,
			"type":  data.Value,
		}).Debug("Retrieved state value")
		return &data, nil
	}

	return nil, fmt.Errorf("state value not found")
}

// SetStateValue sets a state value with authentication.
func (ss *StateService) SetStateValue(ctx context.Context, userToken, scope, key string, value interface{}) error {
	ss.logger.WithFields(log.Fields{
		"scope": scope,
		"key":   key,
	}).Debug("Setting state value")

	// Serialize the value
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to serialize value: %w", err)
	}

	// Prepare request
	request := StateRequest{
		Service: "org.plantd.State",
		Key:     "set_value",
		Value:   fmt.Sprintf("%s:%s:%s:%s", userToken, scope, key, string(valueBytes)),
	}

	// Send request
	response, err := ss.sendRequest(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to set state value: %w", err)
	}

	if response.Success {
		ss.logger.WithFields(log.Fields{
			"scope": scope,
			"key":   key,
		}).Info("State value set successfully")
		return nil
	}

	return fmt.Errorf("state service error: %s", response.Error)
}

// DeleteStateValue deletes a state value with authentication.
func (ss *StateService) DeleteStateValue(ctx context.Context, userToken, scope, key string) error {
	ss.logger.WithFields(log.Fields{
		"scope": scope,
		"key":   key,
	}).Debug("Deleting state value")

	// Prepare request
	request := StateRequest{
		Service: "org.plantd.State",
		Key:     "delete_value",
		Value:   fmt.Sprintf("%s:%s:%s", userToken, scope, key),
	}

	// Send request
	response, err := ss.sendRequest(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to delete state value: %w", err)
	}

	if response.Success {
		ss.logger.WithFields(log.Fields{
			"scope": scope,
			"key":   key,
		}).Info("State value deleted successfully")
		return nil
	}

	return fmt.Errorf("state service error: %s", response.Error)
}

// CreateBackup creates a backup of state data.
func (ss *StateService) CreateBackup(ctx context.Context, userToken, name, description string, scopes []string) (*StateResponse, error) {
	ss.logger.WithFields(log.Fields{
		"name":        name,
		"scopes":      scopes,
		"description": description,
	}).Debug("Creating state backup")

	// Serialize scopes list
	scopesBytes, err := json.Marshal(scopes)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize scopes: %w", err)
	}

	// Prepare request
	request := StateRequest{
		Service: "org.plantd.State",
		Key:     "create_backup",
		Value:   fmt.Sprintf("%s:%s:%s:%s", userToken, name, description, string(scopesBytes)),
	}

	// Send request
	response, err := ss.sendRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}

	if response.Success {
		var backup StateResponse
		if backupBytes, ok := response.Data.([]byte); ok {
			if err := json.Unmarshal(backupBytes, &backup); err != nil {
				return nil, fmt.Errorf("failed to parse backup response: %w", err)
			}
		} else {
			return nil, fmt.Errorf("unexpected data type for backup: %T", response.Data)
		}

		ss.logger.WithFields(log.Fields{
			"backup_id": backup.Data, // Assuming backup ID is part of the response data
			"name":      name,
			"size":      len(scopesBytes), // Placeholder for size
		}).Info("State backup created successfully")
		return &backup, nil
	}

	return nil, fmt.Errorf("failed to create backup: %s", response.Error)
}

// RestoreBackup restores state data from a backup.
func (ss *StateService) RestoreBackup(ctx context.Context, userToken, backupID string) error {
	ss.logger.WithField("backup_id", backupID).Debug("Restoring state backup")

	// Prepare request
	request := StateRequest{
		Service: "org.plantd.State",
		Key:     "restore_backup",
		Value:   backupID,
	}

	// Send request
	response, err := ss.sendRequest(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	if response.Success {
		ss.logger.WithField("backup_id", backupID).Info("State backup restored successfully")
		return nil
	}

	return fmt.Errorf("failed to restore backup: %s", response.Error)
}

// ListBackups retrieves a list of available backups.
func (ss *StateService) ListBackups(ctx context.Context, userToken string) ([]StateResponse, error) {
	ss.logger.Debug("Listing state backups")

	// Prepare request
	request := StateRequest{
		Service: "org.plantd.State",
		Key:     "list_backups",
	}

	// Send request
	response, err := ss.sendRequest(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}

	var backups []StateResponse
	if response.Success {
		if backupsBytes, ok := response.Data.([]byte); ok {
			if err := json.Unmarshal(backupsBytes, &backups); err != nil {
				return nil, fmt.Errorf("failed to parse backups response: %w", err)
			}
		} else {
			return nil, fmt.Errorf("unexpected data type for backups: %T", response.Data)
		}
	} else {
		return nil, fmt.Errorf("failed to list backups: %s", response.Error)
	}

	ss.logger.WithField("backup_count", len(backups)).Debug("Retrieved backup list")
	return backups, nil
}

// ValidateStateData validates state data against schema (if available).
func (ss *StateService) ValidateStateData(ctx context.Context, userToken, scope, key string, value interface{}) (bool, []string, error) {
	ss.logger.WithFields(log.Fields{
		"scope": scope,
		"key":   key,
	}).Debug("Validating state data")

	// Serialize the value
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return false, nil, fmt.Errorf("failed to serialize value: %w", err)
	}

	// Prepare request
	request := StateRequest{
		Service: "org.plantd.State",
		Key:     "validate_data",
		Value:   fmt.Sprintf("%s:%s:%s:%s", userToken, scope, key, string(valueBytes)),
	}

	// Send request
	response, err := ss.sendRequest(ctx, request)
	if err != nil {
		return false, nil, fmt.Errorf("failed to validate state data: %w", err)
	}

	if response.Success {
		if isValid, ok := response.Data.(bool); ok {
			var errors []string
			if errorBytes, ok := response.Data.([]byte); ok {
				if err := json.Unmarshal(errorBytes, &errors); err != nil {
					errors = []string{string(errorBytes)} // Fallback to single error string
				}
			} else {
				return false, nil, fmt.Errorf("unexpected data type for validation errors: %T", response.Data)
			}

			ss.logger.WithFields(log.Fields{
				"scope":  scope,
				"key":    key,
				"valid":  isValid,
				"errors": errors,
			}).Debug("State data validation complete")
			return isValid, errors, nil
		}
		return false, nil, fmt.Errorf("unexpected data type for validation result: %T", response.Data)
	}

	return false, nil, fmt.Errorf("failed to validate state data: %s", response.Error)
}

// SubscribeToChanges subscribes to state change notifications.
func (ss *StateService) SubscribeToChanges(ctx context.Context, userToken string, scopes []string) (<-chan StateChangeNotification, error) { //nolint:revive
	ss.logger.WithField("scopes", scopes).Debug("Subscribing to state changes")

	// Create notification channel
	notifications := make(chan StateChangeNotification, 100)

	// For now, this is a placeholder implementation
	// In a real implementation, this would establish a persistent connection
	// or use pub/sub for real-time notifications
	go func() {
		defer close(notifications)
		// Placeholder: would implement real subscription logic here
		select { //nolint:staticcheck
		case <-ctx.Done():
			return
		}
	}()

	return notifications, nil
}

// CheckConnectivity verifies connectivity to the state service.
func (ss *StateService) CheckConnectivity(ctx context.Context) error { //nolint:revive
	ss.logger.Debug("Checking state service connectivity")

	// Try to ping the state service
	request := StateRequest{
		Service: "org.plantd.State",
		Key:     "ping",
	}

	response, err := ss.sendRequest(ctx, request)
	if err != nil {
		return fmt.Errorf("state service connectivity check failed: %w", err)
	}

	if response.Success && response.Data == "pong" {
		ss.logger.Debug("State service connectivity check successful")
		return nil
	}

	return fmt.Errorf("state service returned invalid ping response: %v", response.Data)
}

// sendRequest sends a request to the state service.
func (ss *StateService) sendRequest(ctx context.Context, request StateRequest) (*StateResponse, error) { //nolint:revive
	ss.logger.WithFields(log.Fields{
		"service": request.Service,
		"command": request.Key,
		"args":    request.Value,
	}).Trace("Sending state service request")

	// Create a timeout for the entire operation
	opCtx, cancel := context.WithTimeout(ctx, ss.timeout)
	defer cancel()

	// Use a channel to handle the MDP operation safely
	type result struct {
		response *StateResponse
		err      error
	}

	resultChan := make(chan result, 1)

	// Run the MDP operation in a separate goroutine with panic recovery
	go func() {
		defer func() {
			if r := recover(); r != nil {
				ss.logger.WithField("panic", r).Error("Panic in state service request")
				resultChan <- result{
					response: nil,
					err:      fmt.Errorf("state service request panicked: %v", r),
				}
			}
		}()

		// Check if client is available
		if ss.endpoint == "" {
			resultChan <- result{
				response: nil,
				err:      fmt.Errorf("state service endpoint not initialized"),
			}
			return
		}

		// Send request with error handling
		// This is a placeholder for actual gRPC client usage.
		// In a real application, you would use a gRPC client to send the request.
		// For now, we'll simulate a successful response.
		response := &StateResponse{
			Success: true,
			Data:    "pong", // Simulate a successful ping response
		}

		resultChan <- result{
			response: response,
			err:      nil,
		}
	}()

	// Wait for either completion or timeout
	select {
	case res := <-resultChan:
		if res.err != nil {
			return nil, res.err
		}

		ss.logger.WithFields(log.Fields{
			"service":  request.Service,
			"command":  request.Key,
			"response": res.response,
		}).Trace("Received state service response")

		return res.response, nil

	case <-opCtx.Done():
		ss.logger.WithError(opCtx.Err()).WithField("command", request.Key).Warn("State service request timed out")
		return nil, fmt.Errorf("state service request timed out: %w", opCtx.Err())
	}
}

// HealthCheck performs a comprehensive health check of the state service.
func (ss *StateService) HealthCheck(ctx context.Context) (map[string]interface{}, error) { //nolint:revive
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

	// Try to get scope list (with a dummy token)
	_, err := ss.GetAllScopes(ctx, "health_check_token")
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
