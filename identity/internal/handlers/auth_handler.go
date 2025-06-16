package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/geoffjay/plantd/identity/internal/auth"
	"github.com/sirupsen/logrus"
)

// AuthHandler handles authentication-related MDP messages.
type AuthHandler struct {
	*BaseHandler
	authService *auth.AuthService
}

// NewAuthHandler creates a new authentication handler.
func NewAuthHandler(authService *auth.AuthService, logger *logrus.Logger) *AuthHandler {
	return &AuthHandler{
		BaseHandler: NewBaseHandler("identity.auth", logger),
		authService: authService,
	}
}

// HandleMessage handles incoming MDP messages for authentication operations.
func (h *AuthHandler) HandleMessage(ctx context.Context, message []string) ([]string, error) {
	h.logger.WithFields(logrus.Fields{
		"handler":        "auth",
		"message_length": len(message),
		"raw_message":    message,
	}).Debug("Auth handler received message")

	defer func() {
		if responseBytes, err := h.HandlePanic(unknownOperation); responseBytes != nil { //nolint:revive
			// Return the panic response
		} else if err != nil {
			h.logger.WithError(err).Error("Error handling panic")
		}
	}()

	if len(message) < 2 {
		h.logger.Warn("Auth handler: message too short, expected at least 2 parts")
		return h.createErrorMessage("", "INVALID_MESSAGE", "Message must contain operation and data", "")
	}

	operation := message[0]
	data := message[1]

	h.logger.WithFields(logrus.Fields{
		"operation": operation,
		"data_len":  len(data),
	}).Debug("Auth handler processing operation")

	switch operation {
	case "login":
		h.logger.Debug("Routing to handleLogin")
		return h.handleLogin(ctx, data)
	case "refresh":
		h.logger.Debug("Routing to handleRefreshToken")
		return h.handleRefreshToken(ctx, data)
	case "logout":
		h.logger.Debug("Routing to handleLogout")
		return h.handleLogout(ctx, data)
	case "validate":
		h.logger.Debug("Routing to handleValidateToken")
		return h.handleValidateToken(ctx, data)
	case "change_password":
		h.logger.Debug("Routing to handleChangePassword")
		return h.handleChangePassword(ctx, data)
	default:
		h.logger.WithField("operation", operation).Warn("Unknown operation in auth handler")
		return h.createErrorMessage("", "UNKNOWN_OPERATION", fmt.Sprintf("Unknown operation: %s", operation), "")
	}
}

// handleLogin processes login requests.
func (h *AuthHandler) handleLogin(ctx context.Context, data string) ([]string, error) {
	h.logger.WithFields(logrus.Fields{
		"method":   "handleLogin",
		"data_len": len(data),
	}).Debug("Starting login processing")

	var req LoginRequest
	if err := h.ParseRequest([]byte(data), &req); err != nil {
		h.logger.WithError(err).Error("Failed to parse login request")
		return h.createErrorMessage(req.Header.RequestID, "INVALID_REQUEST", err.Error(), "")
	}

	h.logger.WithFields(logrus.Fields{
		"identifier": req.Identifier,
		"request_id": req.Header.RequestID,
	}).Debug("Parsed login request")

	requestID := h.ExtractRequestID(&req)
	userID := h.ExtractUserID(&req)
	h.LogRequest("login", requestID, userID)

	// Convert to auth service request
	authReq := &auth.AuthRequest{
		Identifier: req.Identifier,
		Password:   req.Password,
		IPAddress:  req.IPAddress,
		UserAgent:  req.UserAgent,
	}

	h.logger.WithField("request_id", requestID).Debug("Calling auth service login")

	// Call auth service
	authResp, err := h.authService.Login(ctx, authReq)
	if err != nil {
		h.logger.WithError(err).Error("Auth service login failed")
		h.LogResponse("login", requestID, false, err)
		return h.createErrorMessage(requestID, "LOGIN_FAILED", err.Error(), "")
	}

	h.logger.WithFields(logrus.Fields{
		"request_id": requestID,
		"user_email": authResp.User.Email,
	}).Debug("Auth service login successful, creating response")

	// Create response
	response := LoginResponse{
		Header: ResponseHeader{
			RequestID: requestID,
			Success:   true,
			Timestamp: time.Now().Unix(),
		},
		User:         authResp.User,
		AccessToken:  authResp.TokenPair.AccessToken,
		RefreshToken: authResp.TokenPair.RefreshToken,
		ExpiresAt:    authResp.ExpiresAt.Unix(),
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		h.logger.WithError(err).Error("Failed to marshal login response")
		h.LogResponse("login", requestID, false, err)
		return h.createErrorMessage(requestID, "RESPONSE_ERROR", err.Error(), "")
	}

	h.logger.WithFields(logrus.Fields{
		"request_id":     requestID,
		"response_bytes": len(responseBytes),
	}).Debug("Login response created successfully")

	h.LogResponse("login", requestID, true, nil)
	return []string{string(responseBytes)}, nil
}

