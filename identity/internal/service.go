// Package internal provides the main service implementation for the identity service.
package internal

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/geoffjay/plantd/core/mdp"
	"github.com/geoffjay/plantd/identity/internal/auth"
	"github.com/geoffjay/plantd/identity/internal/config"
	"github.com/geoffjay/plantd/identity/internal/handlers"
	"github.com/geoffjay/plantd/identity/internal/repositories"
	"github.com/geoffjay/plantd/identity/internal/services"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Service represents the main identity service.
type Service struct {
	config          *config.Config
	db              *gorm.DB
	logger          *logrus.Logger
	worker          *mdp.Worker
	handlerRegistry *handlers.HandlerRegistry

	// Services
	userService services.UserService
	orgService  services.OrganizationService
	roleService services.RoleService
	authService *auth.AuthService

	// Repositories
	userRepo repositories.UserRepository
	orgRepo  repositories.OrganizationRepository
	roleRepo repositories.RoleRepository

	// Service state
	startTime time.Time
	shutdown  bool
}

// NewService creates a new identity service instance.
func NewService(cfg *config.Config, db *gorm.DB) (*Service, error) {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Initialize repositories
	repoContainer := repositories.NewContainer(db)

	// Initialize services
	serviceFactory := services.NewServiceFactory(repoContainer)
	userService := serviceFactory.CreateUserService()
	orgService := serviceFactory.CreateOrganizationService()
	roleService := serviceFactory.CreateRoleService()

	// Initialize auth service
	authConfig := auth.DefaultAuthConfig()
	authService := auth.NewAuthService(authConfig, repoContainer.User, userService, logger)

	// Initialize handler registry
	handlerRegistry := handlers.NewHandlerRegistry(
		userService,
		orgService,
		roleService,
		authService,
		logger,
	)

	service := &Service{
		config:          cfg,
		db:              db,
		logger:          logger,
		handlerRegistry: handlerRegistry,
		userService:     userService,
		orgService:      orgService,
		roleService:     roleService,
		authService:     authService,
		userRepo:        repoContainer.User,
		orgRepo:         repoContainer.Organization,
		roleRepo:        repoContainer.Role,
		startTime:       time.Now(),
		shutdown:        false,
	}

	return service, nil
}

// Run starts the identity service.
func (s *Service) Run(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	s.logger.WithFields(logrus.Fields{
		"service": "identity",
		"context": "service.run",
	}).Info("Starting identity service")

	// Initialize MDP worker
	if err := s.initWorker(); err != nil {
		return fmt.Errorf("failed to initialize MDP worker: %w", err)
	}
	defer s.worker.Close()

	// Start message processing loop
	wg.Add(1)
	go s.runMessageLoop(ctx, wg)

	// Wait for shutdown signal
	<-ctx.Done()

	s.logger.WithFields(logrus.Fields{
		"service": "identity",
		"context": "service.run",
	}).Info("Identity service shutting down")

	s.shutdown = true
	return nil
}

// initWorker initializes the MDP worker.
func (s *Service) initWorker() error {
	brokerEndpoint := "tcp://127.0.0.1:9797" // Default broker endpoint
	serviceName := "org.plantd.Identity"

	var err error
	s.worker, err = mdp.NewWorker(brokerEndpoint, serviceName)
	if err != nil {
		return fmt.Errorf("failed to create MDP worker: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"broker_endpoint": brokerEndpoint,
		"service_name":    serviceName,
	}).Info("MDP worker initialized")

	return nil
}

// runMessageLoop processes incoming MDP messages.
func (s *Service) runMessageLoop(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	s.logger.WithFields(logrus.Fields{
		"service": "identity",
		"context": "message_loop",
	}).Info("Starting MDP message processing loop")

	var reply []string
	for !s.shutdown && !s.worker.Terminated() {
		// Receive message from broker and send reply from previous iteration
		message, err := s.worker.Recv(reply)
		if err != nil {
			if !s.shutdown {
				s.logger.WithError(err).Error("Failed to receive MDP message")
			}
			continue
		}

		if len(message) == 0 {
			continue
		}

		// Process message and prepare reply for next iteration
		reply = s.processMessage(ctx, message)
	}

	s.logger.WithFields(logrus.Fields{
		"service": "identity",
		"context": "message_loop",
	}).Info("MDP message processing loop stopped")
}

// processMessage processes a single MDP message.
func (s *Service) processMessage(ctx context.Context, message []string) []string {
	// Add detailed debug logging
	s.logger.WithFields(logrus.Fields{
		"message_length": len(message),
		"raw_message":    message,
	}).Debug("Identity service received MDP message")

	if len(message) < 1 {
		s.logger.Warn("Received empty MDP message")
		return []string{}
	}

	// Extract service name from message
	// MDP message format: [service_name, operation, data...]
	serviceName := "identity.health" // Default to health check
	messageData := message

	// If message has multiple parts, first part might be service name
	if len(message) > 1 {
		// Check if first part looks like a service name
		if len(message[0]) > 0 && (message[0] == "auth" || message[0] == "user" ||
			message[0] == "organization" || message[0] == "role" || message[0] == "health") {
			serviceName = "identity." + message[0]
			messageData = message[1:]
		}
	}

	s.logger.WithFields(logrus.Fields{
		"extracted_service": serviceName,
		"message_data":      messageData,
		"message_len":       len(messageData),
	}).Debug("Processing MDP message")

	// Route to appropriate handler
	response, err := s.handlerRegistry.HandleMessage(ctx, serviceName, messageData)
	if err != nil {
		s.logger.WithError(err).Error("Failed to handle MDP message")
		return []string{fmt.Sprintf(`{"error": "Internal server error: %s"}`, err.Error())}
	}

	s.logger.WithFields(logrus.Fields{
		"service":       serviceName,
		"response_len":  len(response),
		"response_type": fmt.Sprintf("%T", response),
	}).Debug("Identity service returning response")

	return response
}

// GetUptime returns the service uptime.
func (s *Service) GetUptime() time.Duration {
	return time.Since(s.startTime)
}

// GetStatus returns the service status.
func (s *Service) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"status":    "running",
		"uptime":    s.GetUptime().String(),
		"services":  s.handlerRegistry.GetRegisteredServices(),
		"db_status": "connected",
		"version":   "1.0.0",
	}
}

// Shutdown gracefully shuts down the service.
func (s *Service) Shutdown() {
	s.shutdown = true
	if s.worker != nil {
		s.worker.Shutdown()
	}
	if s.authService != nil {
		s.authService.Stop()
	}
}
