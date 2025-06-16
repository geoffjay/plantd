// Package mdp provides Majordomo Protocol implementation.
package mdp

import (
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/geoffjay/plantd/core/util"

	log "github.com/sirupsen/logrus"
	czmq "github.com/zeromq/goczmq/v4"
)

// Broker defines a single broker instance.
type Broker struct {
	Socket       *czmq.Sock               // Socket for clients & workers
	endpoint     string                   // Broker binds to this endpoint
	services     map[string]*Service      // hash of known services
	workers      map[string]*brokerWorker // hash of known workers
	Waiting      []*brokerWorker          // list of waiting workers
	HeartbeatAt  time.Time                // when to send HEARTBEAT
	isBound      bool                     // if the socket is bound to an endpoint
	ErrorChannel chan error
	EventChannel chan Event
	// Request durability support
	requestManager *RequestManager // manages request persistence and retry
	cleanupTicker  *time.Ticker    // periodic cleanup of expired requests
}

// Service defines a single service instance.
type Service struct {
	broker   *Broker         // Broker instance
	name     string          // Service name
	requests [][]string      // list of client requests
	waiting  []*brokerWorker // list of waiting workers
}

// brokerWorker defines a single worker, idle or active.
type brokerWorker struct {
	broker        *Broker   // Broker instance
	idString      string    // ID of worker as string
	identity      string    // ID frame for routing
	service       *Service  // owning service, if known
	expiry        time.Time // expires at unless heartbeat
	totalRequests int64
}

// WorkerInfo is used to return certain information about a worker.
type WorkerInfo struct {
	ID            string `json:"id"`
	Identity      string `json:"identity"`
	ServiceName   string `json:"service-name"`
	TotalRequests int64  `json:"total-requests"`
}

// NewBroker creates a new broker instance.
func NewBroker(endpoint string) (broker *Broker, err error) {
	// Initialize persistence store
	persistenceStore := NewMemoryPersistenceStore()
	requestManager := NewRequestManager(persistenceStore)

	broker = &Broker{
		endpoint:       endpoint,
		services:       make(map[string]*Service),
		workers:        make(map[string]*brokerWorker),
		Waiting:        make([]*brokerWorker, 0),
		HeartbeatAt:    time.Now().Add(HeartbeatInterval),
		isBound:        false,
		ErrorChannel:   make(chan error, 1),
		EventChannel:   make(chan Event),
		requestManager: requestManager,
		cleanupTicker:  time.NewTicker(1 * time.Minute), // cleanup every minute
	}

	// Start cleanup goroutine for expired requests
	go broker.cleanupExpiredRequests()

	return
}

// GetWorkerInfo is used to request all information about connected workers.
func (b *Broker) GetWorkerInfo() []WorkerInfo {
	var info []WorkerInfo
	for _, worker := range b.workers {
		info = append(info, WorkerInfo{
			ID:            worker.idString,
			Identity:      worker.identity,
			ServiceName:   worker.service.name,
			TotalRequests: worker.totalRequests,
		})
	}
	return info
}

// nolint
func initMonitor(socket *czmq.Sock) {
	monitor := czmq.NewMonitor(socket)
	defer monitor.Destroy()

	_ = monitor.Verbose()
	// _ = monitor.Listen("ALL")
	_ = monitor.Listen("CONNECTED")
	_ = monitor.Listen("CONNECT_DELAYED")
	_ = monitor.Listen("CONNECT_RETRIED")
	_ = monitor.Listen("LISTENING")
	_ = monitor.Listen("BIND_FAILED")
	_ = monitor.Listen("ACCEPTED")
	_ = monitor.Listen("ACCEPT_FAILED")
	_ = monitor.Listen("CLOSED")
	_ = monitor.Listen("CLOSE_FAILED")
	_ = monitor.Listen("DISCONNECTED")
	_ = monitor.Listen("MONITOR_STOPPED")
	_ = monitor.Start()

	go (func() {
		poller, _ := czmq.NewPoller()
		defer poller.Destroy()

		if err := poller.Add(monitor.Socket()); err != nil {
			log.Errorf("failed to create poller for the monitor socket: %s", err)
		}

		for {
			socket, err := poller.Wait(1000)
			if err != nil {
				log.Errorf("an error occurred waiting for the monitor socket: %s", err)
			}

			if socket == nil {
				log.Error("no messages received on monitor socket for 1 second")
				continue
			}

			msg, _ := socket.RecvMessage()

			if len(msg) != 3 {
				log.Errorf("expected message with 3 frames, got %v", len(msg))
				continue
			}

			eventName := string(msg[0])
			log.Debugf("received event: %s", eventName)
		}
	})()
}

