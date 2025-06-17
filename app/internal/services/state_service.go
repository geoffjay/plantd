// Package services provides business logic for service integrations.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/geoffjay/plantd/app/config"
	"github.com/geoffjay/plantd/core/mdp"

	log "github.com/sirupsen/logrus"
)

// StateService handles communication with the plantd state service for configuration and state management.
type StateService struct {
	client *mdp.Client
	config *config.Config
	logger *log.Entry
}

// StateData represents a state key-value pair with metadata.
type StateData struct {
	Scope       string      `json:"scope"`
	Key         string      `json:"key"`
	Value       interface{} `json:"value"`
	Type        string      `json:"type"`
	LastUpdated time.Time   `json:"last_updated"`
	Version     int         `json:"version"`
	CreatedBy   string      `json:"created_by"`
	UpdatedBy   string      `json:"updated_by"`
}

// StateScope represents a logical grouping of state data.
type StateScope struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	KeyCount    int       `json:"key_count"`
	LastUpdated time.Time `json:"last_updated"`
	AccessLevel string    `json:"access_level"` // "read", "write", "admin"
}

// StateBackup represents a backup snapshot of state data.
type StateBackup struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Scopes      []string               `json:"scopes"`
	CreatedAt   time.Time              `json:"created_at"`
	CreatedBy   string                 `json:"created_by"`
	Size        int64                  `json:"size"`
	Data        map[string][]StateData `json:"data"`
	Description string                 `json:"description"`
}

// StateChangeNotification represents a state change event.
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

	// Workaround: If state endpoint is empty, use broker endpoint
	// (state service communicates through broker)
	stateEndpoint := cfg.Services.StateEndpoint
	if stateEndpoint == "" {
		// Use broker endpoint since state service is an MDP service
		stateEndpoint = cfg.Services.BrokerEndpoint
		if stateEndpoint == "" {
			stateEndpoint = "tcp://127.0.0.1:9797"
		}
		logger.Warn("State endpoint was empty, using broker endpoint for MDP communication")
	}

	// Create MDP client for state service communication
	client, err := mdp.NewClient(stateEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create MDP client for state service: %w", err)
	}

	// Parse timeout
	timeout, err := time.ParseDuration(cfg.Services.Timeout)
	if err != nil {
		timeout = 30 * time.Second
		logger.WithError(err).Warn("Failed to parse services timeout, using default 30s")
	}
	client.SetTimeout(timeout)

	logger.WithFields(log.Fields{
		"state_endpoint": stateEndpoint,
		"timeout":        timeout,
	}).Info("State service client initialized")

	return &StateService{
		client: client,
		config: cfg,
		logger: logger,
	}, nil
}

// Close closes the state service client connection.
func (ss *StateService) Close() error {
	if ss.client != nil {
		return ss.client.Close()
	}
	return nil
}

// GetAllScopes retrieves a list of all state scopes.
func (ss *StateService) GetAllScopes(ctx context.Context, userToken string) ([]StateScope, error) {
	ss.logger.Debug("Getting all state scopes")

	// Send authenticated request to state service
	response, err := ss.sendAuthenticatedRequest(ctx, userToken, "get_scopes")
	if err != nil {
		return nil, fmt.Errorf("failed to get scopes: %w", err)
	}

	var scopes []StateScope
	if len(response) > 0 && response[0] != "" {
		if err := json.Unmarshal([]byte(response[0]), &scopes); err != nil {
			return nil, fmt.Errorf("failed to parse scopes response: %w", err)
		}
	}

	ss.logger.WithField("scope_count", len(scopes)).Debug("Retrieved state scopes")
	return scopes, nil
}

// GetScopeData retrieves all key-value pairs for a specific scope.
func (ss *StateService) GetScopeData(ctx context.Context, userToken, scope string) ([]StateData, error) {
	ss.logger.WithField("scope", scope).Debug("Getting scope data")

	// Send authenticated request to state service
	response, err := ss.sendAuthenticatedRequest(ctx, userToken, "get_scope_data", scope)
	if err != nil {
		return nil, fmt.Errorf("failed to get scope data: %w", err)
	}

	var data []StateData
	if len(response) > 0 && response[0] != "" {
		if err := json.Unmarshal([]byte(response[0]), &data); err != nil {
			return nil, fmt.Errorf("failed to parse scope data response: %w", err)
		}
	}

	ss.logger.WithFields(log.Fields{
		"scope":     scope,
		"key_count": len(data),
	}).Debug("Retrieved scope data")
	return data, nil
}

