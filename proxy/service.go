package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	phttp "github.com/geoffjay/plantd/core/http"
	"github.com/geoffjay/plantd/core/util"

	"github.com/gin-gonic/gin"
	"github.com/nelkinda/health-go"
	log "github.com/sirupsen/logrus"
)

// Service type for REST API.
type Service struct {
	port int
	bind string
}

// NewService constructs and instance of a service type.
func NewService(port int, bind string) *Service {
	return &Service{
		port,
		bind,
	}
}

func (s *Service) run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	wg.Add(1)
	go s.runHealth(ctx, wg)

	go func() {
		gin.SetMode(gin.ReleaseMode)
		r := gin.New()

		r.Use(gin.Recovery())
		r.Use(phttp.LoggerMiddleware())

		initializeRoutes(r)

		if err := r.Run(fmt.Sprintf("%s:%d", s.bind, s.port)); err != nil {
			panic(err)
		}
	}()

	<-ctx.Done()
	log.Debug("exiting service")
}

func (s *Service) runHealth(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	log.WithFields(log.Fields{"context": "service.run-health"}).Debug(
		"starting")

	port, err := strconv.Atoi(util.Getenv("PLANTD_PROXY_HEALTH_PORT", "8081"))
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
		if err := http.ListenAndServe(
			fmt.Sprintf(":%d", port),
			nil,
		); err != nil {
			log.WithFields(
				log.Fields{"error": err},
			).Fatal(
				"failed to start health server",
			)
		}
	}()

	<-ctx.Done()

	log.WithFields(log.Fields{"context": "service.run-health"}).Debug("exiting")
}
