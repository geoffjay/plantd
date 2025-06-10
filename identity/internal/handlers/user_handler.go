package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/geoffjay/plantd/identity/internal/models"
	"github.com/geoffjay/plantd/identity/internal/services"
	"github.com/sirupsen/logrus"
)

const (
	createUserOperation           = "create"
	getUserOperation              = "get"
	updateUserOperation           = "update"
	deleteUserOperation           = "delete"
	listUsersOperation            = "list"
	activateUserOperation         = "activate"
	deactivateUserOperation       = "deactivate"
	verifyEmailOperation          = "verify_email"
	assignRoleOperation           = "assign_role"
	unassignRoleOperation         = "unassign_role"
	assignOrganizationOperation   = "assign_organization"
	unassignOrganizationOperation = "unassign_organization"
	unknownOperation              = "unknown"
)

// UserHandler handles user management MDP messages.
type UserHandler struct {
	*BaseHandler
	userService services.UserService
}

// NewUserHandler creates a new user management handler.
func NewUserHandler(userService services.UserService, logger *logrus.Logger) *UserHandler {
	return &UserHandler{
		BaseHandler: NewBaseHandler("identity.user", logger),
		userService: userService,
	}
}

// HandleMessage handles incoming MDP messages for user operations.
func (h *UserHandler) HandleMessage(ctx context.Context, message []string) ([]string, error) {
	defer func() {
		if responseBytes, err := h.HandlePanic(unknownOperation); responseBytes != nil { //nolint:revive
			// Return the panic response
		} else if err != nil {
			h.logger.WithError(err).Error("Error handling panic")
		}
	}()

	if len(message) < 2 {
		return h.createErrorMessage("", "INVALID_MESSAGE", "Message must contain operation and data", "")
	}

	operation := message[0]
	data := message[1]

	switch operation {
	case createUserOperation:
		return h.handleCreateUser(ctx, data)
	case getUserOperation:
		return h.handleGetUser(ctx, data)
	case updateUserOperation:
		return h.handleUpdateUser(ctx, data)
	case deleteUserOperation:
		return h.handleDeleteUser(ctx, data)
	case listUsersOperation:
		return h.handleListUsers(ctx, data)
	case activateUserOperation:
		return h.handleActivateUser(ctx, data)
	case deactivateUserOperation:
		return h.handleDeactivateUser(ctx, data)
	case verifyEmailOperation:
		return h.handleVerifyEmail(ctx, data)
	case assignRoleOperation:
		return h.handleAssignRole(ctx, data)
	case unassignRoleOperation:
		return h.handleUnassignRole(ctx, data)
	case assignOrganizationOperation:
		return h.handleAssignOrganization(ctx, data)
	case unassignOrganizationOperation:
		return h.handleUnassignOrganization(ctx, data)
	default:
		return h.createErrorMessage("", "UNKNOWN_OPERATION", fmt.Sprintf("Unknown operation: %s", operation), "")
	}
}

// handleCreateUser processes user creation requests.
func (h *UserHandler) handleCreateUser(ctx context.Context, data string) ([]string, error) {
	var req CreateUserRequest
	if err := h.ParseRequest([]byte(data), &req); err != nil {
		return h.createErrorMessage(req.Header.RequestID, "INVALID_REQUEST", err.Error(), "")
	}

	requestID := h.ExtractRequestID(&req)
	userID := h.ExtractUserID(&req)
	h.LogRequest("create_user", requestID, userID)

	// Convert to service request
	serviceReq := &services.CreateUserRequest{
		Email:            req.Email,
		Username:         req.Username,
		Password:         req.Password,
		FirstName:        req.FirstName,
		LastName:         req.LastName,
		SendWelcomeEmail: req.SendWelcomeEmail,
	}

	// Call user service
	user, err := h.userService.CreateUser(ctx, serviceReq)
	if err != nil {
		h.LogResponse("create_user", requestID, false, err)
		return h.createErrorMessage(requestID, "CREATE_USER_FAILED", err.Error(), "")
	}

	// Create response
	response := CreateUserResponse{
		Header: ResponseHeader{
			RequestID: requestID,
			Success:   true,
			Timestamp: time.Now().Unix(),
		},
		User: user,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		h.LogResponse("create_user", requestID, false, err)
		return h.createErrorMessage(requestID, "RESPONSE_ERROR", err.Error(), "")
	}

	h.LogResponse("create_user", requestID, true, nil)
	return []string{string(responseBytes)}, nil
}