// GetStateValue retrieves a specific state value.
func (ss *StateService) GetStateValue(ctx context.Context, userToken, scope, key string) (*StateData, error) {
	ss.logger.WithFields(log.Fields{
		"scope": scope,
		"key":   key,
	}).Debug("Getting state value")

	// Send authenticated request to state service
	response, err := ss.sendAuthenticatedRequest(ctx, userToken, "get_value", scope, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get state value: %w", err)
	}

	if len(response) == 0 || response[0] == "" {
		return nil, fmt.Errorf("state value not found")
	}

	var data StateData
	if err := json.Unmarshal([]byte(response[0]), &data); err != nil {
		return nil, fmt.Errorf("failed to parse state value response: %w", err)
	}

	ss.logger.WithFields(log.Fields{
		"scope": scope,
		"key":   key,
		"type":  data.Type,
	}).Debug("Retrieved state value")
	return &data, nil
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

	// Send authenticated request to state service
	response, err := ss.sendAuthenticatedRequest(ctx, userToken, "set_value", scope, key, string(valueBytes))
	if err != nil {
		return fmt.Errorf("failed to set state value: %w", err)
	}

	// Check response for success
	if len(response) > 0 && response[0] == "error" {
		errorMsg := "unknown error"
		if len(response) > 1 {
			errorMsg = response[1]
		}
		return fmt.Errorf("state service error: %s", errorMsg)
	}

	ss.logger.WithFields(log.Fields{
		"scope": scope,
		"key":   key,
	}).Info("State value set successfully")
	return nil
}

// DeleteStateValue deletes a state value with authentication.
func (ss *StateService) DeleteStateValue(ctx context.Context, userToken, scope, key string) error {
	ss.logger.WithFields(log.Fields{
		"scope": scope,
		"key":   key,
	}).Debug("Deleting state value")

	// Send authenticated request to state service
	response, err := ss.sendAuthenticatedRequest(ctx, userToken, "delete_value", scope, key)
	if err != nil {
		return fmt.Errorf("failed to delete state value: %w", err)
	}

	// Check response for success
	if len(response) > 0 && response[0] == "error" {
		errorMsg := "unknown error"
		if len(response) > 1 {
			errorMsg = response[1]
		}
		return fmt.Errorf("state service error: %s", errorMsg)
	}

	ss.logger.WithFields(log.Fields{
		"scope": scope,
		"key":   key,
	}).Info("State value deleted successfully")
	return nil
}

