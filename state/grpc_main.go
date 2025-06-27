package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/geoffjay/plantd/gen/proto/go/plantd/state/v1/statev1connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	// Initialize the store
	store := NewStore()

	// Load the database
	dbPath := os.Getenv("PLANTD_STATE_DB_PATH")
	if dbPath == "" {
		dbPath = "./state.db"
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}

	if err := store.Load(dbPath); err != nil {
		log.Fatalf("Failed to load database: %v", err)
	}
	defer store.Unload()

	// Create gRPC server
	grpcServer := NewStateGRPCServer(store)

	// Create HTTP mux
	mux := http.NewServeMux()

	// Register gRPC service
	path, handler := statev1connect.NewStateServiceHandler(grpcServer)
	mux.Handle(path, handler)

	// Create HTTP client for MDP compatibility
	httpClient := &http.Client{}
	grpcClient := statev1connect.NewStateServiceClient(httpClient, "http://localhost:8080")

	// Add MDP compatibility layer
	mdpHandler := NewMDPCompatibilityHandler(grpcClient)
	mux.Handle("/mdp/", mdpHandler.ConvertMDPRequestToHTTP())

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","service":"state"}`)
	})

	// Add status endpoint
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		scopes := store.ListAllScope()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"scopes":%d,"service":"state","mode":"grpc"}`, len(scopes))
	})

	// Get port from environment
	port := os.Getenv("PLANTD_STATE_GRPC_PORT")
	if port == "" {
		port = "8080"
	}

	// Create HTTP/2 server with h2c (HTTP/2 without TLS)
	server := &http.Server{
		Addr:    ":" + port,
		Handler: h2c.NewHandler(mux, &http2.Server{}),
	}

	log.Printf("Starting State gRPC service on port %s", port)
	log.Printf("gRPC endpoint: http://localhost:%s", port)
	log.Printf("MDP compatibility: http://localhost:%s/mdp/", port)
	log.Printf("Health check: http://localhost:%s/health", port)

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
