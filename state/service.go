package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/geoffjay/plantd/core/mdp"
	"github.com/geoffjay/plantd/core/util"
	"github.com/geoffjay/plantd/identity/pkg/client"
	"github.com/geoffjay/plantd/state/auth"

	"github.com/nelkinda/health-go"
	log "github.com/sirupsen/logrus"
)

// Service defines the service type.
type Service struct {
	handler        *Handler
	manager        *Manager
	store          *Store
	worker         *mdp.Worker
	identityClient *client.Client
	authMiddleware *auth.AuthMiddleware
}

// NewService creates an instance of the service.
func NewService() *Service {
	return &Service{
		manager: NewManager(">tcp://localhost:11001"),
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

func (s *Service) setupWorker() {
	var err error
	endpoint := util.Getenv("PLANTD_STATE_BROKER_ENDPOINT",
		"tcp://127.0.0.1:9797")
	if s.worker, err = mdp.NewWorker(endpoint, "org.plantd.State"); err != nil {
		log.WithFields(log.Fields{"err": err}).Panic(
			"failed to setup message queue worker")
	}
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
	s.setupWorker()

	defer s.store.Unload()
	defer s.worker.Close()
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

	wg.Add(3)
	go s.runHealth(ctx, wg)
	go s.manager.Run(ctx, wg)
	go s.runWorker(ctx, wg)

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

func (s *Service) runWorker(ctx context.Context, wg *sync.WaitGroup) {
	var err error
	fields := log.Fields{"context": "service.worker"}
	defer wg.Done()

	go func() {
		var request, reply []string
		for !s.worker.Terminated() {
			log.WithFields(fields).Debug("waiting for request")

			if request, err = s.worker.Recv(reply); err != nil {
				log.WithFields(log.Fields{"error": err}).Error(
					"failed while receiving request")
				continue
			}

			log.WithFields(log.Fields{
				"context": "service.worker",
				"request": request,
			}).Debug("received request")

			if len(request) == 0 {
				log.WithFields(fields).Debug("received request is empty")
				continue
			}

			// Process the message - expecting format: [client_id, operation, data...]
			// The client_id is included in the message for reply routing
			reply = s.processMessage(ctx, request)

			log.WithFields(log.Fields{
				"context": "service.worker",
				"reply":   reply,
			}).Debug("prepared reply")
		}
	}()

	<-ctx.Done()
	s.worker.Shutdown()

	log.WithFields(fields).Debug("exiting")
}

// processMessage processes a single MDP message for the state service
func (s *Service) processMessage(_ context.Context, message []string) []string {
	log.WithFields(log.Fields{
		"message_length": len(message),
		"raw_message":    message,
	}).Debug("State service received MDP message")

	if len(message) < 2 {
		log.Warn("Received message with insufficient frames")
		return []string{`{"error": "Invalid message format"}`}
	}

	// Extract operation from message
	// Message format: [operation, ...args]
	operation := message[0]
	args := message[1:]

	log.WithFields(log.Fields{
		"operation": operation,
		"args":      args,
	}).Debug("Processing state service request")

	// Check if the operation is valid
	callback, exists := s.handler.callbacks[operation]
	if !exists {
		log.WithFields(log.Fields{
			"operation": operation,
		}).Error("Invalid operation requested")
		return []string{`{"error": "Invalid operation"}`}
	}

	// Process the arguments and execute the callback
	var data []byte
	var err error

	// For most operations, we need to combine the args into a single string
	// This matches the expected behavior of the original implementation
	if len(args) > 0 {
		argData := args[0] // Most operations expect a single argument
		data, err = callback.Execute(argData)
	} else {
		// Some operations like "list-scopes" don't need arguments
		data, err = callback.Execute("")
	}

	if err != nil {
		log.WithFields(log.Fields{
			"operation": operation,
			"error":     err,
		}).Warn("Operation failed")
		return []string{fmt.Sprintf(`{"error": "Operation failed: %s"}`, err.Error())}
	}

	log.WithFields(log.Fields{
		"operation":    operation,
		"response_len": len(data),
	}).Debug("State service operation completed successfully")

	return []string{string(data)}
}

// RegisterCallback is a pointless wrapper around the handler.
func (s *Service) RegisterCallback(name string,
	callback HandlerCallback) error {
	return s.handler.AddCallback(name, callback)
}