// handleGetUser processes user retrieval requests.
func (h *UserHandler) handleGetUser(ctx context.Context, data string) ([]string, error) {
	var req GetUserRequest
	if err := h.ParseRequest([]byte(data), &req); err != nil {
		return h.createErrorMessage(req.Header.RequestID, "INVALID_REQUEST", err.Error(), "")
	}

	requestID := h.ExtractRequestID(&req)
	userID := h.ExtractUserID(&req)
	h.LogRequest("get_user", requestID, userID)

	var user *models.User
	var err error

	// Determine which lookup method to use
	if req.UserID != nil {
		user, err = h.userService.GetUserByID(ctx, *req.UserID)
	} else if req.Email != "" {
		user, err = h.userService.GetUserByEmail(ctx, req.Email)
	} else if req.Username != "" {
		user, err = h.userService.GetUserByUsername(ctx, req.Username)
	} else {
		h.LogResponse("get_user", requestID, false, fmt.Errorf("no identifier provided"))
		return h.createErrorMessage(requestID, "INVALID_REQUEST", "Must provide user_id, email, or username", "")
	}

	if err != nil {
		h.LogResponse("get_user", requestID, false, err)
		return h.createErrorMessage(requestID, "GET_USER_FAILED", err.Error(), "")
	}

	// Create response
	response := GetUserResponse{
		Header: ResponseHeader{
			RequestID: requestID,
			Success:   true,
			Timestamp: time.Now().Unix(),
		},
		User: user,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		h.LogResponse("get_user", requestID, false, err)
		return h.createErrorMessage(requestID, "RESPONSE_ERROR", err.Error(), "")
	}

	h.LogResponse("get_user", requestID, true, nil)
	return []string{string(responseBytes)}, nil
}

// handleUpdateUser processes user update requests.
func (h *UserHandler) handleUpdateUser(ctx context.Context, data string) ([]string, error) {
	var req UpdateUserRequest
	if err := h.ParseRequest([]byte(data), &req); err != nil {
		return h.createErrorMessage(req.Header.RequestID, "INVALID_REQUEST", err.Error(), "")
	}

	requestID := h.ExtractRequestID(&req)
	userID := h.ExtractUserID(&req)
	h.LogRequest("update_user", requestID, userID)

	// Convert to service request
	serviceReq := &services.UpdateUserRequest{
		Email:     req.Email,
		Username:  req.Username,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		IsActive:  req.IsActive,
	}

	// Call user service
	user, err := h.userService.UpdateUser(ctx, req.UserID, serviceReq)
	if err != nil {
		h.LogResponse("update_user", requestID, false, err)
		return h.createErrorMessage(requestID, "UPDATE_USER_FAILED", err.Error(), "")
	}

	// Create response
	response := UpdateUserResponse{
		Header: ResponseHeader{
			RequestID: requestID,
			Success:   true,
			Timestamp: time.Now().Unix(),
		},
		User: user,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		h.LogResponse("update_user", requestID, false, err)
		return h.createErrorMessage(requestID, "RESPONSE_ERROR", err.Error(), "")
	}

	h.LogResponse("update_user", requestID, true, nil)
	return []string{string(responseBytes)}, nil
}

// handleDeleteUser processes user deletion requests.
func (h *UserHandler) handleDeleteUser(ctx context.Context, data string) ([]string, error) {
	var req DeleteUserRequest
	if err := h.ParseRequest([]byte(data), &req); err != nil {
		return h.createErrorMessage(req.Header.RequestID, "INVALID_REQUEST", err.Error(), "")
	}

	requestID := h.ExtractRequestID(&req)
	userID := h.ExtractUserID(&req)
	h.LogRequest("delete_user", requestID, userID)

	// Call user service
	err := h.userService.DeleteUser(ctx, req.UserID)
	if err != nil {
		h.LogResponse("delete_user", requestID, false, err)
		return h.createErrorMessage(requestID, "DELETE_USER_FAILED", err.Error(), "")
	}

	// Create response
	response := DeleteUserResponse{
		Header: ResponseHeader{
			RequestID: requestID,
			Success:   true,
			Timestamp: time.Now().Unix(),
		},
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		h.LogResponse("delete_user", requestID, false, err)
		return h.createErrorMessage(requestID, "RESPONSE_ERROR", err.Error(), "")
	}

	h.LogResponse("delete_user", requestID, true, nil)
	return []string{string(responseBytes)}, nil
}

