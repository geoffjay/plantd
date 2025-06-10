// Package main provides callback handlers for the PlantD state service.
package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/geoffjay/plantd/core/service"

	log "github.com/sirupsen/logrus"
)

// Response represents a standardized callback response.
type Response struct {
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// createSuccessResponse creates a success response with optional data.
func createSuccessResponse(data interface{}) []byte {
	response := Response{
		Success: true,
		Data:    data,
	}
	bytes, _ := json.Marshal(response)
	return bytes
}

// createErrorResponse creates an error response with a message.
func createErrorResponse(message string) []byte {
	response := Response{
		Success: false,
		Error:   message,
	}
	bytes, _ := json.Marshal(response)
	return bytes
}

type createScopeCallback struct {
	name    string
	store   *Store
	manager *Manager
}

type deleteScopeCallback struct {
	name    string
	store   *Store
	manager *Manager
}

type deleteCallback struct {
	name  string
	store *Store
}

type getCallback struct {
	name  string
	store *Store
}

type setCallback struct {
	name  string
	store *Store
}

type sinkCallback struct {
	store *Store
}

type healthCallback struct {
	name  string
	store *Store
}

// Execute callback function to handle `create-scope` requests.
func (cb *createScopeCallback) Execute(msgBody string) ([]byte, error) {
	var (
		scope   string
		found   bool
		request service.RawRequest
	)

	log.WithFields(log.Fields{
		"callback":  cb.name,
		"operation": "create-scope",
	}).Debug("Processing create-scope request")

	if err := json.Unmarshal([]byte(msgBody), &request); err != nil {
		log.WithFields(log.Fields{
			"callback": cb.name,
			"error":    err,
		}).Error("Failed to parse request JSON")
		return createErrorResponse("Invalid request format: " + err.Error()), err
	}

	if scope, found = request["service"].(string); !found {
		err := errors.New("service parameter missing")
		log.WithFields(log.Fields{
			"callback": cb.name,
		}).Error("Service scope missing from request")
		return createErrorResponse("Service scope required for create-scope request"), err
	}

	// Validate scope name
	if scope == "" {
		err := errors.New("empty service scope")
		log.WithFields(log.Fields{
			"callback": cb.name,
		}).Error("Empty service scope provided")
		return createErrorResponse("Service scope cannot be empty"), err
	}

	if cb.store.HasScope(scope) {
		log.WithFields(log.Fields{
			"callback": cb.name,
			"scope":    scope,
		}).Warn("Attempted to create existing scope")
		return createErrorResponse(fmt.Sprintf("Scope '%s' already exists", scope)), nil
	}

	err := cb.store.CreateScope(scope)
	if err != nil {
		log.WithFields(log.Fields{
			"callback": cb.name,
			"scope":    scope,
			"error":    err,
		}).Error("Failed to create scope in store")
		return createErrorResponse("Failed to create scope: " + err.Error()), err
	}

	// Add a sink to listen for events on the new scope
	cb.manager.AddSink(scope, &sinkCallback{store: cb.store})

	log.WithFields(log.Fields{
		"callback": cb.name,
		"scope":    scope,
	}).Info("Successfully created scope")

	return createSuccessResponse(map[string]string{
		"scope":  scope,
		"status": "created",
	}), nil
}

// Execute callback function to handle `delete-scope` requests.
func (cb *deleteScopeCallback) Execute(msgBody string) ([]byte, error) {
	var (
		scope   string
		found   bool
		request service.RawRequest
	)

	log.WithFields(log.Fields{
		"callback":  cb.name,
		"operation": "delete-scope",
	}).Debug("Processing delete-scope request")

	if err := json.Unmarshal([]byte(msgBody), &request); err != nil {
		log.WithFields(log.Fields{
			"callback": cb.name,
			"error":    err,
		}).Error("Failed to parse request JSON")
		return createErrorResponse("Invalid request format: " + err.Error()), err
	}

	if scope, found = request["service"].(string); !found {
		err := errors.New("service parameter missing")
		log.WithFields(log.Fields{
			"callback": cb.name,
		}).Error("Service scope missing from request")
		return createErrorResponse("Service scope required for delete-scope request"), err
	}

	if !cb.store.HasScope(scope) {
		log.WithFields(log.Fields{
			"callback": cb.name,
			"scope":    scope,
		}).Warn("Attempted to delete non-existent scope")
		return createErrorResponse(fmt.Sprintf("Scope '%s' does not exist", scope)), nil
	}

	err := cb.store.DeleteScope(scope)
	if err != nil {
		log.WithFields(log.Fields{
			"callback": cb.name,
			"scope":    scope,
			"error":    err,
		}).Error("Failed to delete scope from store")
		return createErrorResponse("Failed to delete scope: " + err.Error()), err
	}

	// Remove the sink for this scope
	cb.manager.RemoveSink(scope)

	log.WithFields(log.Fields{
		"callback": cb.name,
		"scope":    scope,
	}).Info("Successfully deleted scope")

	return createSuccessResponse(map[string]string{
		"scope":  scope,
		"status": "deleted",
	}), nil
}

// Execute callback function to handle `delete` requests.
func (cb *deleteCallback) Execute(msgBody string) ([]byte, error) {
	var (
		scope   string
		key     string
		found   bool
		request service.RawRequest
	)

	log.WithFields(log.Fields{
		"callback":  cb.name,
		"operation": "delete",
	}).Debug("Processing delete request")

	if err := json.Unmarshal([]byte(msgBody), &request); err != nil {
		log.WithFields(log.Fields{
			"callback": cb.name,
			"error":    err,
		}).Error("Failed to parse request JSON")
		return createErrorResponse("Invalid request format: " + err.Error()), err
	}

	if scope, found = request["service"].(string); !found {
		err := errors.New("service parameter missing")
		log.WithFields(log.Fields{
			"callback": cb.name,
		}).Error("Service scope missing from request")
		return createErrorResponse("Service scope required for delete request"), err
	}

	if key, found = request["key"].(string); !found {
		err := errors.New("key parameter missing")
		log.WithFields(log.Fields{
			"callback": cb.name,
			"scope":    scope,
		}).Error("Key missing from request")
		return createErrorResponse("Key required for delete request"), err
	}

	// Validate inputs
	if scope == "" {
		return createErrorResponse("Service scope cannot be empty"), errors.New("empty service scope")
	}
	if key == "" {
		return createErrorResponse("Key cannot be empty"), errors.New("empty key")
	}

	err := cb.store.Delete(scope, key)
	if err != nil {
		log.WithFields(log.Fields{
			"callback": cb.name,
			"scope":    scope,
			"key":      key,
			"error":    err,
		}).Error("Failed to delete key from store")
		return createErrorResponse("Failed to delete key: " + err.Error()), err
	}

	log.WithFields(log.Fields{
		"callback": cb.name,
		"scope":    scope,
		"key":      key,
	}).Info("Successfully deleted key")

	return createSuccessResponse(map[string]string{
		"scope":  scope,
		"key":    key,
		"status": "deleted",
	}), nil
}

// Execute callback function to handle `get` requests.
func (cb *getCallback) Execute(msgBody string) ([]byte, error) {
	var (
		scope   string
		key     string
		found   bool
		request service.RawRequest
	)

	log.WithFields(log.Fields{
		"callback":  cb.name,
		"operation": "get",
	}).Debug("Processing get request")

	if err := json.Unmarshal([]byte(msgBody), &request); err != nil {
		log.WithFields(log.Fields{
			"callback": cb.name,
			"error":    err,
		}).Error("Failed to parse request JSON")
		return createErrorResponse("Invalid request format: " + err.Error()), err
	}

	if scope, found = request["service"].(string); !found {
		err := errors.New("service parameter missing")
		log.WithFields(log.Fields{
			"callback": cb.name,
		}).Error("Service scope missing from request")
		return createErrorResponse("Service scope required for get request"), err
	}

	if key, found = request["key"].(string); !found {
		err := errors.New("key parameter missing")
		log.WithFields(log.Fields{
			"callback": cb.name,
			"scope":    scope,
		}).Error("Key missing from request")
		return createErrorResponse("Key required for get request"), err
	}

	// Validate inputs
	if scope == "" {
		return createErrorResponse("Service scope cannot be empty"), errors.New("empty service scope")
	}
	if key == "" {
		return createErrorResponse("Key cannot be empty"), errors.New("empty key")
	}

	value, err := cb.store.Get(scope, key)
	if err != nil {
		log.WithFields(log.Fields{
			"callback": cb.name,
			"scope":    scope,
			"key":      key,
			"error":    err,
		}).Error("Failed to get key from store")
		return createErrorResponse("Failed to get key: " + err.Error()), err
	}

	log.WithFields(log.Fields{
		"callback": cb.name,
		"scope":    scope,
		"key":      key,
	}).Debug("Successfully retrieved key")

	return createSuccessResponse(map[string]string{
		"scope": scope,
		"key":   key,
		"value": value,
	}), nil
}

// Execute callback function to handle `set` requests.
func (cb *setCallback) Execute(msgBody string) ([]byte, error) {
	var (
		scope   string
		key     string
		value   string
		found   bool
		request service.RawRequest
	)

	log.WithFields(log.Fields{
		"callback":  cb.name,
		"operation": "set",
	}).Debug("Processing set request")

	if err := json.Unmarshal([]byte(msgBody), &request); err != nil {
		log.WithFields(log.Fields{
			"callback": cb.name,
			"error":    err,
		}).Error("Failed to parse request JSON")
		return createErrorResponse("Invalid request format: " + err.Error()), err
	}

	if scope, found = request["service"].(string); !found {
		err := errors.New("service parameter missing")
		log.WithFields(log.Fields{
			"callback": cb.name,
		}).Error("Service scope missing from request")
		return createErrorResponse("Service scope required for set request"), err
	}

	if key, found = request["key"].(string); !found {
		err := errors.New("key parameter missing")
		log.WithFields(log.Fields{
			"callback": cb.name,
			"scope":    scope,
		}).Error("Key missing from request")
		return createErrorResponse("Key required for set request"), err
	}

	if value, found = request["value"].(string); !found {
		err := errors.New("value parameter missing")
		log.WithFields(log.Fields{
			"callback": cb.name,
			"scope":    scope,
			"key":      key,
		}).Error("Value missing from request")
		return createErrorResponse("Value required for set request"), err
	}

	// Validate inputs
	if scope == "" {
		return createErrorResponse("Service scope cannot be empty"), errors.New("empty service scope")
	}
	if key == "" {
		return createErrorResponse("Key cannot be empty"), errors.New("empty key")
	}

	err := cb.store.Set(scope, key, value)
	if err != nil {
		log.WithFields(log.Fields{
			"callback": cb.name,
			"scope":    scope,
			"key":      key,
			"error":    err,
		}).Error("Failed to set key in store")
		return createErrorResponse("Failed to set key: " + err.Error()), err
	}

	log.WithFields(log.Fields{
		"callback":     cb.name,
		"scope":        scope,
		"key":          key,
		"value_length": len(value),
	}).Info("Successfully set key")

	return createSuccessResponse(map[string]string{
		"scope":  scope,
		"key":    key,
		"value":  value,
		"status": "set",
	}), nil
}

// Handle callback handles subscriber events on the state bus.
func (cb *sinkCallback) Handle(data []byte) error {
	log.WithFields(log.Fields{
		"data_length":  len(data),
		"data_preview": string(data)[:min(len(data), 100)], // First 100 chars
	}).Debug("Data received on state bus")
	return nil
}

// Execute callback function to handle `health` requests.
func (cb *healthCallback) Execute(msgBody string) ([]byte, error) {
	var request service.RawRequest

	log.WithFields(log.Fields{
		"callback":  cb.name,
		"operation": "health",
	}).Debug("Processing health check request")

	if err := json.Unmarshal([]byte(msgBody), &request); err != nil {
		log.WithFields(log.Fields{
			"callback": cb.name,
			"error":    err,
		}).Error("Failed to parse health check request JSON")
		return createErrorResponse("Invalid request format: " + err.Error()), err
	}

	// Health check doesn't require service scope, but if provided, we'll include it in response
	scope, _ := request["service"].(string)

	// Perform basic health checks
	healthData := map[string]interface{}{
		"status":          "healthy",
		"service":         "org.plantd.State",
		"store_available": cb.store != nil,
	}

	if scope != "" {
		healthData["scope"] = scope
		healthData["scope_exists"] = cb.store.HasScope(scope)
	}

	log.WithFields(log.Fields{
		"callback": cb.name,
		"scope":    scope,
	}).Debug("Health check completed successfully")

	return createSuccessResponse(healthData), nil
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
