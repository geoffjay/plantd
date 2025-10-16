package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/geoffjay/plantd/core/util"
	"github.com/geoffjay/plantd/identity/pkg/client"
	"github.com/geoffjay/plantd/state/auth"

	"github.com/nelkinda/health-go"
	log "github.com/sirupsen/logrus"
)

// Service defines the service type.
type Service struct {
	handler          *Handler
	manager          *Manager
	store            *Store
	identityClient   *client.Client
	authMiddleware   *auth.AuthMiddleware
	transportManager *TransportManager
}

// NewService creates an instance of the service.
func NewService() *Service {
	return &Service{
		manager:          NewManager(">tcp://localhost:11001"),
		transportManager: NewTransportManager(),
	}
}

func (s *Service) setupStore() {
	s.store = NewStore()
	path := util.Getenv("PLANTD_STATE_DB", "plantd-state.db")
	if err := s.store.Load(path); err != nil {
		log.WithFields(log.Fields{"err": err}).Panic("failed to setup KV store")
	}
}

func (s *Service) setupIdentityClient() {
	config := GetConfig()

	// Create identity client configuration
	clientConfig := &client.Config{
		BrokerEndpoint: config.Identity.Endpoint,
		Timeout:        30 * time.Second,
		Logger:         log.StandardLogger(),
	}

	// Parse timeout if provided
	if config.Identity.Timeout != "" {
		if timeout, err := time.ParseDuration(config.Identity.Timeout); err == nil {
			clientConfig.Timeout = timeout
		}
	}

	// Create identity client with graceful degradation
	identityClient, err := client.NewClient(clientConfig)
	if err != nil {
		log.WithFields(log.Fields{
			"endpoint": config.Identity.Endpoint,
			"error":    err,
		}).Warn("Failed to setup identity client - authentication will be disabled")

		// Set auth middleware to nil to trigger unauthenticated mode
		s.identityClient = nil
		s.authMiddleware = nil
		return
	}

	s.identityClient = identityClient

	// Create authentication middleware
	authConfig := &auth.Config{
		IdentityClient: identityClient,
		CacheTTL:       5 * time.Minute,
		Logger:         log.StandardLogger(),
	}

	s.authMiddleware = auth.NewAuthMiddleware(authConfig)

	log.WithFields(log.Fields{
		"endpoint": config.Identity.Endpoint,
		"timeout":  clientConfig.Timeout,
	}).Info("Identity client initialized successfully - authentication enabled")
}

func (s *Service) setupHandler() {
	var err error
	s.handler = NewHandler()

	// Create original callbacks (without authentication)
	originalCallbacks := map[string]interface{ Execute(string) ([]byte, error) }{
		"create-scope": &createScopeCallback{
			name: "create-scope", store: s.store, manager: s.manager,
		},
		"delete-scope": &deleteScopeCallback{
			name: "delete-scope", store: s.store, manager: s.manager,
		},
		"delete": &deleteCallback{
			name: "delete", store: s.store,
		},
		"state-get": &getCallback{
			name: "state-get", store: s.store,
		},
		"state-set": &setCallback{
			name: "state-set", store: s.store,
		},
		"health": &healthCallback{
			name: "health", store: s.store,
		},
		"list-scopes": &listScopesCallback{
			name: "list-scopes", store: s.store,
		},
		"list-keys": &listKeysCallback{
			name: "list-keys", store: s.store,
		},
	}

	// Wrap callbacks with authentication if auth middleware is available
	if s.authMiddleware != nil {
		authenticatedCallbacks := auth.CreateAuthenticatedCallbacks(
			originalCallbacks,
			s.authMiddleware,
		)

		// Register authenticated callbacks
		for name, callback := range authenticatedCallbacks {
			// Convert back to HandlerCallback for registration
			handlerCallback := callback.(HandlerCallback)
			err = s.RegisterCallback(name, handlerCallback)
			if err != nil {
				log.WithFields(log.Fields{
					"callback": name,
					"error":    err,
				}).Fatal("Failed to register authenticated callback")
			}
		}

		log.Info("All callbacks registered with authentication middleware")
	} else {
		// Fallback: register original callbacks without authentication
		log.Warn("Auth middleware not available, registering callbacks without authentication")

		for name, callback := range originalCallbacks {
			// Convert back to HandlerCallback for registration
			handlerCallback := callback.(HandlerCallback)
			err = s.RegisterCallback(name, handlerCallback)
			if err != nil {
				log.WithFields(log.Fields{
					"callback": name,
					"error":    err,
				}).Fatal("Failed to register callback")
			}
		}
	}
}

func (s *Service) setupTransports() {
	config := GetConfig()

	// Allow environment variable to override config for gRPC port
	grpcPort := config.Transport.GRPCPort
	if envPort := util.Getenv("PLANTD_STATE_GRPC_PORT", ""); envPort != "" {
		grpcPort = envPort
	}

	log.WithFields(log.Fields{
		"grpc_enabled": true,
		"grpc_port":    grpcPort,
	}).Info("Configuring gRPC-only transport")

	// Add gRPC transport with authentication middleware
	grpcTransport := NewGRPCTransport(grpcPort, s.store, s.authMiddleware)
	s.transportManager.AddTransport(grpcTransport)
	log.WithField("port", grpcPort).Info("gRPC transport configured")
}

