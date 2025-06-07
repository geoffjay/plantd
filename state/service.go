package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/geoffjay/plantd/core/mdp"
	"github.com/geoffjay/plantd/core/util"

	"github.com/nelkinda/health-go"
	log "github.com/sirupsen/logrus"
)

// Service defines the service type.
type Service struct {
	handler *Handler
	manager *Manager
	store   *Store
	worker  *mdp.Worker
}

// NewService creates an instance of the service.
func NewService() *Service {
	return &Service{
		manager: NewManager(">tcp://localhost:11001"),
	}
}

func (s *Service) setupStore() {
	s.store = NewStore()
	path := util.Getenv("PLANTD_STATE_DB", "plantd-state.db")
	if err := s.store.Load(path); err != nil {
		log.WithFields(log.Fields{"err": err}).Panic("failed to setup KV store")
	}
}

func (s *Service) setupHandler() {
	var err error
	s.handler = NewHandler()
	err = s.RegisterCallback("create-scope", &createScopeCallback{
		name: "create-scope", store: s.store, manager: s.manager,
	})
	if err != nil {
		panic(err)
	}
	err = s.RegisterCallback("delete-scope", &deleteScopeCallback{
		name: "delete-scope", store: s.store, manager: s.manager,
	})
	if err != nil {
		panic(err)
	}
	err = s.RegisterCallback("delete", &deleteCallback{
		name: "delete", store: s.store})
	if err != nil {
		panic(err)
	}
	err = s.RegisterCallback("get", &getCallback{name: "get", store: s.store})
	if err != nil {
		panic(err)
	}
	err = s.RegisterCallback("set", &setCallback{name: "set", store: s.store})
	if err != nil {
		panic(err)
	}
}

func (s *Service) setupWorker() {
	var err error
	endpoint := util.Getenv("PLANTD_STATE_BROKER_ENDPOINT",
		"tcp://127.0.0.1:9797")
	if s.worker, err = mdp.NewWorker(endpoint, "org.plantd.State"); err != nil {
		log.WithFields(log.Fields{"err": err}).Panic(
			"failed to setup message queue worker")
	}
}

func (s *Service) setupConsumers() {
	if s.store == nil {
		log.Panic("data store must be available for state sinks")
	}
	for _, scope := range s.store.ListAllScope() {
		log.WithFields(log.Fields{"scope": scope}).Debug(
			"creating sink for scope")
		s.manager.AddSink(scope, &sinkCallback{store: s.store})
	}
}

// Run handles the service execution.
func (s *Service) Run(ctx context.Context, wg *sync.WaitGroup) {
	s.setupStore()
	s.setupHandler()
	s.setupConsumers()
	s.setupWorker()

	defer s.store.Unload()
	defer s.worker.Close()
	defer s.manager.Shutdown()

	defer wg.Done()
	log.WithFields(log.Fields{"context": "service.run"}).Debug("starting")

	wg.Add(3)
	go s.runHealth(ctx, wg)
	go s.manager.Run(ctx, wg)
	go s.runWorker(ctx, wg)

	<-ctx.Done()

	log.WithFields(log.Fields{"context": "service.run"}).Debug("exiting")
}

func (s *Service) runHealth(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	log.WithFields(log.Fields{"context": "service.run-health"}).Debug(
		"starting")

	port, err := strconv.Atoi(util.Getenv("PLANTD_STATE_HEALTH_PORT", "8081"))
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

func (s *Service) runWorker(ctx context.Context, wg *sync.WaitGroup) {
	var err error
	fields := log.Fields{"context": "service.worker"}
	defer wg.Done()

	go func() {
		var request, reply []string
		for !s.worker.Terminated() {
			log.WithFields(fields).Debug("waiting for request")

			if request, err = s.worker.Recv(reply); err != nil {
				log.WithFields(log.Fields{"error": err}).Error(
					"failed while receiving request")
			}

			log.WithFields(log.Fields{
				"context": "service.worker",
				"request": request,
			}).Debug("received request")

			if len(request) == 0 {
				log.WithFields(fields).Debug("received request is empty")
				continue
			}

			msgType := request[0]

			// Reset reply
			reply = []string{}
			for _, part := range request[1:] {
				log.WithFields(log.Fields{
					"context": "worker",
					"part":    part,
				}).Debug("processing message")
				var data []byte
				switch msgType {
				case "create-scope", "delete-scope", "delete", "get", "set":
					log.Tracef("part: %s", part)
					if data, err = s.handler.callbacks[msgType].Execute(
						part); err != nil {
						log.WithFields(log.Fields{
							"context": "service.worker",
							"type":    msgType,
							"error":   err,
						}).Warn("message failed")
						break
					}
					log.Tracef("data: %s", data)
				default:
					log.Error("invalid message type provided")
				}

				reply = append(reply, string(data))
			}

			log.Tracef("reply: %+v", reply)
		}
	}()

	<-ctx.Done()
	s.worker.Shutdown()

	log.WithFields(fields).Debug("exiting")
}

// RegisterCallback is a pointless wrapper around the handler.
func (s *Service) RegisterCallback(name string,
	callback HandlerCallback) error {
	return s.handler.AddCallback(name, callback)
}
