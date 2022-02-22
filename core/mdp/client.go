package mdp

// Majordomo Protocol Client API.
// Implements the MDP/Worker spec at http://rfc.zeromq.org/spec:7.

import (
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

// NewClient creates a new instance of an MDP client.
func NewClient(broker string) (c *Client, err error) {
	c = &Client{
		broker:  broker,
		timeout: 2500,
	}

	err = c.ConnectToBroker()
	runtime.SetFinalizer(c, (*Client).Close)

	return
}

// Close the client socket.
func (c *Client) Close() (err error) {
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
	_ = c.Close()
	if c.client, err = czmq.NewDealer(c.broker); err != nil {
		_ = c.Close()
		return
	}
	if c.poller, err = czmq.NewPoller(); err != nil {
		_ = c.Close()
		return
	}
	if err = c.poller.Add(c.client); err != nil {
		c.poller.Destroy()
		_ = c.Close()
		return
	}
	if err = c.client.Connect(c.broker); err != nil {
		c.poller.Destroy()
		_ = c.Close()
		return
	}

	return
}

// SetTimeout requests the timeout.
func (c *Client) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}

// Send just sends one message, without waiting for a reply. Since we're using
// a DEALER socket we have to send an empty frame at the start, to create the
// same envelope that the REQ socket would normally make.
func (c *Client) Send(service string, request ...string) (err error) {
	// Prefix request with protocol frames
	// Frame 0: empty (REQ emulation)
	// Frame 1: "MDPCxy" (six bytes, MDP/Client x.y)
	// Frame 2: Service name (printable string)

	req := make([]string, 3, len(request)+3)
	req = append(req, request...)
	req[2] = service
	req[1] = MdpcClient
	req[0] = ""
	err = c.client.SendMessage(stringArrayToByte2D(req))

	return
}

// Recv waits for a reply message and returns that to the caller. Returns the
// reply message or NULL if there was no reply. Does not attempt to recover
// from a broker failure, this is not possible without storing all unanswered
// requests and resending them all.
//nolint: funlen, nestif
func (c *Client) Recv() (msg []string, err error) {
	// poll socket for a reply, with timeout
	socket, perr := c.poller.Wait(int(c.timeout))
	if perr != nil {
		log.WithFields(log.Fields{
			"err": perr,
		}).Error("client failure while socket poller was waiting")
		return
	}
	if socket == nil {
		// log in the client in warn and not trace because it expects a response
		log.WithFields(log.Fields{
			"timeout (ms)": int(c.timeout),
		}).Warn("no messages received on client socket for the timeout duration")
		return
	}

	recv, _ := socket.RecvMessage()
	recvMsg := byte2DToStringArray(recv)

	// if we got a reply, process it
	if len(recvMsg) > 0 {
		// don't try to handle errors, just assert noisily
		if len(recvMsg) < 4 {
			log.WithFields(log.Fields{
				"expected": 4,
				"received": len(recvMsg),
			}).Warn("message received had less than the required number of frames")
		} else {
			// var empty string
			// empty, msg = util.PopStr(msg)
			// if empty != "" {
			if recvMsg[0] != "" {
				log.WithFields(log.Fields{
					"received": recvMsg[0],
				}).Warn("message frame didn't contain expected value (empty)")
			}

			// var header string
			// header, msg = util.PopStr(msg)
			// if header != MDPC_CLIENT {
			if recvMsg[1] != MdpcClient {
				log.WithFields(log.Fields{
					"expected": MdpcClient,
					"received": recvMsg[1],
				}).Warn("message frame didn't contain expected value")
			}

			// FIXME: this fails when len(msg) < 4
			service := recvMsg[2]
			msg = recvMsg[3:]
			// service, msg = util.PopStr(msg)
			log.WithFields(log.Fields{"service": service, "msg": msg}).Debug("received message")
		}

		// if this was reached the request was successful
		return
	}

	// FIXME: why freak out on timeout?
	err = errPermanent
	log.Error(err.Error())
	msg = []string{"timeout error"}

	return
}
