package mdp

import (
	"errors"
	"fmt"
)

// Standard MDP errors
var (
	errPermanent = errors.New("permanent error, abandoning request")

	// Protocol-level errors
	ErrInvalidMessage       = errors.New("invalid message format")
	ErrProtocolViolation    = errors.New("protocol violation")
	ErrTimeout              = errors.New("operation timeout")
	ErrBrokerUnavailable    = errors.New("broker unavailable")
	ErrServiceNotFound      = errors.New("service not found")
	ErrWorkerDisconnected   = errors.New("worker disconnected")
	ErrClientDisconnected   = errors.New("client disconnected")
	ErrHeartbeatFailed      = errors.New("heartbeat failed")
	ErrConnectionFailed     = errors.New("connection failed")
	ErrSocketError          = errors.New("socket error")
	ErrMessageTooLarge      = errors.New("message too large")
	ErrInvalidService       = errors.New("invalid service name")
	ErrInvalidCommand       = errors.New("invalid command")
	ErrBrokerOverloaded     = errors.New("broker overloaded")
	ErrAuthenticationFailed = errors.New("authentication failed")
	ErrAuthorizationFailed  = errors.New("authorization failed")
)

// Error represents a structured MDP protocol error with context
type Error struct {
	Code    string
	Message string
	Cause   error
	Context map[string]interface{}
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("MDP %s: %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("MDP %s: %s", e.Code, e.Message)
}

// Unwrap implements error unwrapping for Go 1.13+ error handling
func (e *Error) Unwrap() error {
	return e.Cause
}

// Is implements error comparison for Go 1.13+ error handling
func (e *Error) Is(target error) bool {
	if target == nil {
		return false
	}

	if mdpErr, ok := target.(*Error); ok {
		return e.Code == mdpErr.Code
	}

	return errors.Is(e.Cause, target)
}

// WithContext adds context information to the error
func (e *Error) WithContext(key string, value interface{}) *Error {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// Error code constants for structured error handling
const (
	ErrCodeInvalidMessage     = "INVALID_MESSAGE"
	ErrCodeProtocolViolation  = "PROTOCOL_VIOLATION"
	ErrCodeTimeout            = "TIMEOUT"
	ErrCodeBrokerUnavailable  = "BROKER_UNAVAILABLE"
	ErrCodeServiceNotFound    = "SERVICE_NOT_FOUND"
	ErrCodeWorkerDisconnected = "WORKER_DISCONNECTED"
	ErrCodeClientDisconnected = "CLIENT_DISCONNECTED"
	ErrCodeHeartbeatFailed    = "HEARTBEAT_FAILED"
	ErrCodeConnectionFailed   = "CONNECTION_FAILED"
	ErrCodeSocketError        = "SOCKET_ERROR"
	ErrCodeMessageTooLarge    = "MESSAGE_TOO_LARGE"
	ErrCodeInvalidService     = "INVALID_SERVICE"
	ErrCodeInvalidCommand     = "INVALID_COMMAND"
	ErrCodeBrokerOverloaded   = "BROKER_OVERLOADED"
	ErrCodeAuthFailed         = "AUTH_FAILED"
	ErrCodeAuthzFailed        = "AUTHZ_FAILED"
)

// NewMDPError creates a new structured MDP error
func NewMDPError(code, message string, cause error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
	}
}

// NewInvalidMessageError creates a new invalid message error
func NewInvalidMessageError(message string, cause error) *Error {
	return NewMDPError(ErrCodeInvalidMessage, message, cause)
}

// NewProtocolViolationError creates a new protocol violation error
func NewProtocolViolationError(message string, cause error) *Error {
	return NewMDPError(ErrCodeProtocolViolation, message, cause)
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(message string, cause error) *Error {
	return NewMDPError(ErrCodeTimeout, message, cause)
}

// NewBrokerUnavailableError creates a new broker unavailable error
func NewBrokerUnavailableError(message string, cause error) *Error {
	return NewMDPError(ErrCodeBrokerUnavailable, message, cause)
}

// NewServiceNotFoundError creates a new service not found error
func NewServiceNotFoundError(service string, cause error) *Error {
	return NewMDPError(ErrCodeServiceNotFound, fmt.Sprintf("service '%s' not found", service), cause).
		WithContext("service", service)
}

// NewWorkerDisconnectedError creates a new worker disconnected error
func NewWorkerDisconnectedError(worker string, cause error) *Error {
	return NewMDPError(ErrCodeWorkerDisconnected, fmt.Sprintf("worker '%s' disconnected", worker), cause).
		WithContext("worker", worker)
}

// NewConnectionFailedError creates a new connection failed error
func NewConnectionFailedError(endpoint string, cause error) *Error {
	return NewMDPError(ErrCodeConnectionFailed, fmt.Sprintf("failed to connect to '%s'", endpoint), cause).
		WithContext("endpoint", endpoint)
}

// NewInvalidServiceError creates a new invalid service error
func NewInvalidServiceError(service string, cause error) *Error {
	return NewMDPError(ErrCodeInvalidService, fmt.Sprintf("invalid service: %s", service), cause).
		WithContext("service", service)
}

// IsRetryableError determines if an error condition is retryable
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	var mdpErr *Error
	if errors.As(err, &mdpErr) {
		switch mdpErr.Code {
		case ErrCodeTimeout, ErrCodeBrokerUnavailable, ErrCodeConnectionFailed, ErrCodeSocketError, ErrCodeWorkerDisconnected:
			return true
		default:
			return false
		}
	}

	// Check standard errors
	return errors.Is(err, ErrTimeout) ||
		errors.Is(err, ErrBrokerUnavailable) ||
		errors.Is(err, ErrConnectionFailed) ||
		errors.Is(err, ErrSocketError) ||
		errors.Is(err, ErrWorkerDisconnected)
}

// IsPermanentError determines if an error condition is permanent (non-retryable)
func IsPermanentError(err error) bool {
	if err == nil {
		return false
	}

	var mdpErr *Error
	if errors.As(err, &mdpErr) {
		switch mdpErr.Code {
		case ErrCodeProtocolViolation, ErrCodeInvalidMessage, ErrCodeInvalidService,
			ErrCodeInvalidCommand, ErrCodeAuthFailed, ErrCodeAuthzFailed:
			return true
		default:
			return false
		}
	}

	return errors.Is(err, errPermanent) ||
		errors.Is(err, ErrProtocolViolation) ||
		errors.Is(err, ErrInvalidMessage) ||
		errors.Is(err, ErrAuthenticationFailed) ||
		errors.Is(err, ErrAuthorizationFailed)
}