// Close is used to terminate the broker socket.
func (b *Broker) Close() (err error) {
	// Stop cleanup ticker
	if b.cleanupTicker != nil {
		b.cleanupTicker.Stop()
	}

	// Close request manager
	if b.requestManager != nil {
		_ = b.requestManager.Close()
	}

	if b.isBound && b.Socket != nil {
		err = b.Socket.Unbind(b.endpoint)
		b.Socket.Destroy()
		b.Socket = nil
		b.isBound = false
	}
	close(b.ErrorChannel)
	close(b.EventChannel)

	return
}

// cleanupExpiredRequests periodically cleans up expired requests
func (b *Broker) cleanupExpiredRequests() {
	for range b.cleanupTicker.C {
		if store, ok := b.requestManager.store.(*MemoryPersistenceStore); ok {
			removed := store.CleanupExpiredRequests()
			if removed > 0 {
				log.WithFields(log.Fields{
					"expired_requests": removed,
				}).Debug("cleaned up expired requests")
			}
		}
	}
}

// Bind the broker instance to an endpoint. We can call this multiple times.
// Note that MDP uses a single socket for both clients and workers.
func (b *Broker) Bind() (err error) {
	// creating the socket binds by default
	b.Socket, err = czmq.NewRouter(b.endpoint)
	if err != nil {
		b.ErrorChannel <- err
		log.WithFields(log.Fields{
			"endpoint": b.endpoint,
		}).Error("MDP broker/0.2.0 failed to bind")
		return err
	}

	b.Socket.SetOption(czmq.SockSetRcvhwm(500000))
	runtime.SetFinalizer(b, (*Broker).Close)

	// time.Sleep(1000)
	// initMonitor(b.Socket)

	go func() {
		b.EventChannel <- NewBrokerEvent(fmt.Sprintf("broker bound to endpoint %s", b.endpoint))
	}()

	err = nil
	log.WithFields(log.Fields{
		"endpoint": b.endpoint,
	}).Info("MDP broker/0.2.0 is active")

	b.isBound = true

	return
}

