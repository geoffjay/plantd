package mdp

// Majordomo Protocol Worker API.
// Implements the MDP/Worker spec at http://rfc.zeromq.org/spec:7.

import (
	"fmt"
	"runtime"
	"time"

	"github.com/geoffjay/plantd/core/util"

	log "github.com/sirupsen/logrus"
	czmq "github.com/zeromq/goczmq/v4"
)

// Worker defines a single MDP worker instance.
type Worker struct {
	broker  string
	service string
	worker  *czmq.Sock // Socket to broker
	poller  *czmq.Poller

	// Heartbeat management
	heartbeatAt time.Time     // When to send HEARTBEAT
	liveness    int           // How many attempts left
	heartbeat   time.Duration // Heartbeat delay, msecs
	reconnect   time.Duration // Reconnect delay, msecs

	expectReply bool   // False only at start
	replyTo     string // Return identity, if any

	shutdown bool
}

// WorkerResponseStream represents a streaming response handler for workers
type WorkerResponseStream struct {
	worker  *Worker
	client  string
	service string
}

// SendPartial sends a partial response to the client
func (rs *WorkerResponseStream) SendPartial(data []string) error {
	if rs.client == "" {
		return fmt.Errorf("no client address available for response")
	}

	// Create message with client address
	m := make([]string, 1, 1+len(data))
	m = append(m, data...)
	m[0] = rs.client

	return rs.worker.SendToBroker(MdpwPartial, "", m)
}

// SendFinal sends the final response to the client and closes the stream
func (rs *WorkerResponseStream) SendFinal(data []string) error {
	if rs.client == "" {
		return fmt.Errorf("no client address available for response")
	}

	// Create message with client address
	m := make([]string, 1, 1+len(data))
	m = append(m, data...)
	m[0] = rs.client

	return rs.worker.SendToBroker(MdpwFinal, "", m)
}

// NewWorker creates a new instance of the worker class.
func NewWorker(broker, service string) (w *Worker, err error) {
	w = &Worker{
		broker:    broker,
		service:   service,
		heartbeat: 2500 * time.Millisecond,
		reconnect: 2500 * time.Millisecond,
		shutdown:  false,
	}

	err = w.ConnectToBroker()
	runtime.SetFinalizer(w, (*Worker).Close)

	return
}

// SendToBroker sends a message to the broker using MDP v0.2 format (no empty frames).
func (w *Worker) SendToBroker(command string, option string, msg []string) (err error) {
	n := 3 // Always include empty delimiter frame
	if option != "" {
		n++
	}
	m := make([]string, n, n+len(msg))
	m = append(m, msg...)

	// Stack protocol envelope to start of message (MDP v0.2 with empty delimiter)
	if option != "" {
		m[3] = option
	}
	m[2] = command
	m[1] = MdpwWorker
	m[0] = "" // Empty delimiter frame for DEALER socket routing

	// Validate the message before sending (worker-to-broker message format)
	// Skip validation for now since we're sending with empty delimiter frame
	// The broker will validate after processing the routing frames

	err = w.worker.SendMessage(stringArrayToByte2D(m))
	if err != nil {
		log.WithFields(log.Fields{
			"command": command,
			"option":  option,
			"error":   err,
		}).Error("failed to send message to broker")
	} else {
		log.WithFields(log.Fields{
			"command": command,
			"option":  option,
			"frames":  len(m),
		}).Debug("sent message to broker")
	}
	return
}

// ConnectToBroker connects or reconnects to the broker.
func (w *Worker) ConnectToBroker() (err error) {
	w.Close()

	if w.worker, err = czmq.NewDealer(w.broker); err != nil {
		log.WithFields(log.Fields{"error": err}).Error("failed to create dealer")
	}
	if err = w.worker.Connect(w.broker); err != nil {
		log.WithFields(log.Fields{"error": err}).Error("failed to connect to broker")
		return
	}
	if w.poller, err = czmq.NewPoller(); err != nil {
		log.WithFields(log.Fields{"error": err}).Error("failed to create socket poller")
		return
	}
	if err = w.poller.Add(w.worker); err != nil {
		log.WithFields(log.Fields{"error": err}).Error("failed to add worker socket to poller")
		return
	}

	// Register service with broker
	if err = w.SendToBroker(MdpwReady, w.service, []string{}); err != nil {
		log.WithFields(log.Fields{"error": err}).Error("failed to send ready message to broker")
		return
	}

	// If liveness hits zero, queue is considered disconnected
	w.liveness = HeartbeatLiveness
	w.heartbeatAt = time.Now().Add(w.heartbeat)

	log.WithFields(log.Fields{
		"broker":  w.broker,
		"service": w.service,
	}).Info("worker connected to broker")

	return
}

// Shutdown attempts to bail on execution after the poller timeout.
func (w *Worker) Shutdown() {
	w.shutdown = true
	time.Sleep(w.heartbeat)
}

