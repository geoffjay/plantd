package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/geoffjay/plantd/core/mdp"
	identityv1connect "github.com/geoffjay/plantd/gen/proto/go/plantd/identity/v1/identityv1connect"
	"github.com/geoffjay/plantd/identity/internal/handlers"

	log "github.com/sirupsen/logrus"
)

// Transport defines the interface for different transport layers
type Transport interface {
	Start(ctx context.Context, wg *sync.WaitGroup) error
	Stop() error
	Name() string
}

// MDPTransport implements the MDP (ZeroMQ) transport layer
type MDPTransport struct {
	worker          *mdp.Worker
	endpoint        string
	service         string
	handlerRegistry *handlers.HandlerRegistry
}

// NewMDPTransport creates a new MDP transport
func NewMDPTransport(endpoint, service string, handlerRegistry *handlers.HandlerRegistry) *MDPTransport {
	return &MDPTransport{
		endpoint:        endpoint,
		service:         service,
		handlerRegistry: handlerRegistry,
	}
}

func (t *MDPTransport) Start(ctx context.Context, wg *sync.WaitGroup) error {
	var err error

	// Create MDP worker
	t.worker, err = mdp.NewWorker(t.endpoint, t.service)
	if err != nil {
		return fmt.Errorf("failed to create MDP worker: %w", err)
	}

	log.WithFields(log.Fields{
		"transport": "MDP",
		"endpoint":  t.endpoint,
		"service":   t.service,
	}).Info("Starting MDP transport")

	// Start worker in goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer t.worker.Close()

		for {
			select {
			case <-ctx.Done():
				log.WithField("transport", "MDP").Info("MDP transport context cancelled, shutting down")
				return
			default:
				// Receive message from broker
				message, err := t.worker.Recv([]string{})
				if err != nil {
					log.WithError(err).Error("Error receiving MDP message")
					continue
				}

				// Process message through handler registry
				response := t.processMessage(ctx, message)

				// Send response back through worker
				if err := t.worker.Reply(response); err != nil {
					log.WithError(err).Error("Error sending MDP response")
				}
			}
		}
	}()

	return nil
}

func (t *MDPTransport) Stop() error {
	if t.worker != nil {
		t.worker.Close()
	}
	return nil
}

func (t *MDPTransport) Name() string {
	return "MDP"
}

func (t *MDPTransport) processMessage(ctx context.Context, message []string) []string {
	log.WithFields(log.Fields{
		"message_length": len(message),
		"raw_message":    message,
	}).Debug("Identity service received MDP message")

	if len(message) < 1 {
		log.Warn("Received empty MDP message")
		return []string{}
	}

	// Extract service name and data from message
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

	log.WithFields(log.Fields{
		"extracted_service": serviceName,
		"message_data":      messageData,
		"message_len":       len(messageData),
	}).Debug("Processing MDP message")

	// Route to appropriate handler
	response, err := t.handlerRegistry.HandleMessage(ctx, serviceName, messageData)
	if err != nil {
		log.WithError(err).Error("Failed to handle MDP message")
		return []string{fmt.Sprintf(`{"error": "Internal server error: %s"}`, err.Error())}
	}

	log.WithFields(log.Fields{
		"service":       serviceName,
		"response_len":  len(response),
		"response_type": fmt.Sprintf("%T", response),
	}).Debug("Identity service returning response")

	return response
}

// GRPCTransport implements the gRPC transport layer
type GRPCTransport struct {
	server     *http.Server
	port       string
	grpcServer *IdentityGRPCServer
}

// NewGRPCTransport creates a new gRPC transport
func NewGRPCTransport(port string, grpcServer *IdentityGRPCServer) *GRPCTransport {
	return &GRPCTransport{
		port:       port,
		grpcServer: grpcServer,
	}
}

func (t *GRPCTransport) Start(ctx context.Context, wg *sync.WaitGroup) error {
	// Create HTTP mux
	mux := http.NewServeMux()

	// Register gRPC service
	path, handler := identityv1connect.NewIdentityServiceHandler(t.grpcServer)
	mux.Handle(path, handler)

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","service":"identity","transport":"grpc"}`)
	})

	// Add status endpoint
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		// For Identity service, we can provide user count or service info
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"service":"identity","mode":"grpc","transport":"grpc","status":"running"}`)
	})

	// Create HTTP/2 server with h2c (HTTP/2 without TLS)
	t.server = &http.Server{
		Addr:    ":" + t.port,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	log.WithFields(log.Fields{
		"transport": "gRPC",
		"port":      t.port,
		"endpoints": []string{
			"gRPC: http://localhost:" + t.port,
			"Health: http://localhost:" + t.port + "/health",
		},
	}).Info("Starting gRPC transport")

	// Start server in goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()

		if err := t.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Error("gRPC server error")
		}
	}()

	// Handle graceful shutdown
	go func() {
		<-ctx.Done()
		log.WithField("transport", "gRPC").Info("gRPC transport context cancelled, shutting down")

		if err := t.server.Shutdown(context.Background()); err != nil {
			log.WithError(err).Error("Error shutting down gRPC server")
		}
	}()

	return nil
}

func (t *GRPCTransport) Stop() error {
	if t.server != nil {
		return t.server.Shutdown(context.Background())
	}
	return nil
}

func (t *GRPCTransport) Name() string {
	return "gRPC"
}

// TransportManager manages multiple transport layers
type TransportManager struct {
	transports []Transport
}

// NewTransportManager creates a new transport manager
func NewTransportManager() *TransportManager {
	return &TransportManager{
		transports: make([]Transport, 0),
	}
}

// AddTransport adds a transport to the manager
func (tm *TransportManager) AddTransport(transport Transport) {
	tm.transports = append(tm.transports, transport)
}

// StartAll starts all registered transports
func (tm *TransportManager) StartAll(ctx context.Context, wg *sync.WaitGroup) error {
	for _, transport := range tm.transports {
		if err := transport.Start(ctx, wg); err != nil {
			return fmt.Errorf("failed to start %s transport: %w", transport.Name(), err)
		}
	}
	return nil
}

// StopAll stops all registered transports
func (tm *TransportManager) StopAll() error {
	var lastErr error
	for _, transport := range tm.transports {
		if err := transport.Stop(); err != nil {
			log.WithError(err).WithField("transport", transport.Name()).Error("Error stopping transport")
			lastErr = err
		}
	}
	return lastErr
}