// Run the service.
// nolint: cyclop
func (b *Broker) Run(done chan bool) {
	poller, _ := czmq.NewPoller(b.Socket)

	log.Debug("starting broker...")
	for {
		socket, err := poller.Wait(int(HeartbeatInterval / 1e6))
		if err != nil {
			break
		}
		if socket == nil {
			log.WithFields(log.Fields{
				"timeout (ms)": int(HeartbeatInterval) / 1e6,
			}).Trace("no messages received on broker endpoint for the timeout duration")
		} else {
			recv, _ := socket.RecvMessage()
			msg := byte2DToStringArray(recv)

			// Process next input message, if any
			if len(msg) > 0 {
				// msg, err := b.Socket.RecvMessage()
				// if err != nil {
				// 	break // Interrupted
				// }
				log.WithFields(log.Fields{"data": msg}).Trace("received message")

				// Enhanced debugging: log the raw message structure
				log.WithFields(log.Fields{
					"total_frames": len(msg),
					"raw_frames":   msg,
				}).Debug("processing incoming message")

				sender, msg := util.PopStr(msg)
				log.WithFields(log.Fields{
					"sender":           sender,
					"remaining_frames": len(msg),
				}).Debug("extracted sender")

				_, msg = util.PopStr(msg) // Pop empty delimiter
				log.WithFields(log.Fields{
					"remaining_frames_after_delimiter": len(msg),
				}).Debug("popped empty delimiter")

				header, msg := util.PopStr(msg)
				log.WithFields(log.Fields{
					"header":           header,
					"expected_client":  MdpcClient,
					"expected_worker":  MdpwWorker,
					"remaining_frames": len(msg),
					"remaining_data":   msg,
				}).Debug("extracted header for processing")

				switch header {
				case MdpcClient:
					// Strip the command frame (should be "REQUEST" for MDP v0.2)
					if len(msg) < 1 {
						log.WithFields(log.Fields{
							"sender": sender,
							"msg":    msg,
						}).Error("client message missing command frame")
						continue
					}

					command, msg := util.PopStr(msg)
					log.WithFields(log.Fields{
						"sender":         sender,
						"command":        command,
						"message_frames": len(msg),
					}).Debug("routing to ClientMsg")

					// Validate command is REQUEST for MDP v0.2
					if command != MdpcRequest {
						log.WithFields(log.Fields{
							"sender":           sender,
							"command":          command,
							"expected_command": MdpcRequest,
						}).Warn("invalid client command")
						continue
					}

					b.ClientMsg(sender, msg)
				case MdpwWorker:
					log.WithFields(log.Fields{
						"sender":         sender,
						"message_frames": len(msg),
					}).Debug("routing to WorkerMsg")
					b.WorkerMsg(sender, msg)
				default:
					log.WithFields(log.Fields{
						"header":            header,
						"expected_client":   MdpcClient,
						"expected_worker":   MdpwWorker,
						"sender":            sender,
						"remaining_message": msg,
					}).Warn("invalid message header")
					log.Warnf("invalid message: %s", msg)
				}
			}
		}

		// disconnect and delete any expired workers sending heartbeats to idle workers if needed
		if time.Now().After(b.HeartbeatAt) {
			b.Purge()
			for _, worker := range b.Waiting {
				log.WithFields(log.Fields{
					"service": worker.service.name,
				}).Trace("sending heartbeat to worker")
				if err = worker.Send(MdpwHeartbeat, "", []string{}); err != nil {
					b.ErrorChannel <- err
					log.WithFields(log.Fields{
						"error": err,
					}).Error("failed to send heartbeat message")
				}
			}
			b.HeartbeatAt = time.Now().Add(HeartbeatInterval)
		}
	}

	done <- true
}

// WorkerMsg processes one READY, REPLY, HEARTBEAT or DISCONNECT message sent
// to the broker by a worker.
// nolint: cyclop
func (b *Broker) WorkerMsg(sender string, msg []string) {
	// at least, command
	if len(msg) == 0 {
		log.Error("zero length message")
	}

	command, msg := util.PopStr(msg)
	idString := fmt.Sprintf("%q", sender)
	_, workerReady := b.workers[idString]
	worker := b.workerRequire(sender)
	worker.totalRequests++

	switch command {
	case MdpwReady:
		switch {
		case workerReady:
			// not first command in session
			worker.Delete(true)
		case len(sender) >= 4 /* reserved service name */ && sender[:4] == MMINamespace:
			worker.Delete(true)
		default:
			// attach worker to service and mark as idle
			worker.service = b.ServiceRequire(msg[0])
			worker.Waiting()
		}
	case MdpwPartial:
		if workerReady {
			// remove & save client return envelope and insert the
			// protocol header and service name, then re-wrap envelope.
			client, msg := util.Unwrap(msg)
			snd := stringArrayToByte2D(append(
				[]string{client, MdpcClient, MdpcPartial, worker.service.name}, msg...))
			if err := b.Socket.SendMessage(snd); err != nil {
				b.ErrorChannel <- err
				log.WithFields(log.Fields{"error": err}).Error("failed to send partial message to client")
				return
			}
			// Don't set worker to waiting for partial responses - wait for final
		} else {
			worker.Delete(true)
		}
	case MdpwFinal:
		if workerReady {
			// remove & save client return envelope and insert the
			// protocol header and service name, then re-wrap envelope.
			client, msg := util.Unwrap(msg)
			snd := stringArrayToByte2D(append(
				[]string{client, MdpcClient, MdpcFinal, worker.service.name}, msg...))
			if err := b.Socket.SendMessage(snd); err != nil {
				b.ErrorChannel <- err
				log.WithFields(log.Fields{"error": err}).Error("failed to send final message to client")
				return
			}
			worker.Waiting()
		} else {
			worker.Delete(true)
		}
	case MdpwHeartbeat:
		if workerReady {
			worker.expiry = time.Now().Add(HeartbeatExpiry)
		} else {
			worker.Delete(true)
		}
	case MdpwDisconnect:
		worker.Delete(false)
	default:
		message := fmt.Sprintf("invalid input message %q", msg)
		err := errors.New(message)
		b.ErrorChannel <- err
		log.Error(err)
	}
}

