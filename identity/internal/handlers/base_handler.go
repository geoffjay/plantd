// Package handlers provides MDP protocol handlers for the identity service.
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
)

// Handler interface defines the contract for MDP message handlers.
type Handler interface {
	HandleMessage(ctx context.Context, message []string) ([]string, error)
	GetServiceName() string
}

// BaseHandler provides common functionality for all handlers.
type BaseHandler struct {
	serviceName string
	validator   *validator.Validate
	logger      *logrus.Logger
}

// NewBaseHandler creates a new base handler.
func NewBaseHandler(serviceName string, logger *logrus.Logger) *BaseHandler {
	return &BaseHandler{
		serviceName: serviceName,
		validator:   validator.New(),
		logger:      logger,
	}
}

// GetServiceName returns the service name for this handler.
func (h *BaseHandler) GetServiceName() string {
	return h.serviceName
}

// ParseRequest parses and validates a JSON request.
func (h *BaseHandler) ParseRequest(data []byte, req interface{}) error {
	if err := json.Unmarshal(data, req); err != nil {
		return fmt.Errorf("failed to parse request JSON: %w", err)
	}

	if err := h.validator.Struct(req); err != nil {
		return fmt.Errorf("request validation failed: %w", err)
	}

	return nil
}

// CreateSuccessResponse creates a successful response with the given data.
func (h *BaseHandler) CreateSuccessResponse(requestID string, data interface{}) ([]byte, error) {
	response := map[string]interface{}{
		"header": ResponseHeader{
			RequestID: requestID,
			Success:   true,
			Timestamp: time.Now().Unix(),
		},
	}

	// Add data fields to response
	if data != nil {
		responseBytes, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response data: %w", err)
		}

		var responseMap map[string]interface{}
		if err := json.Unmarshal(responseBytes, &responseMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal response data: %w", err)
		}

		// Merge data into response
		for k, v := range responseMap {
			if k != "header" {
				response[k] = v
			}
		}
	}

	return json.Marshal(response)
}

// CreateErrorResponse creates an error response.
func (h *BaseHandler) CreateErrorResponse(requestID, errorCode, errorMessage, detail string) ([]byte, error) {
	response := ErrorResponse{
		Header: ResponseHeader{
			RequestID: requestID,
			Success:   false,
			Error:     errorMessage,
			Timestamp: time.Now().Unix(),
		},
		Code:   errorCode,
		Detail: detail,
	}

	return json.Marshal(response)
}

// LogRequest logs the incoming request.
func (h *BaseHandler) LogRequest(operation string, requestID string, userID *uint) {
	fields := logrus.Fields{
		"service":    h.serviceName,
		"operation":  operation,
		"request_id": requestID,
	}

	if userID != nil {
		fields["user_id"] = *userID
	}

	h.logger.WithFields(fields).Info("Processing MDP request")
}

// LogResponse logs the response.
func (h *BaseHandler) LogResponse(operation string, requestID string, success bool, err error) {
	fields := logrus.Fields{
		"service":    h.serviceName,
		"operation":  operation,
		"request_id": requestID,
		"success":    success,
	}

	if err != nil {
		fields["error"] = err.Error()
		h.logger.WithFields(fields).Error("MDP request failed")
	} else {
		h.logger.WithFields(fields).Info("MDP request completed")
	}
}

// ExtractRequestID extracts the request ID from a request structure.
func (h *BaseHandler) ExtractRequestID(req interface{}) string { //nolint:cyclop
	// Use reflection to get the RequestID from the Header field
	switch r := req.(type) {
	case *LoginRequest:
		return r.Header.RequestID
	case *RefreshTokenRequest:
		return r.Header.RequestID
	case *LogoutRequest:
		return r.Header.RequestID
	case *ValidateTokenRequest:
		return r.Header.RequestID
	case *CreateUserRequest:
		return r.Header.RequestID
	case *GetUserRequest:
		return r.Header.RequestID
	case *UpdateUserRequest:
		return r.Header.RequestID
	case *DeleteUserRequest:
		return r.Header.RequestID
	case *ListUsersRequest:
		return r.Header.RequestID
	case *CreateOrganizationRequest:
		return r.Header.RequestID
	case *GetOrganizationRequest:
		return r.Header.RequestID
	case *UpdateOrganizationRequest:
		return r.Header.RequestID
	case *DeleteOrganizationRequest:
		return r.Header.RequestID
	case *ListOrganizationsRequest:
		return r.Header.RequestID
	case *CreateRoleRequest:
		return r.Header.RequestID
	case *GetRoleRequest:
		return r.Header.RequestID
	case *UpdateRoleRequest:
		return r.Header.RequestID
	case *DeleteRoleRequest:
		return r.Header.RequestID
	case *ListRolesRequest:
		return r.Header.RequestID
	case *CheckPermissionRequest:
		return r.Header.RequestID
	case *AssignRoleRequest:
		return r.Header.RequestID
	case *UnassignRoleRequest:
		return r.Header.RequestID
	case *HealthCheckRequest:
		return r.Header.RequestID
	default:
		return unknownOperation
	}
}

// ExtractUserID extracts the user ID from a request structure.
func (h *BaseHandler) ExtractUserID(req interface{}) *uint { //nolint:cyclop
	// Use reflection to get the UserID from the Header field
	switch r := req.(type) {
	case *LoginRequest:
		return r.Header.UserID
	case *RefreshTokenRequest:
		return r.Header.UserID
	case *LogoutRequest:
		return r.Header.UserID
	case *ValidateTokenRequest:
		return r.Header.UserID
	case *CreateUserRequest:
		return r.Header.UserID
	case *GetUserRequest:
		return r.Header.UserID
	case *UpdateUserRequest:
		return r.Header.UserID
	case *DeleteUserRequest:
		return r.Header.UserID
	case *ListUsersRequest:
		return r.Header.UserID
	case *CreateOrganizationRequest:
		return r.Header.UserID
	case *GetOrganizationRequest:
		return r.Header.UserID
	case *UpdateOrganizationRequest:
		return r.Header.UserID
	case *DeleteOrganizationRequest:
		return r.Header.UserID
	case *ListOrganizationsRequest:
		return r.Header.UserID
	case *CreateRoleRequest:
		return r.Header.UserID
	case *GetRoleRequest:
		return r.Header.UserID
	case *UpdateRoleRequest:
		return r.Header.UserID
	case *DeleteRoleRequest:
		return r.Header.UserID
	case *ListRolesRequest:
		return r.Header.UserID
	case *CheckPermissionRequest:
		return r.Header.UserID
	case *AssignRoleRequest:
		return r.Header.UserID
	case *UnassignRoleRequest:
		return r.Header.UserID
	case *HealthCheckRequest:
		return r.Header.UserID
	default:
		return nil
	}
}

// HandlePanic recovers from panics and converts them to error responses.
func (h *BaseHandler) HandlePanic(requestID string) ([]byte, error) {
	if r := recover(); r != nil {
		h.logger.WithFields(logrus.Fields{
			"service":    h.serviceName,
			"request_id": requestID,
			"panic":      r,
		}).Error("Handler panic occurred")

		return h.CreateErrorResponse(requestID, "INTERNAL_ERROR", "Internal server error", fmt.Sprintf("Panic: %v", r))
	}
	return nil, nil
}
