package service

import (
	"encoding/json"
	"errors"

	"github.com/geoffjay/plantd/core/mdp"

	log "github.com/sirupsen/logrus"
)

type Connection interface {
	Close() error
	Send(service string, request ...string) (err error)
	Recv() (reply []string, err error)
}

type Client struct {
	conn Connection
}

// Establish a connection using the ZeroMQ API device.
func NewClient(endpoint string) (c *Client, err error) {
	conn, err := mdp.NewClient(endpoint)
	if err != nil {
		return nil, err
	}

	c = &Client{conn}

	return
}

// Close the connection to the ZeroMQ API device.
func (c *Client) Close() error {
	log.Debug("closing client connection")
	return c.conn.Close()
}

func (c *Client) sendMessage(id, message string, in interface{}, out interface{}) error {
	req := make([]string, 2)
	req[0] = message

	// Serialize message body to send
	bytes, err := json.Marshal(in)
	if err != nil {
		return err
	}

	// Send the message
	req[1] = string(bytes)
	_ = c.conn.Send(id, req...)
	// Wait for a reply
	reply, err := c.conn.Recv()
	if err != nil {
		return err
	}

	// Validate response
	if len(reply) == 0 {
		return errors.New("didn't receive expected response")
	}

	idx := 0
	if len(reply) > 2 && reply[idx] == "" {
		idx = 2
	}

	log.Debugf("reply: %+v\n", reply)

	// Deserialize reply into a response
	err = json.Unmarshal([]byte(reply[idx]), out)
	return err
}

type RawRequest map[string]interface{}
type RawResponse map[string]interface{}

func (c *Client) SendRawRequest(id, requestType string, request *RawRequest) (response RawResponse, err error) {
	response = make(RawResponse)
	err = c.sendMessage(id, requestType, request, &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}
