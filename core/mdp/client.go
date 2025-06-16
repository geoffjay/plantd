package mdp

// Majordomo Protocol Client API.
// Implements the MDP/Worker spec at http://rfc.zeromq.org/spec:7.

import (
	"fmt"
	"runtime"
	"time"

	log "github.com/sirupsen/logrus"
	czmq "github.com/zeromq/goczmq/v4"
)

// Client defines a single MDP client instance.
type Client struct {
	broker  string
	client  *czmq.Sock    // Socket to broker
	timeout time.Duration // Request timeout
	poller  *czmq.Poller
}

// ResponseStream represents a streaming response handler for MDP v0.2
type ResponseStream struct {
	client   *Client
	service  string
	finished bool
}

// Next waits for the next response in the stream
func (rs *ResponseStream) Next() (msg []string, final bool, err error) {
	if rs.finished {
		return nil, true, fmt.Errorf("stream already finished")
	}

	// poll socket for a reply, with timeout
	socket, perr := rs.client.poller.Wait(int(rs.client.timeout / time.Millisecond))
	if perr != nil {
		log.WithFields(log.Fields{
			"err": perr,
		}).Error("client failure while socket poller was waiting")
		return nil, false, perr
	}
	if socket == nil {
		log.WithFields(log.Fields{
			"timeout (ms)": int(rs.client.timeout / time.Millisecond),
		}).Warn("no messages received on client socket for the timeout duration")

		// Attempt to reconnect on timeout
		log.Info("attempting to reconnect to broker due to timeout")
		if reconnectErr := rs.client.ConnectToBroker(); reconnectErr != nil {
			log.WithFields(log.Fields{
				"err": reconnectErr,
			}).Error("failed to reconnect to broker after timeout")
			return nil, false, reconnectErr
		}
		log.Info("successfully reconnected to broker")
		return nil, false, fmt.Errorf("timeout - connection refreshed, please retry")
	}

	recv, _ := socket.RecvMessage()
	recvMsg := byte2DToStringArray(recv)

	// if we got a reply, process it
	if len(recvMsg) > 0 {
		// Validate message format using robust validation
		if err := ValidateClientMessage(recvMsg); err != nil {
			log.WithError(err).Error("received invalid client message")
			return nil, false, fmt.Errorf("invalid message format: %w", err)
		}

		command := recvMsg[1]
		service := recvMsg[2]
		data := recvMsg[3:]

		// Check if this is the final response
		if command == MdpcFinal {
			rs.finished = true
			final = true
		}

		log.WithFields(log.Fields{
			"service": service,
			"command": command,
			"data":    data,
			"final":   final,
		}).Debug("received streaming response")

		return data, final, nil
	}

	return nil, false, fmt.Errorf("empty response received")
}

// NewClient creates a new instance of an MDP client.
func NewClient(broker string) (c *Client, err error) {
	c = &Client{
		broker:  broker,
		timeout: 2500 * time.Millisecond,
	}

	err = c.ConnectToBroker()
	runtime.SetFinalizer(c, (*Client).Close)

	return
}

// Close the client socket.
func (c *Client) Close() (err error) {
	if c.poller != nil {
		c.poller.Destroy()
		c.poller = nil
	}
	if c.client != nil {
		c.client.Destroy()
		c.client = nil
	}
	return
}

// ConnectToBroker is used to connect or reconnect to a broker. In this
// asynchronous class we use a DEALER socket instead of a REQ socket; this lets
// us send any number of requests without waiting for a reply.
func (c *Client) ConnectToBroker() (err error) {
	log.WithFields(log.Fields{
		"broker": c.broker,
	}).Debug("connecting to broker")

	// Clean up existing connection
	_ = c.Close()

	// Create new DEALER socket
	if c.client, err = czmq.NewDealer(c.broker); err != nil {
		log.WithFields(log.Fields{
			"broker": c.broker,
			"error":  err,
		}).Error("failed to create DEALER socket")
		_ = c.Close()
		return
	}

	// Create new poller
	if c.poller, err = czmq.NewPoller(); err != nil {
		log.WithFields(log.Fields{
			"broker": c.broker,
			"error":  err,
		}).Error("failed to create poller")
		_ = c.Close()
		return
	}

	// Add socket to poller
	if err = c.poller.Add(c.client); err != nil {
		log.WithFields(log.Fields{
			"broker": c.broker,
			"error":  err,
		}).Error("failed to add socket to poller")
		c.poller.Destroy()
		_ = c.Close()
		return
	}

	// Connect to broker
	if err = c.client.Connect(c.broker); err != nil {
		log.WithFields(log.Fields{
			"broker": c.broker,
			"error":  err,
		}).Error("failed to connect socket to broker")
		c.poller.Destroy()
		_ = c.Close()
		return
	}

	log.WithFields(log.Fields{
		"broker": c.broker,
	}).Info("successfully connected to broker")

	return
}