// handleListUsers processes user listing requests.
func (h *UserHandler) handleListUsers(ctx context.Context, data string) ([]string, error) {
	var req ListUsersRequest
	if err := h.ParseRequest([]byte(data), &req); err != nil {
		return h.createErrorMessage(req.Header.RequestID, "INVALID_REQUEST", err.Error(), "")
	}

	requestID := h.ExtractRequestID(&req)
	userID := h.ExtractUserID(&req)
	h.LogRequest("list_users", requestID, userID)

	// Convert to service request
	serviceReq := &services.ListUsersRequest{
		Offset:            req.Offset,
		Limit:             req.Limit,
		IncludeInactive:   req.IncludeInactive,
		IncludeUnverified: req.IncludeUnverified,
		SortBy:            req.SortBy,
		SortOrder:         req.SortOrder,
	}

	// Call user service
	users, err := h.userService.ListUsers(ctx, serviceReq)
	if err != nil {
		h.LogResponse("list_users", requestID, false, err)
		return h.createErrorMessage(requestID, "LIST_USERS_FAILED", err.Error(), "")
	}

	// Get total count
	total, err := h.userService.CountUsers(ctx)
	if err != nil {
		h.LogResponse("list_users", requestID, false, err)
		return h.createErrorMessage(requestID, "COUNT_USERS_FAILED", err.Error(), "")
	}

	// Create response
	response := ListUsersResponse{
		Header: ResponseHeader{
			RequestID: requestID,
			Success:   true,
			Timestamp: time.Now().Unix(),
		},
		Users:  users,
		Total:  total,
		Offset: req.Offset,
		Limit:  req.Limit,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		h.LogResponse("list_users", requestID, false, err)
		return h.createErrorMessage(requestID, "RESPONSE_ERROR", err.Error(), "")
	}

	h.LogResponse("list_users", requestID, true, nil)
	return []string{string(responseBytes)}, nil
}

// handleActivateUser processes user activation requests.
func (h *UserHandler) handleActivateUser(ctx context.Context, data string) ([]string, error) {
	var req struct {
		Header RequestHeader `json:"header"`
		UserID uint          `json:"user_id" validate:"required"`
	}
	if err := h.ParseRequest([]byte(data), &req); err != nil {
		return h.createErrorMessage(req.Header.RequestID, "INVALID_REQUEST", err.Error(), "")
	}

	requestID := h.ExtractRequestID(&req)
	userID := h.ExtractUserID(&req)
	h.LogRequest("activate_user", requestID, userID)

	// Call user service
	err := h.userService.ActivateUser(ctx, req.UserID)
	if err != nil {
		h.LogResponse("activate_user", requestID, false, err)
		return h.createErrorMessage(requestID, "ACTIVATE_USER_FAILED", err.Error(), "")
	}

	// Create response
	response := struct {
		Header ResponseHeader `json:"header"`
	}{
		Header: ResponseHeader{
			RequestID: requestID,
			Success:   true,
			Timestamp: time.Now().Unix(),
		},
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		h.LogResponse("activate_user", requestID, false, err)
		return h.createErrorMessage(requestID, "RESPONSE_ERROR", err.Error(), "")
	}

	h.LogResponse("activate_user", requestID, true, nil)
	return []string{string(responseBytes)}, nil
}

