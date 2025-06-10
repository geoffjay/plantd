package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/geoffjay/plantd/identity/internal/auth"
	"github.com/geoffjay/plantd/identity/internal/services"
	"github.com/sirupsen/logrus"
)

// HandlerRegistry manages and routes MDP messages to appropriate handlers.
type HandlerRegistry struct {
	handlers map[string]Handler
	logger   *logrus.Logger
}

// NewHandlerRegistry creates a new handler registry.
func NewHandlerRegistry(
	userService services.UserService,
	orgService services.OrganizationService,
	roleService services.RoleService,
	authService *auth.AuthService,
	logger *logrus.Logger,
) *HandlerRegistry {
	registry := &HandlerRegistry{
		handlers: make(map[string]Handler),
		logger:   logger,
	}

	// Register handlers
	registry.RegisterHandler("identity.auth", NewAuthHandler(authService, logger))
	registry.RegisterHandler("identity.user", NewUserHandler(userService, logger))
	registry.RegisterHandler("identity.organization", NewOrganizationHandler(orgService, logger))
	registry.RegisterHandler("identity.role", NewRoleHandler(roleService, logger))
	registry.RegisterHandler("identity.health", NewHealthHandler(logger))

	return registry
}

// RegisterHandler registers a handler for a specific service name.
func (r *HandlerRegistry) RegisterHandler(serviceName string, handler Handler) {
	r.handlers[serviceName] = handler
	r.logger.WithFields(logrus.Fields{
		"service": serviceName,
		"handler": fmt.Sprintf("%T", handler),
	}).Info("Registered MDP handler")
}

// HandleMessage routes an incoming MDP message to the appropriate handler.
func (r *HandlerRegistry) HandleMessage(ctx context.Context, serviceName string, message []string) ([]string, error) {
	r.logger.WithFields(logrus.Fields{
		"service":     serviceName,
		"message_len": len(message),
	}).Debug("Routing MDP message")

	// Find handler for service
	handler, exists := r.handlers[serviceName]
	if !exists {
		return r.createServiceNotFoundError(serviceName)
	}

	// Route to handler
	return handler.HandleMessage(ctx, message)
}

// GetRegisteredServices returns a list of all registered service names.
func (r *HandlerRegistry) GetRegisteredServices() []string {
	services := make([]string, 0, len(r.handlers))
	for serviceName := range r.handlers {
		services = append(services, serviceName)
	}
	return services
}

// GetHandlerForService returns the handler for a specific service.
func (r *HandlerRegistry) GetHandlerForService(serviceName string) (Handler, bool) {
	handler, exists := r.handlers[serviceName]
	return handler, exists
}

// createServiceNotFoundError creates an error response for unknown services.
func (r *HandlerRegistry) createServiceNotFoundError(serviceName string) ([]string, error) {
	errorMsg := fmt.Sprintf("Service not found: %s", serviceName)
	availableServices := strings.Join(r.GetRegisteredServices(), ", ")
	detail := fmt.Sprintf("Available services: %s", availableServices)

	// Create basic error response
	response := fmt.Sprintf(`{
		"header": {
			"request_id": "unknown",
			"success": false,
			"error": "%s",
			"timestamp": %d
		},
		"code": "SERVICE_NOT_FOUND",
		"detail": "%s"
	}`, errorMsg, time.Now().Unix(), detail)

	return []string{response}, nil
}

// HealthHandler handles health check requests.
type HealthHandler struct {
	*BaseHandler
}

// NewHealthHandler creates a new health check handler.
func NewHealthHandler(logger *logrus.Logger) *HealthHandler {
	return &HealthHandler{
		BaseHandler: NewBaseHandler("identity.health", logger),
	}
}

// HandleMessage handles health check MDP messages.
func (h *HealthHandler) HandleMessage(ctx context.Context, message []string) ([]string, error) {
	if len(message) < 1 {
		return h.createErrorMessage("", "INVALID_MESSAGE", "Message must contain operation", "")
	}

	operation := message[0]
	data := ""
	if len(message) > 1 {
		data = message[1]
	}

	switch operation {
	case "check":
		return h.handleHealthCheck(ctx, data)
	default:
		return h.createErrorMessage("", "UNKNOWN_OPERATION", fmt.Sprintf("Unknown operation: %s", operation), "")
	}
}

// handleHealthCheck processes health check requests.
func (h *HealthHandler) handleHealthCheck(_ context.Context, data string) ([]string, error) {
	var req HealthCheckRequest
	requestID := "health-" + time.Now().Format("20060102150405")

	if data != "" {
		if err := h.ParseRequest([]byte(data), &req); err != nil {
			return h.createErrorMessage(requestID, "INVALID_REQUEST", err.Error(), "")
		}
		requestID = h.ExtractRequestID(&req)
	}

	userID := h.ExtractUserID(&req)
	h.LogRequest("health_check", requestID, userID)

	// Create health check response
	response := HealthCheckResponse{
		Header: ResponseHeader{
			RequestID: requestID,
			Success:   true,
			Timestamp: time.Now().Unix(),
		},
		Status:   "healthy",
		Version:  "1.0.0",
		Uptime:   time.Since(time.Now().Add(-time.Hour)), // Placeholder
		DBStatus: "connected",
		Services: []string{"auth", "user", "organization", "role"},
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		h.LogResponse("health_check", requestID, false, err)
		return h.createErrorMessage(requestID, "RESPONSE_ERROR", err.Error(), "")
	}

	h.LogResponse("health_check", requestID, true, nil)
	return []string{string(responseBytes)}, nil
}

// createErrorMessage creates an error response message.
func (h *HealthHandler) createErrorMessage(requestID, code, message, detail string) ([]string, error) {
	if requestID == "" {
		requestID = "unknown"
	}

	responseBytes, err := h.CreateErrorResponse(requestID, code, message, detail)
	if err != nil {
		return nil, fmt.Errorf("failed to create error response: %w", err)
	}

	return []string{string(responseBytes)}, nil
}
