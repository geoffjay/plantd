package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/geoffjay/plantd/core/util"

	"github.com/nelkinda/health-go"
	log "github.com/sirupsen/logrus"
)

// Service provides the main service coordination for the logger application.
type Service struct {
	manager *Manager
}

// NewService constructs and instance of a service type.
func NewService() *Service {
	return &Service{
		manager: NewManager(),
	}
}

func (s *Service) run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	wg.Add(2)
	go s.runHealth(ctx, wg)
	go s.manager.Run(ctx, wg)

	<-ctx.Done()
	log.Debug("exiting service")
}

func (s *Service) runHealth(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	log.WithFields(log.Fields{"context": "service.run-health"}).Debug(
		"starting")

	port, err := strconv.Atoi(util.Getenv("PLANTD_LOGGER_HEALTH_PORT", "8081"))
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