// CreateBackup creates a backup of state data.
func (ss *StateService) CreateBackup(ctx context.Context, userToken, name, description string, scopes []string) (*StateBackup, error) {
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

	// Send authenticated request to state service
	response, err := ss.sendAuthenticatedRequest(ctx, userToken, "create_backup", name, description, string(scopesBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}

	if len(response) == 0 || response[0] == "" {
		return nil, fmt.Errorf("empty backup response")
	}

	var backup StateBackup
	if err := json.Unmarshal([]byte(response[0]), &backup); err != nil {
		return nil, fmt.Errorf("failed to parse backup response: %w", err)
	}

	ss.logger.WithFields(log.Fields{
		"backup_id": backup.ID,
		"name":      name,
		"size":      backup.Size,
	}).Info("State backup created successfully")
	return &backup, nil
}

// RestoreBackup restores state data from a backup.
func (ss *StateService) RestoreBackup(ctx context.Context, userToken, backupID string) error {
	ss.logger.WithField("backup_id", backupID).Debug("Restoring state backup")

	// Send authenticated request to state service
	response, err := ss.sendAuthenticatedRequest(ctx, userToken, "restore_backup", backupID)
	if err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	// Check response for success
	if len(response) > 0 && response[0] == "error" {
		errorMsg := "unknown error"
		if len(response) > 1 {
			errorMsg = response[1]
		}
		return fmt.Errorf("state service error: %s", errorMsg)
	}

	ss.logger.WithField("backup_id", backupID).Info("State backup restored successfully")
	return nil
}

// ListBackups retrieves a list of available backups.
func (ss *StateService) ListBackups(ctx context.Context, userToken string) ([]StateBackup, error) {
	ss.logger.Debug("Listing state backups")

	// Send authenticated request to state service
	response, err := ss.sendAuthenticatedRequest(ctx, userToken, "list_backups")
	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}

	var backups []StateBackup
	if len(response) > 0 && response[0] != "" {
		if err := json.Unmarshal([]byte(response[0]), &backups); err != nil {
			return nil, fmt.Errorf("failed to parse backups response: %w", err)
		}
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

	// Send authenticated request to state service
	response, err := ss.sendAuthenticatedRequest(ctx, userToken, "validate_data", scope, key, string(valueBytes))
	if err != nil {
		return false, nil, fmt.Errorf("failed to validate state data: %w", err)
	}

	if len(response) < 2 {
		return false, nil, fmt.Errorf("invalid validation response")
	}

	// Parse validation result
	isValid := response[0] == "true"
	var errors []string
	if len(response) > 1 && response[1] != "" {
		if err := json.Unmarshal([]byte(response[1]), &errors); err != nil {
			errors = []string{response[1]} // Fallback to single error string
		}
	}

	ss.logger.WithFields(log.Fields{
		"scope":  scope,
		"key":    key,
		"valid":  isValid,
		"errors": errors,
	}).Debug("State data validation complete")

	return isValid, errors, nil
}

// SubscribeToChanges subscribes to state change notifications.
func (ss *StateService) SubscribeToChanges(ctx context.Context, userToken string, scopes []string) (<-chan StateChangeNotification, error) {
	ss.logger.WithField("scopes", scopes).Debug("Subscribing to state changes")

	// Create notification channel
	notifications := make(chan StateChangeNotification, 100)

	// For now, this is a placeholder implementation
	// In a real implementation, this would establish a persistent connection
	// or use pub/sub for real-time notifications
	go func() {
		defer close(notifications)
		// Placeholder: would implement real subscription logic here
		select {
		case <-ctx.Done():
			return
		}
	}()

	return notifications, nil
}

// CheckConnectivity verifies connectivity to the state service.
func (ss *StateService) CheckConnectivity(ctx context.Context) error {
	ss.logger.Debug("Checking state service connectivity")

	// Try to ping the state service
	response, err := ss.sendRequest(ctx, "org.plantd.State", "ping")
	if err != nil {
		return fmt.Errorf("state service connectivity check failed: %w", err)
	}

	if len(response) == 0 || response[0] != "pong" {
		return fmt.Errorf("state service returned invalid ping response: %v", response)
	}

	ss.logger.Debug("State service connectivity check successful")
	return nil
}

// sendAuthenticatedRequest sends an authenticated request to the state service.
func (ss *StateService) sendAuthenticatedRequest(ctx context.Context, userToken, command string, args ...string) ([]string, error) {
	// Prepare message with authentication token
	message := append([]string{command, userToken}, args...)

	return ss.sendRequest(ctx, "org.plantd.State", message...)
}

// sendRequest sends a request to the state service.
func (ss *StateService) sendRequest(ctx context.Context, service string, args ...string) ([]string, error) {
	command := "<no_command>"
	if len(args) > 0 {
		command = args[0]
	}

	ss.logger.WithFields(log.Fields{
		"service": service,
		"command": command,
		"args":    len(args),
	}).Trace("Sending state service request")

	// Send request
	err := ss.client.Send(service, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to send state service request: %w", err)
	}

	// Receive response
	response, err := ss.client.Recv()
	if err != nil {
		return nil, fmt.Errorf("failed to receive state service response: %w", err)
	}

	ss.logger.WithFields(log.Fields{
		"service":  service,
		"command":  command,
		"response": response,
	}).Trace("Received state service response")

	return response, nil
}

// HealthCheck performs a comprehensive health check of the state service.
func (ss *StateService) HealthCheck(ctx context.Context) (map[string]interface{}, error) {
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