// handleRefreshToken processes token refresh requests.
func (h *AuthHandler) handleRefreshToken(ctx context.Context, data string) ([]string, error) {
	var req RefreshTokenRequest
	if err := h.ParseRequest([]byte(data), &req); err != nil {
		return h.createErrorMessage(req.Header.RequestID, "INVALID_REQUEST", err.Error(), "")
	}

	requestID := h.ExtractRequestID(&req)
	userID := h.ExtractUserID(&req)
	h.LogRequest("refresh_token", requestID, userID)

	// Convert to auth service request
	refreshReq := &auth.RefreshRequest{
		RefreshToken: req.RefreshToken,
		IPAddress:    req.IPAddress,
	}

	// Call auth service
	tokenPair, err := h.authService.RefreshToken(ctx, refreshReq)
	if err != nil {
		h.LogResponse("refresh_token", requestID, false, err)
		return h.createErrorMessage(requestID, "REFRESH_FAILED", err.Error(), "")
	}

	// Create response
	response := RefreshTokenResponse{
		Header: ResponseHeader{
			RequestID: requestID,
			Success:   true,
			Timestamp: time.Now().Unix(),
		},
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.AccessTokenExpiresAt.Unix(),
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		h.LogResponse("refresh_token", requestID, false, err)
		return h.createErrorMessage(requestID, "RESPONSE_ERROR", err.Error(), "")
	}

	h.LogResponse("refresh_token", requestID, true, nil)
	return []string{string(responseBytes)}, nil
}

// handleLogout processes logout requests.
func (h *AuthHandler) handleLogout(ctx context.Context, data string) ([]string, error) {
	var req LogoutRequest
	if err := h.ParseRequest([]byte(data), &req); err != nil {
		return h.createErrorMessage(req.Header.RequestID, "INVALID_REQUEST", err.Error(), "")
	}

	requestID := h.ExtractRequestID(&req)
	userID := h.ExtractUserID(&req)
	h.LogRequest("logout", requestID, userID)

	// Call auth service
	err := h.authService.Logout(ctx, req.AccessToken)
	if err != nil {
		h.LogResponse("logout", requestID, false, err)
		return h.createErrorMessage(requestID, "LOGOUT_FAILED", err.Error(), "")
	}

	// Create response
	response := LogoutResponse{
		Header: ResponseHeader{
			RequestID: requestID,
			Success:   true,
			Timestamp: time.Now().Unix(),
		},
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		h.LogResponse("logout", requestID, false, err)
		return h.createErrorMessage(requestID, "RESPONSE_ERROR", err.Error(), "")
	}

	h.LogResponse("logout", requestID, true, nil)
	return []string{string(responseBytes)}, nil
}

