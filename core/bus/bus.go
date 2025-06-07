package bus

import (
	"context"
	"errors"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	czmq "github.com/zeromq/goczmq/v4"
)

// Bus defines the structure of a device for PUB/SUB messaging.
type Bus struct {
	name     string
	unit     string
	backend  string
	frontend string
	capture  string
}

// NewBus instantiates a new PUB/SUB bus type.
func NewBus(name, unit, backend, frontend, capture string) *Bus {
	return &Bus{
		name,
		unit,
		backend,
		frontend,
		capture,
	}
}

// captureThread is used to monitor traffic on a bus for debugging.
// FIXME: This is currently here to test keeping the socket alive.
func (b *Bus) captureThread(done chan bool) {
	pipe, _ := czmq.NewPull(b.capture)

	log.WithFields(log.Fields{"bus": b.name}).Info("capture proxy messages")
	for {
		msg, err := pipe.RecvMessage()
		if err != nil {
			break // Interrupted
		}
		log.WithFields(log.Fields{"bus": b.name}).Tracef("received: %s", msg)
	}

	log.WithFields(log.Fields{"bus": b.name}).Info("capture ended")
	done <- true
}

// Run the bus routine that handles PUB/SUB messages.
// Deprecated: this has been replaced by Start and will be removed in the future.
func (b *Bus) Run(done chan bool) {
	doneCapture := make(chan bool, 1)

	go b.captureThread(doneCapture)

	time.Sleep(100 * time.Millisecond)
	proxy := czmq.NewProxy()
	defer proxy.Destroy()

	if err := proxy.SetFrontend(czmq.XSub, b.frontend); err != nil {
		log.WithFields(log.Fields{
			"bus":      b.name,
			"endpoint": b.frontend,
		}).Error("failed to connect frontend to proxy")
		done <- true
	}
	log.WithFields(log.Fields{
		"bus":      b.name,
		"endpoint": b.frontend,
	}).Info("frontend connected")

	if err := proxy.SetBackend(czmq.XPub, b.backend); err != nil {
		log.WithFields(log.Fields{
			"bus":      b.name,
			"endpoint": b.backend,
		}).Error("failed to connect backend to proxy")
		done <- true
	}
	log.WithFields(log.Fields{
		"bus":      b.name,
		"endpoint": b.backend,
	}).Info("backend connected")

	if err := proxy.SetCapture(b.capture); err != nil {
		log.WithFields(log.Fields{
			"bus":      b.name,
			"endpoint": b.capture,
		}).Error("failed to connect capture to proxy")
		done <- true
	}
	log.WithFields(log.Fields{
		"bus":      b.name,
		"endpoint": b.capture,
	}).Info("capture connected")

	log.WithFields(log.Fields{"bus": b.name}).Info("proxy blocking")
	<-doneCapture

	done <- true
	log.WithFields(log.Fields{"bus": b.name}).Info("proxy exiting")
}

// Start launches the PUB/SUB bus.
func (b *Bus) Start(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	done := make(chan bool)
	errc := make(chan error)

	go func() {
		var err error

		proxy := czmq.NewProxy()
		if proxy == nil {
			err = errors.New("failed to create proxy")
			log.Error(err)
			errc <- err
		}
		defer proxy.Destroy()

		fields := log.Fields{
			"bus":      b.name,
			"backend":  b.backend,
			"frontend": b.frontend,
			"capture":  b.capture,
		}

		if err = proxy.SetFrontend(czmq.Sub, b.frontend); err != nil {
			log.WithFields(fields).Error("failed to connect frontend to proxy")
			errc <- err
		}
		log.WithFields(fields).Info("frontend connected")

		if err = proxy.SetBackend(czmq.XPub, b.backend); err != nil {
			log.WithFields(fields).Error("failed to connect backend to proxy")
			errc <- err
		}
		log.WithFields(fields).Info("backend connected")

		if err = proxy.SetCapture(b.capture); err != nil {
			log.WithFields(fields).Error("failed to connect capture to proxy")
			errc <- err
		}
		log.WithFields(log.Fields{"bus": b.name, "endpoint": b.capture}).Info("capture connected")

		<-done
	}()

	log.Debug("waiting for bus to complete")

	for {
		select {
		case err := <-errc:
			log.WithFields(log.Fields{"error": err}).Error("received error from bus")
			close(done)
			return err
		case <-ctx.Done():
			log.Debug("exiting bus")
			done <- true
			return nil
		}
	}
}
