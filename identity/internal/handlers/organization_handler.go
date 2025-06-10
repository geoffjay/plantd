package handlers

import (
	"context"
	"fmt"

	"github.com/geoffjay/plantd/identity/internal/services"
	"github.com/sirupsen/logrus"
)

// OrganizationHandler handles organization management MDP messages.
type OrganizationHandler struct {
	*BaseHandler
	orgService services.OrganizationService
}

// NewOrganizationHandler creates a new organization management handler.
func NewOrganizationHandler(orgService services.OrganizationService, logger *logrus.Logger) *OrganizationHandler {
	return &OrganizationHandler{
		BaseHandler: NewBaseHandler("identity.organization", logger),
		orgService:  orgService,
	}
}

// HandleMessage handles incoming MDP messages for organization operations.
func (h *OrganizationHandler) HandleMessage(_ context.Context, message []string) ([]string, error) {
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
		return h.createErrorMessage("", "NOT_IMPLEMENTED", "Organization creation not yet implemented", "")
	case "get":
		return h.createErrorMessage("", "NOT_IMPLEMENTED", "Organization retrieval not yet implemented", "")
	case "update":
		return h.createErrorMessage("", "NOT_IMPLEMENTED", "Organization update not yet implemented", "")
	case "delete":
		return h.createErrorMessage("", "NOT_IMPLEMENTED", "Organization deletion not yet implemented", "")
	case "list":
		return h.createErrorMessage("", "NOT_IMPLEMENTED", "Organization listing not yet implemented", "")
	default:
		return h.createErrorMessage("", "UNKNOWN_OPERATION", fmt.Sprintf("Unknown operation: %s", operation), "")
	}
}

// createErrorMessage creates an error response message.
func (h *OrganizationHandler) createErrorMessage(requestID, code, message, detail string) ([]string, error) {
	if requestID == "" {
		requestID = unknownOperation
	}

	responseBytes, err := h.CreateErrorResponse(requestID, code, message, detail)
	if err != nil {
		return nil, fmt.Errorf("failed to create error response: %w", err)
	}

	return []string{string(responseBytes)}, nil
}
