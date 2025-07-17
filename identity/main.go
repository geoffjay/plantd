// Package main provides the main entry point for the PlantD identity service.
package main

import (
	"context"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"syscall"

	"github.com/geoffjay/plantd/core"
	plog "github.com/geoffjay/plantd/core/log"
	"github.com/geoffjay/plantd/identity/internal/config"
	"github.com/geoffjay/plantd/identity/internal/models"

	log "github.com/sirupsen/logrus"
)

func main() {
	processArgs()

	// Load configuration
	cfg := config.GetConfig()

	// Initialize logging with configuration
	plog.Initialize(cfg.Log)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("configuration validation failed: %v", err)
	}

	// Initialize database connection
	db, err := config.NewDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// Run auto-migration
	if err := models.AutoMigrate(db); err != nil {
		log.Fatalf("failed to run auto-migration: %v", err)
	}

	// Initialize service
	service, err := NewService(cfg, db)
	if err != nil {
		log.Fatalf("failed to initialize service: %v", err)
	}

	fields := log.Fields{"service": "identity", "context": "main"}

	ctx, cancelFunc := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	// Start service
	wg.Add(1)
	go func() {
		service.Run(ctx, wg)
	}()

	log.WithFields(fields).Info("PlantD Identity Service starting")
	log.WithFields(fields).Infof("environment: %s", cfg.Env)
	log.WithFields(fields).Infof("database driver: %s", cfg.Database.Driver)
	log.WithFields(fields).Infof("transport config - gRPC port: %s", cfg.Transport.GRPCPort)

	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)
	<-termChan

	log.WithFields(fields).Info("shutdown signal received")

	// Graceful shutdown
	service.Shutdown()
	cancelFunc()
	wg.Wait()

	log.WithFields(fields).Info("PlantD Identity Service stopped")
}

func processArgs() {
	log.Infof("PlantD Identity v%s", core.VERSION)

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]

		switch arg {
		case "-v", "--version":
			os.Exit(0)
		case "-h", "--help":
			showHelp()
			os.Exit(0)
		default:
			if matched, _ := regexp.MatchString(`^-`, arg); matched {
				log.Warnf("unknown argument: %s", arg)
			}
		}
	}
}

func showHelp() {
	help := `PlantD Identity Service

USAGE:
    plantd-identity [FLAGS]

FLAGS:
    -h, --help       Show this help message
    -v, --version    Show version information

ENVIRONMENT VARIABLES:
    PLANTD_IDENTITY_CONFIG          Configuration file path
    PLANTD_IDENTITY_ENABLE_MDP      Enable MDP transport (true/false)
    PLANTD_IDENTITY_ENABLE_GRPC     Enable gRPC transport (true/false)
    PLANTD_IDENTITY_GRPC_PORT       gRPC service port
    PLANTD_IDENTITY_BROKER_ENDPOINT MDP broker endpoint

EXAMPLES:
    # Start with default configuration (MDP enabled)
    plantd-identity

    # Start with gRPC enabled
    PLANTD_IDENTITY_ENABLE_GRPC=true plantd-identity

    # Start with both transports
    PLANTD_IDENTITY_ENABLE_MDP=true PLANTD_IDENTITY_ENABLE_GRPC=true plantd-identity`

	log.Info(help)
}
