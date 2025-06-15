package mdp

// Majordomo Protocol Worker API.
// Implements the MDP/Worker spec at http://rfc.zeromq.org/spec:7.

import (
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

// SendToBroker sends a message to the broker.
func (w *Worker) SendToBroker(command string, option string, msg []string) (err error) {
	n := 3
	if option != "" {
		n++
	}
	m := make([]string, n, n+len(msg))
	m = append(m, msg...)

	// Stack protocol envelope to start of message
	if option != "" {
		m[3] = option
	}
	m[2] = command
	m[1] = MdpwWorker
	m[0] = ""

	err = w.worker.SendMessage(stringArrayToByte2D(m))
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
		log.WithFields(log.Fields{"error": err}).Error("failed to send read message to broker")
		return
	}

	// If liveness hits zero, queue is considered disconnected
	w.liveness = HeartbeatLiveness
	w.heartbeatAt = time.Now().Add(w.heartbeat)

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

// Recv send a reply, if any, to broker and waits for the next request.
// nolint: funlen, cyclop, nestif
func (w *Worker) Recv(reply []string) (msg []string, err error) {
	// format and send the reply if we were provided one
	if len(reply) == 0 && w.expectReply {
		log.Trace("received reply, unhandled")
	}

	if len(reply) > 0 {
		if w.replyTo == "" {
			// FIXME: do something?
			log.Trace("no recipient provided, unhandled")
		}

		m := make([]string, 2, 2+len(reply))
		m = append(m, reply...)
		m[0] = w.replyTo
		m[1] = ""
		err = w.SendToBroker(MdpwReply, "", m)
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

		if socket == nil {
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

				command := recvMsg[2]
				msg = recvMsg[3:]

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