// handleDeactivateUser processes user deactivation requests.
func (h *UserHandler) handleDeactivateUser(ctx context.Context, data string) ([]string, error) {
	var req struct {
		Header RequestHeader `json:"header"`
		UserID uint          `json:"user_id" validate:"required"`
	}
	if err := h.ParseRequest([]byte(data), &req); err != nil {
		return h.createErrorMessage(req.Header.RequestID, "INVALID_REQUEST", err.Error(), "")
	}

	requestID := h.ExtractRequestID(&req)
	userID := h.ExtractUserID(&req)
	h.LogRequest("deactivate_user", requestID, userID)

	// Call user service
	err := h.userService.DeactivateUser(ctx, req.UserID)
	if err != nil {
		h.LogResponse("deactivate_user", requestID, false, err)
		return h.createErrorMessage(requestID, "DEACTIVATE_USER_FAILED", err.Error(), "")
	}

	// Create response
	response := struct {
		Header ResponseHeader `json:"header"`
	}{
		Header: ResponseHeader{
			RequestID: requestID,
			Success:   true,
			Timestamp: time.Now().Unix(),
		},
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		h.LogResponse("deactivate_user", requestID, false, err)
		return h.createErrorMessage(requestID, "RESPONSE_ERROR", err.Error(), "")
	}

	h.LogResponse("deactivate_user", requestID, true, nil)
	return []string{string(responseBytes)}, nil
}

// handleVerifyEmail processes email verification requests.
func (h *UserHandler) handleVerifyEmail(ctx context.Context, data string) ([]string, error) {
	var req struct {
		Header RequestHeader `json:"header"`
		UserID uint          `json:"user_id" validate:"required"`
	}
	if err := h.ParseRequest([]byte(data), &req); err != nil {
		return h.createErrorMessage(req.Header.RequestID, "INVALID_REQUEST", err.Error(), "")
	}

	requestID := h.ExtractRequestID(&req)
	userID := h.ExtractUserID(&req)
	h.LogRequest("verify_email", requestID, userID)

	// Call user service
	err := h.userService.VerifyUserEmail(ctx, req.UserID)
	if err != nil {
		h.LogResponse("verify_email", requestID, false, err)
		return h.createErrorMessage(requestID, "VERIFY_EMAIL_FAILED", err.Error(), "")
	}

	// Create response
	response := struct {
		Header ResponseHeader `json:"header"`
	}{
		Header: ResponseHeader{
			RequestID: requestID,
			Success:   true,
			Timestamp: time.Now().Unix(),
		},
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		h.LogResponse("verify_email", requestID, false, err)
		return h.createErrorMessage(requestID, "RESPONSE_ERROR", err.Error(), "")
	}

	h.LogResponse("verify_email", requestID, true, nil)
	return []string{string(responseBytes)}, nil
}

// handleAssignRole processes role assignment requests.
func (h *UserHandler) handleAssignRole(ctx context.Context, data string) ([]string, error) {
	var req AssignRoleRequest
	if err := h.ParseRequest([]byte(data), &req); err != nil {
		return h.createErrorMessage(req.Header.RequestID, "INVALID_REQUEST", err.Error(), "")
	}

	requestID := h.ExtractRequestID(&req)
	userID := h.ExtractUserID(&req)
	h.LogRequest("assign_role", requestID, userID)

	// Call user service
	err := h.userService.AssignUserToRole(ctx, req.UserID, req.RoleID)
	if err != nil {
		h.LogResponse("assign_role", requestID, false, err)
		return h.createErrorMessage(requestID, "ASSIGN_ROLE_FAILED", err.Error(), "")
	}

	// Create response
	response := AssignRoleResponse{
		Header: ResponseHeader{
			RequestID: requestID,
			Success:   true,
			Timestamp: time.Now().Unix(),
		},
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		h.LogResponse("assign_role", requestID, false, err)
		return h.createErrorMessage(requestID, "RESPONSE_ERROR", err.Error(), "")
	}

	h.LogResponse("assign_role", requestID, true, nil)
	return []string{string(responseBytes)}, nil
}