// ClientMsg processes a request coming from a client. We implement MMI requests
// directly here (at present, we implement only the mmi.service request).
// nolint: nestif
func (b *Broker) ClientMsg(sender string, msg []string) {
	// the message should contain the service name and message body
	if len(msg) < 2 {
		err := errors.New("message contains less than 2 frames")
		b.ErrorChannel <- err
		// XXX: this is a panic() in the example
		log.Error(err)
		return
	}

	serviceFrame, msg := util.PopStr(msg)
	service := b.ServiceRequire(serviceFrame)

	// Set reply return identity to client sender
	m := []string{sender, ""}
	msg = append(m, msg...)

	// If we got a MMI service request, process that internally
	if len(serviceFrame) >= 4 && serviceFrame[:4] == MMINamespace {
		var returnCode string
		if serviceFrame == MMIService {
			name := msg[len(msg)-1]
			service, ok := b.services[name]
			if ok && len(service.waiting) > 0 {
				returnCode = "200"
			} else {
				returnCode = "404"
			}
		} else {
			returnCode = "501"
		}

		msg[len(msg)-1] = returnCode

		// remove & save client return envelope and insert the
		// protocol header and service name, then re-wrap envelope.
		client, msg := util.Unwrap(msg)
		snd := stringArrayToByte2D(append([]string{client, MdpcClient, MdpcFinal, serviceFrame}, msg...))
		if err := b.Socket.SendMessage(snd); err != nil {
			b.ErrorChannel <- err
			log.WithFields(log.Fields{"error": err}).Error("failed to send message to client")
		}
	} else {
		// Phase 3: Persist request for durability before dispatching
		request, err := b.requestManager.CreateRequest(sender, serviceFrame, msg)
		if err != nil {
			log.WithFields(log.Fields{
				"error":   err,
				"client":  sender,
				"service": serviceFrame,
			}).Error("failed to persist request")
			b.ErrorChannel <- err
			return
		}

		log.WithFields(log.Fields{
			"request_id": request.ID,
			"client":     sender,
			"service":    serviceFrame,
		}).Debug("persisted client request")

		// Mark as processing and dispatch to service
		if err := b.requestManager.MarkRequestProcessing(request.ID); err != nil {
			log.WithFields(log.Fields{
				"error":      err,
				"request_id": request.ID,
			}).Warn("failed to mark request as processing")
		}

		// else dispatch the message to the requested service
		service.Dispatch(msg)
	}
}

