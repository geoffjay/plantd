package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

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