// SetTimeout requests the timeout.
func (c *Client) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}

// Send sends a request message using MDP v0.2 protocol format with empty delimiter frame
func (c *Client) Send(service string, request ...string) (err error) {
	// MDP v0.2 format with empty delimiter frame (consistent with worker format)
	// Frame 0: "" (empty delimiter frame for DEALER socket routing)
	// Frame 1: "MDPC02" (six bytes, MDP/Client v0.2)
	// Frame 2: REQUEST command
	// Frame 3: Service name (printable string)
	// Frame 4+: Request body

	req := make([]string, 4, len(request)+4)
	req = append(req, request...)
	req[3] = service
	req[2] = MdpcRequest
	req[1] = MdpcClient
	req[0] = "" // Empty delimiter frame for DEALER socket routing

	// Note: ValidateClientRequestMessage expects the format without empty delimiter
	// so we validate frames 1-3 (skipping the empty delimiter at frame 0)
	if err := ValidateClientRequestMessage(req[1:]); err != nil {
		log.WithError(err).Error("invalid client request message format")
		return fmt.Errorf("invalid request format: %w", err)
	}

	err = c.client.SendMessage(stringArrayToByte2D(req))
	if err != nil {
		log.WithFields(log.Fields{
			"service": service,
			"error":   err,
		}).Error("failed to send request")
	} else {
		log.WithFields(log.Fields{
			"service": service,
			"frames":  len(req),
		}).Debug("sent request")
	}

	return
}

// Recv waits for a reply message and returns that to the caller. Returns the
// reply message or NULL if there was no reply. This method handles both PARTIAL
// and FINAL responses but only returns the FINAL response for backward compatibility.
// Use RecvStream() for full streaming support.
func (c *Client) Recv() (msg []string, err error) {
	// poll socket for a reply, with timeout
	socket, perr := c.poller.Wait(int(c.timeout / time.Millisecond))
	if perr != nil {
		log.WithFields(log.Fields{
			"err": perr,
		}).Error("client failure while socket poller was waiting")
		return
	}
	if socket == nil {
		// log in the client in warn and not trace because it expects a response
		log.WithFields(log.Fields{
			"timeout (ms)": int(c.timeout / time.Millisecond),
		}).Warn("no messages received on client socket for the timeout duration")

		// Attempt to reconnect on timeout - this handles stale connections
		log.Info("attempting to reconnect to broker due to timeout")
		if reconnectErr := c.ConnectToBroker(); reconnectErr != nil {
			log.WithFields(log.Fields{
				"err": reconnectErr,
			}).Error("failed to reconnect to broker after timeout")
			err = reconnectErr
			return
		}
		log.Info("successfully reconnected to broker")

		// Return timeout error - client should retry the request
		err = errPermanent
		msg = []string{"timeout error - connection refreshed, please retry"}
		return
	}

	recv, _ := socket.RecvMessage()
	recvMsg := byte2DToStringArray(recv)

	// if we got a reply, process it
	if len(recvMsg) > 0 {
		// Validate message format using robust validation
		if err := ValidateClientMessage(recvMsg); err != nil {
			log.WithError(err).Error("received invalid client message")
			return nil, fmt.Errorf("invalid message format: %w", err)
		}

		command := recvMsg[1]
		service := recvMsg[2]
		data := recvMsg[3:]

		// For backward compatibility, wait for FINAL response
		// If this is PARTIAL, keep reading until FINAL
		if command == MdpcPartial {
			log.WithFields(log.Fields{
				"service":      service,
				"partial_data": data,
			}).Debug("received partial response, waiting for final")

			// Continue waiting for FINAL response
			return c.Recv()
		}

		log.WithFields(log.Fields{
			"service": service,
			"msg":     data,
		}).Debug("received final response")

		return data, nil
	}

	// FIXME: why freak out on timeout?
	err = errPermanent
	log.Error(err.Error())
	msg = []string{"timeout error"}

	return
}

// RecvStream returns a ResponseStream for handling streaming responses with PARTIAL/FINAL support
func (c *Client) RecvStream(service string) *ResponseStream {
	return &ResponseStream{
		client:   c,
		service:  service,
		finished: false,
	}
}

// SendAndRecvStream sends a request and returns a stream for receiving responses
func (c *Client) SendAndRecvStream(service string, request ...string) (*ResponseStream, error) {
	if err := c.Send(service, request...); err != nil {
		return nil, err
	}
	return c.RecvStream(service), nil
}
