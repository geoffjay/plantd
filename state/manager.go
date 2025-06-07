package main

import (
	"context"
	"sync"

	"github.com/geoffjay/plantd/core/bus"

	log "github.com/sirupsen/logrus"
)

const (
	// SinkAdded is the signal ID for when a sink has been added.
	SinkAdded = iota

	// SinkRemoved is the signal ID for when a sink has been removed.
	SinkRemoved

	// SinkShutdown is the signal ID that's raised on shutdown.
	SinkShutdown
)

// SinkChanBuffer controls the number of sinks that can be created before a
// deadlock occurs. This should actually be fixed to not have a deadlock,
// but that's a problem for another day.
const SinkChanBuffer = 20

// Manager is used to control how some devices are managed.
type Manager struct {
	sinkEndpoint string
	sinkList     map[string]*bus.Sink
	sinkChan     chan event
}

type event struct {
	id    int
	scope string
}

// NewManager creates an instance of the manager.
func NewManager(endpoint string) *Manager {
	return &Manager{
		sinkEndpoint: endpoint,
		sinkList:     make(map[string]*bus.Sink),
		sinkChan:     make(chan event, SinkChanBuffer),
	}
}

// AddSink creates a new message consumer sink and adds it to the list by name.
func (m *Manager) AddSink(scope string, callback bus.SinkCallback) {
	if _, ok := m.sinkList[scope]; ok {
		log.WithFields(
			log.Fields{"scope": scope},
		).Debug("scope with that name already exists")
		return
	}
	sink := bus.NewSink(m.sinkEndpoint, scope)
	sink.SetHandler(&bus.SinkHandler{Callback: callback})
	m.sinkList[scope] = sink
	m.sinkChan <- event{id: SinkAdded, scope: scope}
}

// RemoveSink removes a consumer sink from the list by name if it exists.
func (m *Manager) RemoveSink(scope string) {
	if _, ok := m.sinkList[scope]; ok {
		defer m.sinkList[scope].Stop()
		delete(m.sinkList, scope)
	} else {
		log.WithFields(
			log.Fields{"scope": scope},
		).Debug("scope with that name doesn't exist")
	}
	m.sinkChan <- event{id: SinkRemoved, scope: scope}
}

// Run launches all routines that need to be run and monitors for changes.
func (m *Manager) Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	log.WithFields(log.Fields{"context": "manager.run"}).Debug("starting")

	go func() {
	sinkMonitor:
		for event := range m.sinkChan {
			switch event.id {
			case SinkAdded:
				log.WithFields(log.Fields{"scope": event.scope}).Debug(
					"sink add received")
				wg.Add(1)
				go m.sinkList[event.scope].Run(ctx, wg)
			case SinkRemoved:
				log.WithFields(log.Fields{"scope": event.scope}).Debug(
					"sink remove received")
			case SinkShutdown:
				log.Debug("sink shutdown received")
				break sinkMonitor
			}
		}
	}()

	<-ctx.Done()
	m.sinkChan <- event{id: SinkShutdown, scope: ""}

	log.WithFields(log.Fields{"context": "manager.run"}).Debug("exiting")
}

// Shutdown stops any running routines.
func (m *Manager) Shutdown() {
	for scope := range m.sinkList {
		m.RemoveSink(scope)
	}

	log.WithFields(log.Fields{"context": "manager.shutdown"}).Debug(
		"terminating")
}
