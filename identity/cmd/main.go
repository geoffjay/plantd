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
	"github.com/geoffjay/plantd/identity/internal"
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
	service, err := internal.NewService(cfg, db)
	if err != nil {
		log.Fatalf("failed to initialize service: %v", err)
	}

	fields := log.Fields{"service": "identity", "context": "main"}

	ctx, cancelFunc := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	// Start service
	wg.Add(1)
	go func() {
		if err := service.Run(ctx, wg); err != nil {
			log.WithError(err).Error("Service run failed")
		}
	}()

	log.WithFields(fields).Info("PlantD Identity Service starting")
	log.WithFields(fields).Infof("environment: %s", cfg.Env)
	log.WithFields(fields).Infof("database driver: %s", cfg.Database.Driver)

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
	if len(os.Args) > 1 {
		r := regexp.MustCompile("^-V$|(-{2})?version$")
		if r.Match([]byte(os.Args[1])) {
			log.Info(core.VERSION)
		}
		os.Exit(0)
	}
}
