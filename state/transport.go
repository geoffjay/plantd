package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/geoffjay/plantd/core/mdp"
	"github.com/geoffjay/plantd/gen/proto/go/plantd/state/v1/statev1connect"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// Transport defines the interface for different transport layers
type Transport interface {
	// Start initializes and starts the transport layer
	Start(ctx context.Context, wg *sync.WaitGroup) error

	// Stop gracefully shuts down the transport layer
	Stop() error

	// Name returns the transport name for logging
	Name() string
}

// MDPTransport implements the MDP transport layer
type MDPTransport struct {
	worker   *mdp.Worker
	handler  *Handler
	endpoint string
	service  string
}

// NewMDPTransport creates a new MDP transport
func NewMDPTransport(endpoint, service string, handler *Handler) *MDPTransport {
	return &MDPTransport{
		handler:  handler,
		endpoint: endpoint,
		service:  service,
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

				// Process message through handler
				response := t.processMessage(message)

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

func (t *MDPTransport) processMessage(message []string) []string {
	if len(message) < 2 {
		return []string{"ERROR", "Invalid message format"}
	}

	command := message[0]
	args := message[1:]

	// Convert to handler format and execute
	if callback, exists := t.handler.callbacks[command]; exists {
		result, err := callback.Execute(args[0])
		if err != nil {
			return []string{"ERROR", err.Error()}
		}
		return []string{"OK", string(result)}
	}

	return []string{"ERROR", "Unknown command: " + command}
}

// GRPCTransport implements the gRPC transport layer
type GRPCTransport struct {
	server     *http.Server
	store      *Store
	port       string
	grpcServer *StateGRPCServer
}

// NewGRPCTransport creates a new gRPC transport
func NewGRPCTransport(port string, store *Store) *GRPCTransport {
	return &GRPCTransport{
		store: store,
		port:  port,
	}
}

func (t *GRPCTransport) Start(ctx context.Context, wg *sync.WaitGroup) error {
	// Create gRPC server
	t.grpcServer = NewStateGRPCServer(t.store)

	// Create HTTP mux
	mux := http.NewServeMux()

	// Register gRPC service
	path, handler := statev1connect.NewStateServiceHandler(t.grpcServer)
	mux.Handle(path, handler)

	// Create HTTP client for MDP compatibility
	httpClient := &http.Client{}
	grpcClient := statev1connect.NewStateServiceClient(httpClient, "http://localhost:"+t.port)

	// Add MDP compatibility layer
	mdpHandler := NewMDPCompatibilityHandler(grpcClient)
	mux.Handle("/mdp/", mdpHandler.ConvertMDPRequestToHTTP())

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","service":"state","transport":"grpc"}`)
	})

	// Add status endpoint
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		scopes := t.store.ListAllScope()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"scopes":%d,"service":"state","mode":"grpc","transport":"grpc"}`, len(scopes))
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
			"MDP compatibility: http://localhost:" + t.port + "/mdp/",
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