// handleUnassignRole processes role unassignment requests.
func (h *UserHandler) handleUnassignRole(ctx context.Context, data string) ([]string, error) {
	var req UnassignRoleRequest
	if err := h.ParseRequest([]byte(data), &req); err != nil {
		return h.createErrorMessage(req.Header.RequestID, "INVALID_REQUEST", err.Error(), "")
	}

	requestID := h.ExtractRequestID(&req)
	userID := h.ExtractUserID(&req)
	h.LogRequest("unassign_role", requestID, userID)

	// Call user service
	err := h.userService.RemoveUserFromRole(ctx, req.UserID, req.RoleID)
	if err != nil {
		h.LogResponse("unassign_role", requestID, false, err)
		return h.createErrorMessage(requestID, "UNASSIGN_ROLE_FAILED", err.Error(), "")
	}

	// Create response
	response := UnassignRoleResponse{
		Header: ResponseHeader{
			RequestID: requestID,
			Success:   true,
			Timestamp: time.Now().Unix(),
		},
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		h.LogResponse("unassign_role", requestID, false, err)
		return h.createErrorMessage(requestID, "RESPONSE_ERROR", err.Error(), "")
	}

	h.LogResponse("unassign_role", requestID, true, nil)
	return []string{string(responseBytes)}, nil
}

// handleAssignOrganization processes organization assignment requests.
func (h *UserHandler) handleAssignOrganization(ctx context.Context, data string) ([]string, error) {
	var req struct {
		Header RequestHeader `json:"header"`
		UserID uint          `json:"user_id" validate:"required"`
		OrgID  uint          `json:"org_id" validate:"required"`
	}
	if err := h.ParseRequest([]byte(data), &req); err != nil {
		return h.createErrorMessage(req.Header.RequestID, "INVALID_REQUEST", err.Error(), "")
	}

	requestID := h.ExtractRequestID(&req)
	userID := h.ExtractUserID(&req)
	h.LogRequest("assign_organization", requestID, userID)

	// Call user service
	err := h.userService.AssignUserToOrganization(ctx, req.UserID, req.OrgID)
	if err != nil {
		h.LogResponse("assign_organization", requestID, false, err)
		return h.createErrorMessage(requestID, "ASSIGN_ORGANIZATION_FAILED", err.Error(), "")
	}

	// Create response
	response := struct {
		Header ResponseHeader `json:"header"`
	}{
		Header: ResponseHeader{
			RequestID: requestID,
			Success:   true,
			Timestamp: time.Now().Unix(),
		},
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		h.LogResponse("assign_organization", requestID, false, err)
		return h.createErrorMessage(requestID, "RESPONSE_ERROR", err.Error(), "")
	}

	h.LogResponse("assign_organization", requestID, true, nil)
	return []string{string(responseBytes)}, nil
}

// handleUnassignOrganization processes organization unassignment requests.
func (h *UserHandler) handleUnassignOrganization(ctx context.Context, data string) ([]string, error) {
	var req struct {
		Header RequestHeader `json:"header"`
		UserID uint          `json:"user_id" validate:"required"`
		OrgID  uint          `json:"org_id" validate:"required"`
	}
	if err := h.ParseRequest([]byte(data), &req); err != nil {
		return h.createErrorMessage(req.Header.RequestID, "INVALID_REQUEST", err.Error(), "")
	}

	requestID := h.ExtractRequestID(&req)
	userID := h.ExtractUserID(&req)
	h.LogRequest("unassign_organization", requestID, userID)

	// Call user service
	err := h.userService.RemoveUserFromOrganization(ctx, req.UserID, req.OrgID)
	if err != nil {
		h.LogResponse("unassign_organization", requestID, false, err)
		return h.createErrorMessage(requestID, "UNASSIGN_ORGANIZATION_FAILED", err.Error(), "")
	}

	// Create response
	response := struct {
		Header ResponseHeader `json:"header"`
	}{
		Header: ResponseHeader{
			RequestID: requestID,
			Success:   true,
			Timestamp: time.Now().Unix(),
		},
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		h.LogResponse("unassign_organization", requestID, false, err)
		return h.createErrorMessage(requestID, "RESPONSE_ERROR", err.Error(), "")
	}

	h.LogResponse("unassign_organization", requestID, true, nil)
	return []string{string(responseBytes)}, nil
}

// createErrorMessage creates an error response message.
func (h *UserHandler) createErrorMessage(requestID, code, message, detail string) ([]string, error) {
	if requestID == "" {
		requestID = unknownOperation
	}

	responseBytes, err := h.CreateErrorResponse(requestID, code, message, detail)
	if err != nil {
		return nil, fmt.Errorf("failed to create error response: %w", err)
	}

	return []string{string(responseBytes)}, nil
}