// handleValidateToken processes token validation requests.
func (h *AuthHandler) handleValidateToken(ctx context.Context, data string) ([]string, error) {
	var req ValidateTokenRequest
	if err := h.ParseRequest([]byte(data), &req); err != nil {
		return h.createErrorMessage(req.Header.RequestID, "INVALID_REQUEST", err.Error(), "")
	}

	requestID := h.ExtractRequestID(&req)
	userID := h.ExtractUserID(&req)
	h.LogRequest("validate_token", requestID, userID)

	// Call auth service
	claims, err := h.authService.ValidateToken(ctx, req.Token)
	if err != nil {
		// For validation, we return success=false rather than an error
		response := ValidateTokenResponse{
			Header: ResponseHeader{
				RequestID: requestID,
				Success:   true,
				Timestamp: time.Now().Unix(),
			},
			Valid: false,
		}

		responseBytes, err := json.Marshal(response)
		if err != nil {
			h.LogResponse("validate_token", requestID, false, err)
			return h.createErrorMessage(requestID, "RESPONSE_ERROR", err.Error(), "")
		}

		h.LogResponse("validate_token", requestID, true, nil)
		return []string{string(responseBytes)}, nil
	}

	// Token is valid
	expiresAt := claims.ExpiresAt.Unix()
	response := ValidateTokenResponse{
		Header: ResponseHeader{
			RequestID: requestID,
			Success:   true,
			Timestamp: time.Now().Unix(),
		},
		Valid:       true,
		UserID:      &claims.UserID,
		Email:       claims.Email,
		Roles:       claims.Roles,
		Permissions: claims.Permissions,
		ExpiresAt:   &expiresAt,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		h.LogResponse("validate_token", requestID, false, err)
		return h.createErrorMessage(requestID, "RESPONSE_ERROR", err.Error(), "")
	}

	h.LogResponse("validate_token", requestID, true, nil)
	return []string{string(responseBytes)}, nil
}

// ChangePasswordRequest represents a password change request.
type ChangePasswordRequest struct {
	Header          RequestHeader `json:"header"`
	UserID          uint          `json:"user_id" validate:"required"`
	CurrentPassword string        `json:"current_password" validate:"required"`
	NewPassword     string        `json:"new_password" validate:"required,min=8"`
}

// ChangePasswordResponse represents a password change response.
type ChangePasswordResponse struct {
	Header ResponseHeader `json:"header"`
}

// handleChangePassword processes password change requests.
func (h *AuthHandler) handleChangePassword(ctx context.Context, data string) ([]string, error) {
	var req ChangePasswordRequest
	if err := h.ParseRequest([]byte(data), &req); err != nil {
		return h.createErrorMessage(req.Header.RequestID, "INVALID_REQUEST", err.Error(), "")
	}

	requestID := h.ExtractRequestID(&req)
	userID := h.ExtractUserID(&req)
	h.LogRequest("change_password", requestID, userID)

	// Call auth service
	err := h.authService.ChangePassword(ctx, req.UserID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		h.LogResponse("change_password", requestID, false, err)
		return h.createErrorMessage(requestID, "PASSWORD_CHANGE_FAILED", err.Error(), "")
	}

	// Create response
	response := ChangePasswordResponse{
		Header: ResponseHeader{
			RequestID: requestID,
			Success:   true,
			Timestamp: time.Now().Unix(),
		},
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		h.LogResponse("change_password", requestID, false, err)
		return h.createErrorMessage(requestID, "RESPONSE_ERROR", err.Error(), "")
	}

	h.LogResponse("change_password", requestID, true, nil)
	return []string{string(responseBytes)}, nil
}

// createErrorMessage creates an error response message.
func (h *AuthHandler) createErrorMessage(requestID, code, message, detail string) ([]string, error) {
	if requestID == "" {
		requestID = unknownOperation
	}

	responseBytes, err := h.CreateErrorResponse(requestID, code, message, detail)
	if err != nil {
		return nil, fmt.Errorf("failed to create error response: %w", err)
	}

	return []string{string(responseBytes)}, nil
}