// Terminated is `true` when a shutdown was requested.
func (w *Worker) Terminated() bool {
	return w.shutdown
}

// Close the worker socket.
func (w *Worker) Close() {
	if w.worker != nil {
		w.worker.Destroy()
		w.worker = nil
	}
}

// SetHeartbeat sets the heartbeat delay.
func (w *Worker) SetHeartbeat(heartbeat time.Duration) {
	w.heartbeat = heartbeat
}

// SetReconnect sets the reconnection delay.
func (w *Worker) SetReconnect(reconnect time.Duration) {
	w.reconnect = reconnect
}

// Reply sends a simple reply (backward compatible - sends FINAL response)
func (w *Worker) Reply(reply []string) error {
	if w.replyTo == "" {
		return fmt.Errorf("no recipient provided")
	}

	m := make([]string, 1, 1+len(reply))
	m = append(m, reply...)
	m[0] = w.replyTo

	return w.SendToBroker(MdpwFinal, "", m)
}

// GetResponseStream returns a streaming response handler for the current request
func (w *Worker) GetResponseStream() *WorkerResponseStream {
	return &WorkerResponseStream{
		worker:  w,
		client:  w.replyTo,
		service: w.service,
	}
}

// Recv sends a reply, if any, to broker and waits for the next request.
// Updated for MDP v0.2 protocol with PARTIAL/FINAL support
func (w *Worker) Recv(reply []string) (msg []string, err error) { //nolint:cyclop
	// format and send the reply if we were provided one
	if len(reply) == 0 && w.expectReply {
		log.Trace("received reply, unhandled")
	}

	if len(reply) > 0 {
		if err := w.Reply(reply); err != nil {
			log.WithError(err).Error("failed to send reply")
			return nil, err
		}
	}

	w.expectReply = true

	for {
		socket, perr := w.poller.Wait(int(w.heartbeat / 1e6))
		if perr != nil {
			log.WithFields(
				log.Fields{"err": perr},
			).Error(
				"an error occurred while the worker was receiving data",
			)
			break
		}

		if w.shutdown {
			break
		}

		if socket == nil { //nolint:nestif
			log.WithFields(log.Fields{
				"timeout (ms)": int(HeartbeatInterval) / 1e6,
			}).Tracef("no messages received on worker socket for the timeout duration")
			w.liveness--
			if w.liveness == 0 {
				time.Sleep(w.reconnect)
				if err = w.ConnectToBroker(); err != nil {
					log.WithFields(log.Fields{
						"err": err,
					}).Error("worker failed to connect to broker")
				}
			}
		} else {
			recv, _ := socket.RecvMessage()
			recvMsg := byte2DToStringArray(recv)

			if len(recvMsg) > 0 {
				w.liveness = HeartbeatLiveness

				// Validate message format using robust validation
				if err := ValidateWorkerMessage(recvMsg); err != nil {
					log.WithError(err).Error("received invalid worker message")
					continue // Skip invalid messages and continue processing
				}

				// MDP v0.2 frame format (no empty frames)
				command := recvMsg[1]
				msg = recvMsg[2:]

				switch command {
				case MdpwRequest:
					log.WithFields(log.Fields{
						"command": command,
						"msg":     msg,
					}).Debug("received request")
					// we should pop and save as many addresses as there are
					// up to a null part, but for now, just save one...
					w.replyTo, msg = util.Unwrap(msg)
					// here is where we actually have a message to process; we
					// return it to the caller application:
					return
				case MdpwHeartbeat:
					// do nothing for heartbeats
					log.Trace("worker received a heartbeat command")
				case MdpwDisconnect:
					if err = w.ConnectToBroker(); err != nil {
						log.WithFields(log.Fields{
							"err": err,
						}).Error("worker failed to connect to broker")
					}
					log.Debug("worker received a disconnection command")
				default:
					log.WithField("command", command).Warn("received unknown command")
				}
			} else { // len(RecvMsg) == 0
				log.WithFields(log.Fields{
					"timeout (ms)": int(HeartbeatInterval) / 1e6,
				}).Tracef("empty message received on worker socket")
				w.liveness--
				if w.liveness == 0 {
					time.Sleep(w.reconnect)
					if err = w.ConnectToBroker(); err != nil {
						log.WithFields(log.Fields{
							"err": err,
						}).Error("worker failed to connect to broker")
					}
				}
			}
		}

		// send HEARTBEAT if it's time
		if time.Now().After(w.heartbeatAt) {
			if err = w.SendToBroker(MdpwHeartbeat, "", []string{}); err != nil {
				log.WithFields(log.Fields{
					"err": err,
				}).Error("worker failed to send heartbeat to broker")
			}
			w.heartbeatAt = time.Now().Add(w.heartbeat)
		}
	}

	log.Debug("worker recv completed")

	return
}
