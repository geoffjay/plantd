package auth

import (
	"encoding/json"

	"github.com/geoffjay/plantd/core/service"
)

// AuthenticatedRequest extends RawRequest to include authentication token.
type AuthenticatedRequest struct {
	service.RawRequest
	Token string `json:"token"`
}

// NewAuthenticatedRequest creates a new authenticated request from raw message body.
func NewAuthenticatedRequest(msgBody string) (*AuthenticatedRequest, error) {
	request := &AuthenticatedRequest{
		RawRequest: make(service.RawRequest),
	}

	// Unmarshal JSON into the map
	if err := json.Unmarshal([]byte(msgBody), &request.RawRequest); err != nil {
		return nil, err
	}

	// Extract token from the request
	if token, found := request.RawRequest["token"]; found {
		if tokenStr, ok := token.(string); ok {
			request.Token = tokenStr
		}
	}

	return request, nil
}

// GetService returns the service scope from the request.
func (ar *AuthenticatedRequest) GetService() (string, bool) {
	if service, found := ar.RawRequest["service"]; found {
		if serviceStr, ok := service.(string); ok {
			return serviceStr, true
		}
	}
	return "", false
}

// GetKey returns the key from the request.
func (ar *AuthenticatedRequest) GetKey() (string, bool) {
	if key, found := ar.RawRequest["key"]; found {
		if keyStr, ok := key.(string); ok {
			return keyStr, true
		}
	}
	return "", false
}

// GetValue returns the value from the request.
func (ar *AuthenticatedRequest) GetValue() (string, bool) {
	if value, found := ar.RawRequest["value"]; found {
		if valueStr, ok := value.(string); ok {
			return valueStr, true
		}
	}
	return "", false
}

// AuthenticationError represents authentication-related errors.
type AuthenticationError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// Error implements the error interface.
func (ae *AuthenticationError) Error() string {
	if ae.Detail != "" {
		return ae.Message + ": " + ae.Detail
	}
	return ae.Message
}

// Common authentication error types.
var (
	ErrTokenRequired = &AuthenticationError{
		Code:    "AUTHENTICATION_REQUIRED",
		Message: "Authentication token required",
	}

	ErrTokenInvalid = &AuthenticationError{
		Code:    "AUTHENTICATION_FAILED",
		Message: "Invalid or expired token",
	}

	ErrPermissionDenied = &AuthenticationError{
		Code:    "PERMISSION_DENIED",
		Message: "Insufficient permissions",
	}

	ErrServiceMissing = &AuthenticationError{
		Code:    "SERVICE_REQUIRED",
		Message: "Service scope required",
	}
)

// CreateErrorResponse creates a standardized error response.
func CreateErrorResponse(err error) []byte {
	var authErr *AuthenticationError

	// Check if it's already an AuthenticationError
	if e, ok := err.(*AuthenticationError); ok {
		authErr = e
	} else {
		// Wrap generic errors
		authErr = &AuthenticationError{
			Code:    "REQUEST_FAILED",
			Message: err.Error(),
		}
	}

	// Simple JSON response without full marshaling for performance
	response := `{"error": "` + authErr.Code + `", "message": "` + authErr.Message + `"`
	if authErr.Detail != "" {
		response += `, "detail": "` + authErr.Detail + `"`
	}
	response += `}`

	return []byte(response)
}
