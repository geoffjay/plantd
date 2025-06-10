package handlers

import (
	"context"
	"fmt"

	"github.com/geoffjay/plantd/identity/internal/services"
	"github.com/sirupsen/logrus"
)

// RoleHandler handles role management MDP messages.
type RoleHandler struct {
	*BaseHandler
	roleService services.RoleService
}

// NewRoleHandler creates a new role management handler.
func NewRoleHandler(roleService services.RoleService, logger *logrus.Logger) *RoleHandler {
	return &RoleHandler{
		BaseHandler: NewBaseHandler("identity.role", logger),
		roleService: roleService,
	}
}

// HandleMessage handles incoming MDP messages for role operations.
func (h *RoleHandler) HandleMessage(_ context.Context, message []string) ([]string, error) {
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
	// data := message[1]

	switch operation {
	case "create":
		return h.createErrorMessage("", "NOT_IMPLEMENTED", "Role creation not yet implemented", "")
	case "get":
		return h.createErrorMessage("", "NOT_IMPLEMENTED", "Role retrieval not yet implemented", "")
	case "update":
		return h.createErrorMessage("", "NOT_IMPLEMENTED", "Role update not yet implemented", "")
	case "delete":
		return h.createErrorMessage("", "NOT_IMPLEMENTED", "Role deletion not yet implemented", "")
	case "list":
		return h.createErrorMessage("", "NOT_IMPLEMENTED", "Role listing not yet implemented", "")
	case "check_permission":
		return h.createErrorMessage("", "NOT_IMPLEMENTED", "Permission checking not yet implemented", "")
	default:
		return h.createErrorMessage("", "UNKNOWN_OPERATION", fmt.Sprintf("Unknown operation: %s", operation), "")
	}
}

// createErrorMessage creates an error response message.
func (h *RoleHandler) createErrorMessage(requestID, code, message, detail string) ([]string, error) {
	if requestID == "" {
		requestID = "unknown"
	}

	responseBytes, err := h.CreateErrorResponse(requestID, code, message, detail)
	if err != nil {
		return nil, fmt.Errorf("failed to create error response: %w", err)
	}

	return []string{string(responseBytes)}, nil
}