// Purge deletes any idle workers that haven't pinged us in a
// while. We hold workers from oldest to most recent, so we can stop
// scanning whenever we find a live worker. This means we'll mainly stop
// at the first worker, which is essential when we have large numbers of
// workers (since we call this method in our critical path).
func (b *Broker) Purge() {
	now := time.Now()
	for len(b.Waiting) > 0 {
		if b.Waiting[0].expiry.After(now) {
			// worker is alive, we're done here
			break
		}
		log.WithFields(log.Fields{
			"worker": b.Waiting[0].idString,
		}).Debug("deleting expired worker")
		b.Waiting[0].Delete(false)
	}
}

// ServiceRequire is a lazy constructor that locates a service by name, or
// creates a new service if there is no service already with that name.
func (b *Broker) ServiceRequire(serviceFrame string) (service *Service) {
	name := serviceFrame
	service, ok := b.services[name]
	if !ok {
		service = &Service{
			broker:   b,
			name:     name,
			requests: make([][]string, 0),
			waiting:  make([]*brokerWorker, 0),
		}
		b.services[name] = service
		log.Debugf("added service: %s", name)
	}
	return
}

// Dispatch sends requests to waiting workers.
func (s *Service) Dispatch(msg []string) {
	if len(msg) > 0 {
		// queue message if any
		s.requests = append(s.requests, msg)
	}

	s.broker.Purge()
	for len(s.waiting) > 0 && len(s.requests) > 0 {
		var worker *brokerWorker
		worker, s.waiting = popWorker(s.waiting)
		s.broker.Waiting = delWorker(s.broker.Waiting, worker)
		msg, s.requests = util.PopMsg(s.requests)
		if err := worker.Send(MdpwRequest, "", msg); err != nil {
			s.broker.ErrorChannel <- err
			log.WithFields(log.Fields{"error": err}).Error("failed to dispatch request to worker")
		}
	}
}

// workerRequire is a lazy constructor that locates a worker by identity, or
// creates a new worker if there is no worker already with that identity.
func (b *Broker) workerRequire(identity string) (worker *brokerWorker) {
	// b.workers is keyed off worker identity
	idString := fmt.Sprintf("%q", identity)
	worker, ok := b.workers[idString]
	if !ok {
		worker = &brokerWorker{
			broker:   b,
			idString: idString,
			identity: identity,
		}
		b.workers[idString] = worker
		log.WithFields(log.Fields{"id": idString}).Debug("registering new worker")
	}
	return
}

// Delete removes the current worker.
func (w *brokerWorker) Delete(disconnect bool) {
	if disconnect {
		if err := w.Send(MdpwDisconnect, "", []string{}); err != nil {
			w.broker.ErrorChannel <- err
			log.WithFields(log.Fields{"error": err}).Error("failed to send disconnect to worker")
		}
	}

	if w.service != nil {
		w.service.waiting = delWorker(w.service.waiting, w)
	}

	w.broker.Waiting = delWorker(w.broker.Waiting, w)
	delete(w.broker.workers, w.idString)
}

// Send formats and sends a command to a worker using MDP v0.2 format (no empty frames).
// The caller may also provide a command option, and a message payload.
func (w *brokerWorker) Send(command, option string, msg []string) (err error) {
	n := 3
	if option != "" {
		n++
	}
	m := make([]string, n, n+len(msg))
	m = append(m, msg...)

	// stack protocol envelope to start of message (MDP v0.2 - no empty frame)
	if option != "" {
		m[3] = option
	}
	m[2] = command
	m[1] = MdpwWorker

	// stack routing envelope to start of message
	m[0] = w.identity

	log.WithFields(log.Fields{
		"command": MdpsCommands[command],
		"worker":  m,
	}).Trace("sending message")
	snd := stringArrayToByte2D(m)
	err = w.broker.Socket.SendMessage(snd)

	return
}

// Waiting checks if a worker is expecting work.
func (w *brokerWorker) Waiting() {
	// queue to broker and service waiting lists
	w.broker.Waiting = append(w.broker.Waiting, w)
	w.service.waiting = append(w.service.waiting, w)
	w.expiry = time.Now().Add(HeartbeatExpiry)
	w.service.Dispatch([]string{})
}
