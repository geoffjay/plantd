package bus

import (
	"bytes"
	"context"
	"sync"

	log "github.com/sirupsen/logrus"
	czmq "github.com/zeromq/goczmq/v4"
)

// Source type to function as a bus publisher device.
type Source struct {
	endpoint string
	envelope string
	running  bool
	queue    chan []byte
}

var shutdownCommand = []byte{0x0D, 0x0E, 0x0A, 0x0D}

// NewSource constructs an instance of a message bus sink.
func NewSource(endpoint, envelope string) *Source {
	return &Source{
		endpoint: endpoint,
		envelope: envelope,
		running:  false,
		queue:    make(chan []byte),
	}
}

func (s *Source) defaultFields(err error) log.Fields {
	fields := log.Fields{
		"endpoint": s.endpoint,
		"envelope": s.envelope,
	}
	if err != nil {
		fields["err"] = err
	}
	return fields
}

// Run the source routine.
func (s *Source) Run(ctx context.Context, wg *sync.WaitGroup) {
	var publisher *czmq.Sock
	var err error

	defer wg.Done()

	if publisher, err = czmq.NewPub(s.endpoint); err != nil {
		log.WithFields(s.defaultFields(err)).Panic("publisher create")
	}
	log.WithFields(s.defaultFields(nil)).Debug("created message queue source socket")
	defer publisher.Destroy()

	s.running = true

	go func() {
		for message := range s.queue {
			if bytes.Equal(message, shutdownCommand) {
				log.Debug("received shutdown command")
				break
			}

			frame := append([]byte(s.envelope), message...)
			if err := publisher.SendFrame(frame, 0); err != nil {
				log.WithFields(s.defaultFields(err)).Panic("send error")
			}
		}
		log.Debug("source message queue handler stopped")
	}()

	<-ctx.Done()
	log.Debug("source context done")
	s.Stop()
}

// Shutdown gracefully shuts down the source.
func (s *Source) Shutdown() {
	if s.running {
		s.queue <- shutdownCommand
	}
}

// Stop sets the flag to shutdown the loop handling the message queue.
func (s *Source) Stop() {
	s.running = false
	log.Debug("source closing connection")
	close(s.queue)
}

// Running is used to check if the message queue handler should be running.
func (s *Source) Running() bool {
	return s.running
}

// QueueMessage adds a message to the source's message queue.
func (s *Source) QueueMessage(message []byte) {
	s.queue <- message
}