func (s *Service) setupConsumers() {
	if s.store == nil {
		log.Panic("data store must be available for state sinks")
	}
	for _, scope := range s.store.ListAllScope() {
		log.WithFields(log.Fields{"scope": scope}).Debug(
			"creating sink for scope")
		s.manager.AddSink(scope, &sinkCallback{store: s.store})
	}
}

// Run handles the service execution.
func (s *Service) Run(ctx context.Context, wg *sync.WaitGroup) {
	s.setupStore()
	s.setupIdentityClient()
	s.setupHandler()
	s.setupConsumers()
	s.setupTransports()

	defer s.store.Unload()
	defer s.transportManager.StopAll()
	defer s.manager.Shutdown()
	defer func() {
		if s.identityClient != nil {
			err := s.identityClient.Close()
			if err != nil {
				log.WithError(err).Error("failed to close identity client")
			}
		}
	}()

	defer wg.Done()
	log.WithFields(log.Fields{"context": "service.run"}).Debug("starting")

	// Start transports
	if err := s.transportManager.StartAll(ctx, wg); err != nil {
		log.WithError(err).Fatal("Failed to start transports")
	}

	wg.Add(2)
	go s.runHealth(ctx, wg)
	go s.manager.Run(ctx, wg)

	<-ctx.Done()

	log.WithFields(log.Fields{"context": "service.run"}).Debug("exiting")
}

func (s *Service) runHealth(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	log.WithFields(log.Fields{"context": "service.run-health"}).Debug(
		"starting")

	port, err := strconv.Atoi(util.Getenv("PLANTD_STATE_HEALTH_PORT", "8081"))
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Fatal(
			"failed to parse health port")
	}

	go func() {
		h := health.New(
			health.Health{
				Version:   "1",
				ReleaseID: "1.0.0-SNAPSHOT",
			},
		)

		// Add custom health status endpoint that includes identity service
		http.HandleFunc("/health", s.healthStatusHandler)
		http.HandleFunc("/healthz", h.Handler)

		if err := http.ListenAndServe(fmt.Sprintf(":%d", port),
			nil); err != nil {
			log.WithFields(log.Fields{"error": err}).Fatal(
				"failed to start health server")
		}
	}()

	<-ctx.Done()

	log.WithFields(log.Fields{"context": "service.run-health"}).Debug("exiting")
}

func (s *Service) healthStatusHandler(w http.ResponseWriter, _ *http.Request) {
	status := s.getHealthStatus()

	w.Header().Set("Content-Type", "application/json")

	// Set HTTP status based on overall health
	if status["status"] == "healthy" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	// Enhanced JSON response with detailed identity status
	identityStatus := status["identity"].(map[string]interface{})
	_, err := fmt.Fprintf(w, `{
		"status": "%s",
		"store": %t,
		"identity": {
			"status": "%s",
			"connected": %t,
			"message": "%s"
		},
		"auth_mode": "%s",
		"timestamp": "%s"
	}`,
		status["status"],
		status["store"],
		identityStatus["status"],
		identityStatus["connected"],
		identityStatus["message"],
		status["auth_mode"],
		status["timestamp"])
	if err != nil {
		log.WithError(err).Error("failed to write health status")
	}
}

func (s *Service) getHealthStatus() map[string]interface{} {
	storeHealthy := s.store != nil
	identityStatus := s.getIdentityStatus()

	// Identity service is optional, so don't fail health check if unavailable
	overallStatus := "healthy"
	if !storeHealthy {
		overallStatus = "unhealthy"
	}

	return map[string]interface{}{
		"status":    overallStatus,
		"store":     storeHealthy,
		"identity":  identityStatus,
		"auth_mode": s.getAuthMode(),
		"timestamp": time.Now().Format(time.RFC3339),
	}
}

func (s *Service) isIdentityHealthy() bool { //nolint:unused
	if s.identityClient == nil {
		return false
	}

	// Try a quick health check with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.identityClient.HealthCheck(ctx)
	return err == nil
}

func (s *Service) getIdentityStatus() map[string]interface{} {
	if s.identityClient == nil {
		return map[string]interface{}{
			"status":    "disabled",
			"connected": false,
			"message":   "Identity service not configured or unavailable",
		}
	}

	// Try a quick health check with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.identityClient.HealthCheck(ctx)
	if err != nil {
		return map[string]interface{}{
			"status":    "unhealthy",
			"connected": false,
			"message":   "Identity service health check failed: " + err.Error(),
		}
	}

	return map[string]interface{}{
		"status":    "healthy",
		"connected": true,
		"message":   "Identity service is responsive",
	}
}

func (s *Service) getAuthMode() string {
	if s.authMiddleware == nil {
		return "disabled"
	}
	return "enabled"
}

// Note: Legacy MDP processMessage function removed - all requests now handled via gRPC

// RegisterCallback is a pointless wrapper around the handler.
func (s *Service) RegisterCallback(name string,
	callback HandlerCallback) error {
	return s.handler.AddCallback(name, callback)
}
