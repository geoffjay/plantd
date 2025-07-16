package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/geoffjay/plantd/core/mdp"
	"github.com/geoffjay/plantd/gen/proto/go/plantd/broker/v1/brokerv1connect"
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

// MDPTransport implements the MDP transport layer for broker
type MDPTransport struct {
	worker   *mdp.Worker
	broker   *mdp.Broker
	handler  *Handler
	endpoint string
	service  string
}

// NewMDPTransport creates a new MDP transport for broker
func NewMDPTransport(endpoint, service string, handler *Handler, broker *mdp.Broker) *MDPTransport {
	return &MDPTransport{
		handler:  handler,
		endpoint: endpoint,
		service:  service,
		broker:   broker,
	}
}

func (t *MDPTransport) Start(ctx context.Context, wg *sync.WaitGroup) error {
	var err error

	// Create MDP worker for client communication
	t.worker, err = mdp.NewWorker(t.endpoint, t.service)
	if err != nil {
		log.WithFields(log.Fields{
			"transport": t.Name(),
			"endpoint":  t.endpoint,
			"service":   t.service,
			"error":     err,
		}).Error("Failed to create MDP worker")
		return fmt.Errorf("failed to create MDP worker: %w", err)
	}

	log.WithFields(log.Fields{
		"transport": t.Name(),
		"endpoint":  t.endpoint,
		"service":   t.service,
	}).Info("Starting MDP transport")

	// Start the worker in a goroutine
	go func() {
		defer wg.Done()
		defer t.worker.Close()

		var request, reply []string
		for !t.worker.Terminated() {
			// Wait for requests
			if request, err = t.worker.Recv(reply); err != nil {
				log.WithFields(log.Fields{
					"transport": t.Name(),
					"error":     err,
				}).Error("Failed to receive MDP request")
				continue
			}

			if len(request) == 0 {
				continue
			}

			log.WithFields(log.Fields{
				"transport": t.Name(),
				"request":   request,
			}).Debug("Processing MDP request")

			// Process the message
			reply = t.processMessage(request)

			log.WithFields(log.Fields{
				"transport": t.Name(),
				"reply":     reply,
			}).Debug("Sending MDP reply")
		}
	}()

	return nil
}

func (t *MDPTransport) Stop() error {
	log.WithFields(log.Fields{
		"transport": t.Name(),
	}).Info("Stopping MDP transport")

	if t.worker != nil {
		t.worker.Shutdown()
	}

	return nil
}

func (t *MDPTransport) Name() string {
	return "MDP"
}

func (t *MDPTransport) processMessage(message []string) []string {
	if len(message) < 1 {
		return []string{"ERROR", "Invalid message format"}
	}

	command := message[0]
	var args []string
	if len(message) > 1 {
		args = message[1:]
	}

	log.WithFields(log.Fields{
		"transport": t.Name(),
		"command":   command,
		"args":      args,
	}).Debug("Processing broker message")

	// Convert to handler format and execute
	if callback, exists := t.handler.callbacks[command]; exists {
		var msgBody string
		if len(args) > 0 {
			msgBody = args[0]
		}

		result, err := callback.Execute(msgBody)
		if err != nil {
			log.WithFields(log.Fields{
				"transport": t.Name(),
				"command":   command,
				"error":     err,
			}).Error("Handler execution failed")
			return []string{"ERROR", err.Error()}
		}
		return []string{"OK", string(result)}
	}

	log.WithFields(log.Fields{
		"transport": t.Name(),
		"command":   command,
	}).Warn("Unknown command received")

	return []string{"ERROR", "Unknown command: " + command}
}

// GRPCTransport implements the gRPC transport layer for broker
type GRPCTransport struct {
	server     *http.Server
	port       string
	grpcServer *BrokerGRPCServer
}

// NewGRPCTransport creates a new gRPC transport for broker
func NewGRPCTransport(port string) *GRPCTransport {
	return &GRPCTransport{
		port: port,
	}
}

func (t *GRPCTransport) Start(ctx context.Context, wg *sync.WaitGroup) error {
	// Create gRPC server
	t.grpcServer = NewBrokerGRPCServer()

	// Create HTTP mux
	mux := http.NewServeMux()

	// Register gRPC service
	path, handler := brokerv1connect.NewBrokerServiceHandler(t.grpcServer)
	mux.Handle(path, handler)

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","service":"broker","transport":"gRPC"}`)
	})

	// Create HTTP/2 server with h2c (HTTP/2 over cleartext)
	t.server = &http.Server{
		Addr:    ":" + t.port,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	log.WithFields(log.Fields{
		"transport": t.Name(),
		"port":      t.port,
		"addr":      t.server.Addr,
	}).Info("Starting gRPC transport")

	// Start server in goroutine
	go func() {
		defer wg.Done()

		if err := t.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithFields(log.Fields{
				"transport": t.Name(),
				"error":     err,
			}).Error("gRPC transport server error")
		}
	}()

	return nil
}

func (t *GRPCTransport) Stop() error {
	log.WithFields(log.Fields{
		"transport": t.Name(),
	}).Info("Stopping gRPC transport")

	if t.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return t.server.Shutdown(ctx)
	}

	return nil
}

func (t *GRPCTransport) Name() string {
	return "gRPC"
}

// TransportManager manages multiple transport layers
type TransportManager struct {
	transports []Transport
	wg         *sync.WaitGroup
}

// NewTransportManager creates a new transport manager
func NewTransportManager() *TransportManager {
	return &TransportManager{
		transports: make([]Transport, 0),
		wg:         &sync.WaitGroup{},
	}
}

// AddTransport adds a transport to the manager
func (tm *TransportManager) AddTransport(transport Transport) {
	tm.transports = append(tm.transports, transport)
}

// Start starts all transports
func (tm *TransportManager) Start(ctx context.Context) error {
	log.WithFields(log.Fields{
		"transport_count": len(tm.transports),
	}).Info("Starting transport manager")

	for _, transport := range tm.transports {
		tm.wg.Add(1)
		if err := transport.Start(ctx, tm.wg); err != nil {
			log.WithFields(log.Fields{
				"transport": transport.Name(),
				"error":     err,
			}).Error("Failed to start transport")
			return fmt.Errorf("failed to start %s transport: %w", transport.Name(), err)
		}
	}

	return nil
}

// Stop stops all transports
func (tm *TransportManager) Stop() error {
	log.Info("Stopping transport manager")

	var errors []error
	for _, transport := range tm.transports {
		if err := transport.Stop(); err != nil {
			log.WithFields(log.Fields{
				"transport": transport.Name(),
				"error":     err,
			}).Error("Failed to stop transport")
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors stopping transports: %v", errors)
	}

	return nil
}

// Wait waits for all transports to finish
func (tm *TransportManager) Wait() {
	tm.wg.Wait()
}
