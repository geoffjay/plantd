package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/geoffjay/plantd/core/bus"
	cfg "github.com/geoffjay/plantd/core/config"
	"github.com/geoffjay/plantd/core/util"

	"github.com/nelkinda/health-go"
	log "github.com/sirupsen/logrus"
)

// Service defines the service type with gRPC-only transport support.
type Service struct {
	transportManager *TransportManager
	buses            []*bus.Bus
	running          bool
}

// NewService creates an instance of the service.
func NewService() *Service {
	service := &Service{
		buses:            initBuses(),
		running:          false,
		transportManager: NewTransportManager(),
	}

	service.dumpConfig()

	// Initialize gRPC-only transports
	service.initTransports()

	return service
}

func (s *Service) dumpConfig() {
	config := GetConfig()

	json, err := cfg.MarshalConfig(config)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error(
			"failed to marshal config")
	}

	log.WithFields(log.Fields{"context": "service"}).Trace(json)
}

func initBuses() (buses []*bus.Bus) {
	config := GetConfig()

	for _, b := range config.Buses {
		log.WithFields(log.Fields{
			"bus":      b.Name,
			"backend":  b.Backend,
			"frontend": b.Frontend,
			"capture":  b.Capture,
		}).Info("initializing message bus")
		buses = append(buses, bus.NewBus(bus.Config{
			Name:     b.Name,
			Unit:     b.Name,
			Backend:  b.Backend,
			Frontend: b.Frontend,
			Capture:  b.Capture,
		}))
	}

	return
}

// initTransports initializes gRPC transport based on environment variables
func (s *Service) initTransports() {
	// Check environment variables for gRPC configuration
	grpcPort := util.Getenv("PLANTD_BROKER_GRPC_PORT", "8082")

	log.WithFields(log.Fields{
		"grpc_enabled": true,
		"grpc_port":    grpcPort,
	}).Info("Initializing gRPC-only broker transport")

	// Add gRPC transport
	grpcTransport := NewGRPCTransport(grpcPort)
	s.transportManager.AddTransport(grpcTransport)
	log.WithFields(log.Fields{
		"transport": "gRPC",
		"port":      grpcPort,
	}).Info("Added gRPC transport")
}

// Run handles the service execution.
func (s *Service) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer func() {
		if s.transportManager != nil {
			s.transportManager.Stop()
		}
		wg.Done()
	}()

	log.WithFields(log.Fields{"context": "service.run"}).Debug("starting")

	// Start message buses
	wg.Add(len(s.buses))
	for _, item := range s.buses {
		elem := item
		go func() {
			if err := elem.Start(ctx, wg); err != nil {
				log.Error(err)
			}
		}()
	}

	// Start health endpoint
	wg.Add(1)
	go s.runHealth(ctx, wg)

	// Start transport manager (handles gRPC)
	if err := s.transportManager.Start(ctx); err != nil {
		log.WithFields(log.Fields{
			"context": "service.run",
			"error":   err,
		}).Error("failed to start transport manager")
		return
	}

	<-ctx.Done()

	log.WithFields(log.Fields{"context": "service.run"}).Debug("exiting")
}

func (s *Service) runHealth(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	log.WithFields(log.Fields{"context": "service.run-health"}).Debug(
		"starting")

	port, err := strconv.Atoi(util.Getenv("PLANTD_BROKER_HEALTH_PORT", "8081"))
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

// RegisterCallback is a legacy callback registration function (no longer used in gRPC-only mode)
func (s *Service) RegisterCallback(name string, callback HandlerCallback) error {
	log.WithFields(log.Fields{
		"name": name,
	}).Debug("Legacy callback registration ignored in gRPC-only mode")
	return nil
}
