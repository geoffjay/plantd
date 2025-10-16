package main

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/geoffjay/plantd/core/util"
	"github.com/geoffjay/plantd/identity/internal/auth"
	"github.com/geoffjay/plantd/identity/internal/config"
	"github.com/geoffjay/plantd/identity/internal/handlers"
	"github.com/geoffjay/plantd/identity/internal/repositories"
	"github.com/geoffjay/plantd/identity/internal/services"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Service defines the service type.
type Service struct {
	config             *config.Config
	db                 *gorm.DB
	logger             *logrus.Logger
	handlerRegistry    *handlers.HandlerRegistry
	transportManager   *TransportManager
	identityGRPCServer *IdentityGRPCServer

	// Services
	userService         services.UserService
	orgService          services.OrganizationService
	roleService         services.RoleService
	authService         *auth.AuthService
	rbacService         *auth.RBACService
	registrationService *auth.RegistrationService

	// Repositories
	userRepo repositories.UserRepository
	orgRepo  repositories.OrganizationRepository
	roleRepo repositories.RoleRepository

	// Service state
	startTime time.Time
}

// NewService creates an instance of the service.
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

	// Initialize auth services
	authService := auth.NewAuthService(cfg.ToAuthConfig(), repoContainer.User, userService, logger)

	// Initialize RBAC service
	slogLogger := slog.New(slog.NewTextHandler(logger.Writer(), &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	rbacService := auth.NewRBACService(repoContainer.User, repoContainer.Role, repoContainer.Organization, slogLogger)

	// Initialize registration service
	passwordValidator := auth.NewPasswordValidator(cfg.ToPasswordConfig())
	jwtManager := auth.NewJWTManager(cfg.ToJWTConfig(), auth.NewInMemoryBlacklist())
	rateLimiter := auth.NewRateLimiter(cfg.ToRateLimiterConfig())
	registrationService := auth.NewRegistrationService(
		cfg.ToRegistrationConfig(),
		repoContainer.User,
		userService,
		passwordValidator,
		jwtManager,
		rateLimiter,
		logger,
	)

	// Initialize handler registry for MDP
	handlerRegistry := handlers.NewHandlerRegistry(
		userService,
		orgService,
		roleService,
		authService,
		logger,
	)

	// Initialize gRPC server
	identityGRPCServer := NewIdentityGRPCServer(
		authService,
		userService,
		orgService,
		roleService,
		rbacService,
		registrationService,
	)

	service := &Service{
		config:              cfg,
		db:                  db,
		logger:              logger,
		handlerRegistry:     handlerRegistry,
		transportManager:    NewTransportManager(),
		identityGRPCServer:  identityGRPCServer,
		userService:         userService,
		orgService:          orgService,
		roleService:         roleService,
		authService:         authService,
		rbacService:         rbacService,
		registrationService: registrationService,
		userRepo:            repoContainer.User,
		orgRepo:             repoContainer.Organization,
		roleRepo:            repoContainer.Role,
		startTime:           time.Now(),
	}

	return service, nil
}

func (s *Service) setupTransports() {
	config := s.config

	// Allow environment variable to override config for gRPC port
	grpcPort := config.Transport.GRPCPort
	if envPort := util.Getenv("PLANTD_IDENTITY_GRPC_PORT", ""); envPort != "" {
		grpcPort = envPort
	}

	log.WithFields(log.Fields{
		"grpc_enabled": true,
		"grpc_port":    grpcPort,
	}).Info("Configuring gRPC-only transport")

	// Add gRPC transport
	grpcTransport := NewGRPCTransport(grpcPort, s.identityGRPCServer)
	s.transportManager.AddTransport(grpcTransport)
	log.WithField("port", grpcPort).Info("gRPC transport configured")
}

// Run handles the service execution.
func (s *Service) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer s.transportManager.StopAll()

	log.WithFields(log.Fields{
		"service": "identity",
		"context": "service.run",
	}).Info("Starting identity service")

	// Setup transports
	s.setupTransports()

	// Start transports
	if err := s.transportManager.StartAll(ctx, wg); err != nil {
		log.WithError(err).Fatal("Failed to start transports")
	}

	// Wait for shutdown signal
	<-ctx.Done()

	log.WithFields(log.Fields{
		"service": "identity",
		"context": "service.run",
	}).Info("Identity service shutting down")
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
	if s.transportManager != nil {
		s.transportManager.StopAll()
	}
}
