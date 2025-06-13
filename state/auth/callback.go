// Package auth provides authentication and authorization functionality for the state service.
package auth

import (
	log "github.com/sirupsen/logrus"
)

const (
	listScopesMsgType  = "list-scopes"
	healthMsgType      = "health"
	createScopeMsgType = "create-scope"
	deleteScopeMsgType = "delete-scope"
	setMsgType         = "set"
	getMsgType         = "get"
	deleteMsgType      = "delete"
	listKeysMsgType    = "list-keys"
)

// AuthenticatedCallback wraps existing callbacks with authentication.
type AuthenticatedCallback struct {
	underlying     interface{ Execute(string) ([]byte, error) } // Use interface directly to avoid circular import
	authMiddleware *AuthMiddleware
	msgType        string
	name           string
}

// NewAuthenticatedCallback creates a new authenticated callback wrapper.
func NewAuthenticatedCallback(
	underlying interface{ Execute(string) ([]byte, error) },
	authMiddleware *AuthMiddleware,
	msgType string,
	name string,
) *AuthenticatedCallback {
	return &AuthenticatedCallback{
		underlying:     underlying,
		authMiddleware: authMiddleware,
		msgType:        msgType,
		name:           name,
	}
}

// Execute performs authentication before calling the underlying callback.
func (ac *AuthenticatedCallback) Execute(msgBody string) ([]byte, error) {
	// Parse authenticated request
	authRequest, err := NewAuthenticatedRequest(msgBody)
	if err != nil {
		log.WithFields(log.Fields{
			"callback": ac.name,
			"error":    err,
		}).Error("Failed to parse authenticated request")
		return CreateErrorResponse(err), err
	}

	// Get service scope for permission checking
	// Some operations like list-scopes don't require a service scope
	scope, found := authRequest.GetService()
	if !found && ac.requiresServiceScope() {
		log.WithFields(log.Fields{
			"callback": ac.name,
		}).Error("Service scope missing from request")
		return CreateErrorResponse(ErrServiceMissing), ErrServiceMissing
	}

	// Use empty scope for global operations
	if !found {
		scope = ""
	}

	// Validate authentication and permissions
	userCtx, err := ac.authMiddleware.ValidateRequest(ac.msgType, authRequest.Token, scope)
	if err != nil {
		log.WithFields(log.Fields{
			"callback": ac.name,
			"scope":    scope,
			"msgType":  ac.msgType,
			"error":    err,
		}).Warn("Authentication failed")
		return CreateErrorResponse(ErrTokenInvalid), err
	}

	// Log authenticated operation for audit trail
	log.WithFields(log.Fields{
		"user_email": userCtx.UserEmail,
		"user_id":    userCtx.UserID,
		"operation":  ac.msgType,
		"callback":   ac.name,
		"scope":      scope,
	}).Info("Authenticated state operation")

	// Call the underlying callback with the original message
	// Note: We pass the original msgBody to maintain compatibility
	response, err := ac.underlying.Execute(msgBody)

	if err != nil {
		log.WithFields(log.Fields{
			"user_email": userCtx.UserEmail,
			"operation":  ac.msgType,
			"callback":   ac.name,
			"scope":      scope,
			"error":      err,
		}).Error("Operation failed after authentication")
	} else {
		log.WithFields(log.Fields{
			"user_email": userCtx.UserEmail,
			"operation":  ac.msgType,
			"callback":   ac.name,
			"scope":      scope,
		}).Debug("Operation completed successfully")
	}

	return response, err
}

// requiresServiceScope checks if the operation requires a service scope.
func (ac *AuthenticatedCallback) requiresServiceScope() bool {
	switch ac.msgType {
	case listScopesMsgType, healthMsgType:
		// Global operations that don't require a specific service scope
		return false
	default:
		// Most operations require a service scope
		return true
	}
}

// GetUserContext extracts user context from an authenticated request.
// This can be used by callbacks that need user information.
func (ac *AuthenticatedCallback) GetUserContext(msgBody string) (*UserContext, error) {
	authRequest, err := NewAuthenticatedRequest(msgBody)
	if err != nil {
		return nil, err
	}

	scope, found := authRequest.GetService()
	if !found {
		return nil, ErrServiceMissing
	}

	return ac.authMiddleware.ValidateRequest(ac.msgType, authRequest.Token, scope)
}

// UnauthenticatedCallback wraps callbacks that don't require authentication.
// This is for backward compatibility and testing.
type UnauthenticatedCallback struct {
	underlying interface{ Execute(string) ([]byte, error) }
	name       string
}

// NewUnauthenticatedCallback creates a wrapper for callbacks that skip authentication.
func NewUnauthenticatedCallback(underlying interface{ Execute(string) ([]byte, error) }, name string) *UnauthenticatedCallback {
	return &UnauthenticatedCallback{
		underlying: underlying,
		name:       name,
	}
}

// Execute calls the underlying callback without authentication.
func (uc *UnauthenticatedCallback) Execute(msgBody string) ([]byte, error) {
	log.WithFields(log.Fields{
		"callback": uc.name,
	}).Debug("Executing unauthenticated callback")

	return uc.underlying.Execute(msgBody)
}

// CreateAuthenticatedCallbacks wraps a map of callbacks with authentication.
func CreateAuthenticatedCallbacks(
	callbacks map[string]interface{ Execute(string) ([]byte, error) },
	authMiddleware *AuthMiddleware,
) map[string]interface{ Execute(string) ([]byte, error) } {
	authenticatedCallbacks := make(map[string]interface{ Execute(string) ([]byte, error) })

	for name, callback := range callbacks {
		// Map callback names to message types for permission checking
		msgType := mapCallbackNameToMsgType(name)
		authenticatedCallbacks[name] = NewAuthenticatedCallback(
			callback,
			authMiddleware,
			msgType,
			name,
		)
	}

	return authenticatedCallbacks
}

// mapCallbackNameToMsgType maps callback names to message types for permission checking.
func mapCallbackNameToMsgType(callbackName string) string {
	switch callbackName {
	case createScopeMsgType:
		return createScopeMsgType
	case deleteScopeMsgType:
		return deleteScopeMsgType
	case setMsgType:
		return setMsgType
	case getMsgType:
		return getMsgType
	case deleteMsgType:
		return deleteMsgType
	case listScopesMsgType:
		return listScopesMsgType
	case listKeysMsgType:
		return listKeysMsgType
	default:
		return callbackName
	}
}

// ValidateAuthenticatedRequest validates that a request contains required authentication.
func ValidateAuthenticatedRequest(msgBody string) error {
	authRequest, err := NewAuthenticatedRequest(msgBody)
	if err != nil {
		return err
	}

	if authRequest.Token == "" {
		return ErrTokenRequired
	}

	if _, found := authRequest.GetService(); !found {
		return ErrServiceMissing
	}

	return nil
}
